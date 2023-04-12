package direct

import (
	"fmt"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
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
	cache *cache.Cache
	sub   *substrate.Substrate
}

// NewTwinDB creates a new twinDBImpl instance, with a non expiring cache.
func NewTwinDB(sub *substrate.Substrate) TwinDB {
	return &twinDB{
		cache: cache.New(cache.NoExpiration, cache.NoExpiration),
		sub:   sub,
	}
}

// GetTwin gets Twin from cache if present. if not, gets it from substrate client and caches it.
func (t *twinDB) Get(id uint32) (Twin, error) {
	cachedValue, ok := t.cache.Get(fmt.Sprint(id))
	if ok {
		return cachedValue.(Twin), nil
	}

	substrateTwin, err := t.sub.GetTwin(id)
	if err != nil {
		return Twin{}, errors.Wrapf(err, "could net get twin with id %d", id)
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

	err = t.cache.Add(fmt.Sprint(id), twin, cache.DefaultExpiration)
	if err != nil {
		return Twin{}, errors.Wrapf(err, "could not set cache for twin with id %d", id)
	}

	return twin, nil
}

func (t *twinDB) GetByPk(pk []byte) (uint32, error) {
	return t.sub.GetTwinByPubKey(pk)
}
