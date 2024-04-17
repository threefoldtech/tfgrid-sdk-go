package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
)

// set at build time
var commit string
var version string

func main() {
	var graphql string
	var interval uint64
	var redisURL string
	var redisPassword string
	var versionFlag bool
	flag.StringVar(&graphql, "graphql", "https://graphql.grid.tf/graphql", "graphql url")
	flag.Uint64Var(&interval, "interval", 10, "cache warming interval")
	flag.StringVar(&redisURL, "redis-url", "redis://localhost:6379", "redis url")
	flag.StringVar(&redisPassword, "redis-password", "", "redis password")
	flag.BoolVar(&versionFlag, "version", false, "print version and exit")

	flag.Parse()
	if versionFlag {
		fmt.Println(commit)
		fmt.Println(version)
		os.Exit(0)
	}

	var opts []redis.DialOption
	if redisPassword != "" {
		opts = append(opts, redis.DialPassword(redisPassword))
	}
	pool := redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(redisURL, opts...)
		},
		MaxIdle:   10,
		MaxActive: 20,
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) > 10*time.Second {
				_, err := c.Do("PING")
				return err
			}
			return nil
		},
		Wait: true,
	}

	run(&pool, graphql, time.Duration(interval)*time.Minute)

}

func run(pool *redis.Pool, graphql string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	log.Println("warmer started")
	for {
		select {
		case <-ticker.C:
			err := warmTwins(pool, graphql)
			if err != nil {
				log.Printf("failed to warm twins: %s", err.Error())
			}
		}
	}
}
