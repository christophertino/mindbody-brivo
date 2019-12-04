// Redis DB Connection Utility
//
// Copyright 2019 Christopher Tino. All rights reserved.

package mindbodybrivo

import (
	"github.com/gomodule/redigo/redis"
)

// NewPool creates a new Redis connection Pool
func NewPool(redisURL string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:   5,  // Maximum number of idle connections in the pool
		MaxActive: 10, // Maximum number of connections allocated by the pool at a given time
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(redisURL)
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}
}

// Get executes the Redis GET command
func Get(key string, c redis.Conn) (string, error) {
	value, err := redis.String(c.Do("GET", key))
	if err != nil {
		return "", err
	}
	return value, nil
}

// Set executes the Redis SET command
func Set(key string, value string, c redis.Conn) error {
	_, err := c.Do("SET", key, value)
	if err != nil {
		return err
	}
	return nil
}