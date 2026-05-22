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
func (c *WeatherCache) Get(city string) (map[string]interface{}, bool, bool) {
	val, err := c.client.Get(c.ctx, city).Result()
	if err == redis.Nil {
		return nil, false, false
	} else if err != nil {
		fmt.Printf("Redis Error reading key %s: %v\n", city, err)
		return nil, false, false
	}

	// Unmarshal into our new SWR wrapper structure
	var wrapper swrWrapper
	err = json.Unmarshal([]byte(val), &wrapper)
	if err != nil {
		fmt.Printf("JSON Unmarshal Error for key %s: %v\n", city, err)
		return nil, false, false
	}

	// The data is stale if the current clock time has passed our StaleAt marker
	isStale := time.Now().After(wrapper.StaleAt)

	return wrapper.wrapperDataCheck(), true, isStale
}

// Helper method to safely return inner data map
func (w *swrWrapper) wrapperDataCheck() map[string]interface{} {
	return w.Data
}

// Set saves the data to Redis, calculating an extended TTL safety window
func (c *WeatherCache) Set(city string, value map[string]interface{}, freshDuration time.Duration) {
	now := time.Now()

	wrapper := swrWrapper{
		Data:    value,
		StaleAt: now.Add(freshDuration), // e.g., Mark stale after 1 minute
	}

	jsonData, err := json.Marshal(wrapper)
	if err != nil {
		fmt.Printf("JSON Marshal Error for key %s: %v\n", city, err)
		return
	}

	// CRITICAL: Keep data inside Redis 5x longer than the fresh window.
	// This ensures stale data is preserved so background routines can read it.
	redisTTL := freshDuration * 5

	err = c.client.Set(c.ctx, city, jsonData, redisTTL).Err()
	if err != nil {
		fmt.Printf("Redis Error saving key %s: %v\n", city, err)
	}
}