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

	conn, err := redis.DialURL(redisURL, opts...)
	if err != nil {
		log.Fatalf("failed to connect to redis server: %s", err.Error())
	}

	run(conn, graphql, time.Duration(interval)*time.Minute)

}

func run(conn redis.Conn, graphql string, interval time.Duration) {
	for {
		select {
		case <-time.After(interval):
			go func() {
				twins, err := getTwins(graphql)
				if err != nil {
					log.Printf("failed to get twins: %s", err.Error())
				}
				err = writeTwins(conn, twins)
				if err != nil {
					log.Printf("failed to update cache: %s", err.Error())
				}
			}()
		}
	}
}
