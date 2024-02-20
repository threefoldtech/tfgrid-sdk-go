package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

const query = `query TwinsQuery {
  items: twinsConnection(first: %d, orderBy: id_ASC%s) {
    pageInfo {
  	  endCursor
  	  hasNextPage
    }
  }
  twins(limit: %d, offset: %s, orderBy: id_ASC) {
    accountID
    publicKey
    twinID
    relay
  }
}`

const (
	twinsPerPage     = 500
	requestsInterval = 3 * time.Second
)

type Twin struct {
	ID        uint      `json:"id"`
	AccountID string    `json:"account"`
	Relay     []string  `json:"relay"`
	PK        JSONArray `json:"pk"`
}

// only exists because json marshals []byte to a string instead of json array
type JSONArray []byte

func (u JSONArray) MarshalJSON() ([]byte, error) {
	if u == nil || len(u) == 0 {
		return []byte("null"), nil
	}

	var buf bytes.Buffer
	_, _ = buf.WriteRune('[')
	for i, b := range u {
		if i != 0 {
			_, _ = buf.WriteRune(',')
		}
		_, _ = buf.WriteString(strconv.FormatUint(uint64(b), 10))
	}
	_, _ = buf.WriteRune(']')

	return buf.Bytes(), nil

}

func warmTwins(pool *redis.Pool, graphql string) error {
	offset := "0"

	for {
		afterCondition := ""
		if offset != "0" && offset != "" {
			afterCondition = fmt.Sprintf(`, after:"%s"`, offset)
		}
		body := fmt.Sprintf(
			query,
			twinsPerPage,
			afterCondition,
			twinsPerPage,
			offset,
		)
		pagination, gtwins, err := queryGraphql(graphql, body)
		if err != nil {
			return err
		}
		twins, err := graphqlTwinsToRelayTwins(gtwins)
		if err != nil {
			return err
		}
		offset = pagination.PageInfo.EndCursor

		err = writeTwins(pool, twins)
		if err != nil {
			return err
		}
		if !pagination.PageInfo.HasNextPage {
			break
		}
		time.Sleep(requestsInterval)
	}

	return nil
}

func queryGraphql(graphql, body string) (paginationData, []graphqlTwin, error) {
	bodyBytes, err := json.Marshal(map[string]interface{}{"query": body})
	if err != nil {
		return paginationData{}, nil, err
	}
	reader := bytes.NewReader(bodyBytes)
	resp, err := http.Post(graphql, "application/json", reader)
	if err != nil {
		return paginationData{}, nil, err
	}
	defer resp.Body.Close()
	r, err := io.ReadAll(resp.Body)
	if err != nil {
		return paginationData{}, nil, err
	}
	respBody := map[string]interface{}{}
	err = json.Unmarshal(r, &respBody)
	if err != nil {
		return paginationData{}, nil, err
	}
	var gTwins []graphqlTwin
	data, ok := respBody["data"].(map[string]interface{})
	if !ok {
		return paginationData{}, nil, fmt.Errorf("got unexpected format %s", string(r))
	}

	twins := data["twins"]
	dataBytes, err := json.Marshal(twins)
	if err != nil {
		return paginationData{}, nil, err
	}
	if err := json.Unmarshal(dataBytes, &gTwins); err != nil {
		return paginationData{}, nil, err
	}
	items := data["items"]
	itemsBytes, err := json.Marshal(items)
	if err != nil {
		return paginationData{}, nil, err
	}
	var pdata paginationData

	if err := json.Unmarshal(itemsBytes, &pdata); err != nil {
		return paginationData{}, nil, err
	}
	return pdata, gTwins, nil
}

func graphqlTwinsToRelayTwins(gTwins []graphqlTwin) ([]Twin, error) {
	twins := make([]Twin, 0, len(gTwins))
	for _, gTwin := range gTwins {
		twin := Twin{
			ID:        gTwin.ID,
			AccountID: gTwin.AccountID,
		}
		if gTwin.Relay != nil && len(*gTwin.Relay) != 0 {
			twin.Relay = strings.Split(*gTwin.Relay, "_")
		}
		if gTwin.PK != nil && len(*gTwin.PK) != 0 {
			pk, err := hex.DecodeString(strings.TrimPrefix(*gTwin.PK, "0x"))
			if err != nil {
				return nil, err
			}
			twin.PK = pk
		}
		twins = append(twins, twin)
	}
	return twins, nil
}

type paginationData struct {
	Count    int      `json:"count"`
	PageInfo PageInfo `json:"pageInfo"`
}
type PageInfo struct {
	EndCursor   string `json:"endCursor"`
	HasNextPage bool   `json:"hasNextPage"`
}

type graphqlTwin struct {
	ID        uint    `json:"twinID"`
	AccountID string  `json:"accountID"`
	Relay     *string `json:"relay"`
	PK        *string `json:"publicKey"`
}
