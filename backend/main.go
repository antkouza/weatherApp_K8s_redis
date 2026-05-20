package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// const allowedOrigin = "http://localhost:4200" // allow frontend requests
const allowedOrigin = "*"

var cache *WeatherCache

func main() {
	godotenv.Load()
	port := ":8080"

//////////redis
	// Read environment variable injected by K8s or fallback to localhost for standard testing
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" 
	}

	// Initialize our refactored Redis cache layer
	cache = NewWeatherCache(redisAddr)
/////////////
	fmt.Println("Server is running on port" + port)

	http.HandleFunc("/weather", weatherHandler)
	http.HandleFunc("/weather/", weatherHandler)

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

	if data, ok := cache.Get(cacheKey); ok {
		fmt.Printf("Cache FROM_CACHE: %s\n", cacheKey)
		w.Header().Set("X-Cache", "FROM_CACHE")
		json.NewEncoder(w).Encode(data)
		return
	}

	var wdata weatherData
	var err error
	if cityName != "" {
		wdata, err = fetchWeather(cityName)
	} else {
		wdata, err = fetchWeatherByCoordinates(lat, lon)
	}

	if err != nil || wdata.Name == "" {
		fmt.Printf("err : %s\n", cacheKey)
		http.Error(w, "Weather data not found", http.StatusNotFound)
		return
	}

	response := formatWeatherResponse(wdata)

	cache.Set(cacheKey, response, 1*time.Minute)

	w.Header().Set("X-Cache", "FROM_EXT_API")
	json.NewEncoder(w).Encode(response)
}
