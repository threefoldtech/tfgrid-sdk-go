package main

import (
	"encoding/json"
	"fmt"

	"github.com/gomodule/redigo/redis"
)

const twinTTL = 3600 // an hour

func writeTwins(conn redis.Conn, twins []Twin) error {
	for _, twin := range twins {
		val, err := json.Marshal(twin)
		if err != nil {
			return err
		}
		key := fmt.Sprintf("twin.%d", twin.ID)
		_, err = conn.Do("SET", key, val, "EX", twinTTL)
		if err != nil {
			return err
		}
	}
	return nil
}
