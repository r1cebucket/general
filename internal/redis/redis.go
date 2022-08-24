package redis

import (
	"log"

	"github.com/gomodule/redigo/redis"
)

var pool *redis.Pool

func Init(host, port string) {
	pool = &redis.Pool{
		MaxIdle:     10,
		MaxActive:   0,
		IdleTimeout: 300,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", host+":"+port)
		},
	}

	conn := pool.Get()
	defer conn.Close()
	_, err := conn.Do("ping")
	if err != nil {
		log.Println("connect to redis faild:", err)
	}
}

func Do(commandName string, args ...interface{}) (interface{}, error) {
	conn := pool.Get()
	defer conn.Close()
	reply, err := conn.Do(commandName, args...)
	return reply, err
}

func String(reply interface{}, err error) (string, error) {
	str, err := redis.String(reply, err)
	return str, err
}
