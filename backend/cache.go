package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewWeatherCache initializes connection to the centralized Redis instance
func NewWeatherCache(redisAddr string) *WeatherCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,        // "redis-service:6379" inside K8s
		Password: "",               // No password set for local dev cluster
		DB:       0,                // Default database
	})

	return &WeatherCache{
		client: rdb,
		ctx:    context.Background(),
	}
}

// Get fetches data from Redis and unmarshals the JSON back into a Go map
func (c *WeatherCache) Get(city string) (map[string]interface{}, bool) {
	// Query Redis using the city name as the key
	val, err := c.client.Get(c.ctx, city).Result()
	if err == redis.Nil {
		// Key does not exist or has expired natively in Redis
		return nil, false
	} else if err != nil {
		fmt.Printf("Redis Error reading key %s: %v\n", city, err)
		return nil, false
	}

	// Unmarshal the cached JSON string back into map[string]interface{}
	var data map[string]interface{}
	err = json.Unmarshal([]byte(val), &data)
	if err != nil {
		fmt.Printf("JSON Unmarshal Error for key %s: %v\n", city, err)
		return nil, false
	}

	return data, true
}

// Set marshals the weather map to JSON and saves it to Redis with an absolute expiration time
func (c *WeatherCache) Set(city string, value map[string]interface{}, ttl time.Duration) {
	// Serialize map structure into a JSON string
	jsonData, err := json.Marshal(value)
	if err != nil {
		fmt.Printf("JSON Marshal Error for key %s: %v\n", city, err)
		return
	}

	// Save to Redis. Redis automatically deletes this entry after the ttl duration
	err = c.client.Set(c.ctx, city, jsonData, ttl).Err()
	if err != nil {
		fmt.Printf("Redis Error saving key %s: %v\n", city, err)
	}
}