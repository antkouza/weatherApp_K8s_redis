package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// const allowedOrigin = "http://localhost:4200" // allow frontend requests
const allowedOrigin = "*"
const cacheFreshWindow = 1 * time.Minute // 1-minute freshness

var cache *WeatherCache

func main() {
	godotenv.Load()
	port := ":8080"

	// Read environment variable injected by K8s or fallback to localhost for standard testing
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" 
	}

	// Initialize Redis cache layer
	cache = NewWeatherCache(redisAddr)

	fmt.Println("Server is running on port" + port)

	http.HandleFunc("/weather", weatherHandler)
	http.HandleFunc("/weather/", weatherHandler)

	// Expose standard telemetry endpoint
	http.Handle("/metrics", promhttp.Handler())

	http.ListenAndServe(port, nil)
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Request: %s %s | Query: %s\n", r.Method, r.URL.Path, r.URL.RawQuery)

	w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Content-Type", "application/json")

	cityName := strings.ToLower(getCityName(r))
	query := r.URL.Query()
	lat := query.Get("lat")
	lon := query.Get("lon")

	var cacheKey string
	if cityName != "" {
		cacheKey = cityName
	} else if lat != "" && lon != "" {
		cacheKey = fmt.Sprintf("lat:%s&lon:%s", lat, lon)
	} else {
		http.Error(w, "City or lat/lon required", http.StatusBadRequest)
		return
	}

	// 1. Query cache. Returns data, foundInRedis, isStale
	if data, found, isStale := cache.Get(cacheKey); found {
		cacheHits.Inc()
		if isStale {
			fmt.Printf("SWR HIT: Serving stale data for '%s' instantly. Dispatching background refresh.\n", cacheKey)
			w.Header().Set("X-Cache", "STALE_REVALIDATING")

			// Fire off a background Goroutine to fetch fresh data.
			// This completely bypasses the user's wait time!
			go refreshCacheBackground(cacheKey, cityName, lat, lon)
		} else {
			fmt.Printf("CACHE HIT: Serving fresh data for '%s'.\n", cacheKey)
			w.Header().Set("X-Cache", "FROM_CACHE")
		}

		json.NewEncoder(w).Encode(data)
		return
	}

	// 2. ABSOLUTE CACHE MISS: Complete fallback if data is missing entirely from Redis
	cacheMisses.Inc()
	fmt.Printf("❄️ CACHE MISS: Complete fallback sync fetch for '%s'.\n", cacheKey)
	response, err := fetchAndSaveToCache(cacheKey, cityName, lat, lon)
	if err != nil {
		fmt.Printf("err : %s\n", cacheKey)
		http.Error(w, "Weather data not found", http.StatusNotFound)
		return
	}

	w.Header().Set("X-Cache", "FROM_EXT_API")
	json.NewEncoder(w).Encode(response)
}

// Detached async function executed entirely by the background Goroutine runtime
func refreshCacheBackground(cacheKey, cityName, lat, lon string) {
	_, err := fetchAndSaveToCache(cacheKey, cityName, lat, lon)
	if err != nil {
		fmt.Printf("SWR REVALIDATE failure for '%s': %v\n", cacheKey, err)
		return
	}
	fmt.Printf("SWR REVALIDATE success for '%s'. Redis refreshed.\n", cacheKey)
}

// Reusable orchestrator that handles external calling and cache saving
func fetchAndSaveToCache(cacheKey, cityName, lat, lon string) (map[string]interface{}, error) {
	var wdata weatherData
	var err error

	if cityName != "" {
		wdata, err = fetchWeather(cityName)
	} else {
		wdata, err = fetchWeatherByCoordinates(lat, lon)
	}

	if err != nil || wdata.Name == "" {
		return nil, fmt.Errorf("weather data data not found or invalid response")
	}

	response := formatWeatherResponse(wdata)

	// Save using our custom SWR setup (1-minute freshness window)
	cache.Set(cacheKey, response, cacheFreshWindow)

	return response, nil
}
