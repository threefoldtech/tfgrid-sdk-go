package peer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

var (
	errNoCache = fmt.Errorf("not cached")
)

// TwinDB is used to get Twin instances
type TwinDB interface {
	Get(id uint32) (Twin, error)
	GetByPk(pk []byte) (uint32, error)
}

// Twin is used to store a twin id and its public key
type Twin struct {
	ID        uint32
	PublicKey []byte
	Relay     *string
	E2EKey    []byte
}

type twinDB struct {
	subConn *substrate.Substrate
}

// NewTwinDB creates a new twinDBImpl instance, with a non expiring cache.
func NewTwinDB(subConn *substrate.Substrate) TwinDB {
	return &twinDB{
		subConn: subConn,
	}
}

// GetTwin gets Twin from cache if present. if not, gets it from substrate client and caches it.
func (t *twinDB) Get(id uint32) (Twin, error) {
	substrateTwin, err := t.subConn.GetTwin(id)
	if err != nil {
		return Twin{}, errors.Wrapf(err, "could not get twin with id %d", id)
	}

	var relay *string

	if substrateTwin.Relay.HasValue {
		relay = &substrateTwin.Relay.AsValue
	}

	_, PK := substrateTwin.Pk.Unwrap()
	twin := Twin{
		ID:        id,
		PublicKey: substrateTwin.Account.PublicKey(),
		Relay:     relay,
		E2EKey:    PK,
	}

	return twin, nil
}

func (t *twinDB) GetByPk(pk []byte) (uint32, error) {
	return t.subConn.GetTwinByPubKey(pk)
}

type inMemoryCache struct {
	cache map[uint32]Twin
	inner TwinDB
	m     sync.RWMutex
}

func newInMemoryCache(inner TwinDB) TwinDB {
	return &inMemoryCache{
		cache: make(map[uint32]Twin),
		inner: inner,
	}
}

func (m *inMemoryCache) Get(id uint32) (twin Twin, err error) {
	m.m.RLock()
	twin, ok := m.cache[id]
	m.m.RUnlock()
	if ok {
		return twin, nil
	}

	twin, err = m.inner.Get(id)
	if err != nil {
		return Twin{}, errors.Wrapf(err, "could not get twin with id %d", id)
	}

	m.m.Lock()
	m.cache[id] = twin
	m.m.Unlock()

	return twin, nil
}

func (m *inMemoryCache) GetByPk(pk []byte) (uint32, error) {
	return m.inner.GetByPk(pk)
}

type cachedTwin struct {
	Twin
	Timestamp uint64
}

type tmpCache struct {
	base  string
	ttl   uint64
	inner TwinDB
}

func newTmpCache(ttl uint64, inner TwinDB) (TwinDB, error) {
	path := filepath.Join(os.TempDir(), "rmb-cache")
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}

	return &tmpCache{
		base:  path,
		ttl:   ttl,
		inner: inner,
	}, nil
}

func (r *tmpCache) get(path string) (twin cachedTwin, err error) {
	data, err := os.ReadFile(path)

	if os.IsNotExist(err) {
		return twin, errNoCache
	} else if err != nil {
		return twin, err
	}

	err = json.Unmarshal(data, &twin)
	if err != nil {
		// we return an errNoCache so we don't
		// crash on file corruption
		return twin, errNoCache
	}

	age := uint64(time.Now().Unix()) - twin.Timestamp
	if age > r.ttl {
		log.Trace().Uint64("age", age).Msg("twin cache hit but expired")
		return twin, errNoCache
	}

	log.Trace().Msg("twin cache hit")
	return twin, nil
}

func (r *tmpCache) set(path string, twin Twin) error {
	data, err := json.Marshal(cachedTwin{
		Twin:      twin,
		Timestamp: uint64(time.Now().Unix()),
	})

	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (r *tmpCache) Get(id uint32) (twin Twin, err error) {
	path := filepath.Join(r.base, fmt.Sprint(id))

	cached, err := r.get(path)
	if err == errNoCache {
		twin, err = r.inner.Get(id)
		if err != nil {
			return twin, err
		}
		// set cache
		if err := r.set(path, twin); err != nil {
			log.Error().Err(err).Msg("failed to warm up cache")
		}
		return twin, nil
	} else if err != nil {
		return twin, err
	}

	return cached.Twin, nil
}

func (r *tmpCache) GetByPk(pk []byte) (uint32, error) {
	return r.inner.GetByPk(pk)
}
