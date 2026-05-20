package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

func fetchWeather(city string) (weatherData, error) {
	escapeCity := url.QueryEscape(city)
	fmt.Printf("Debug: City: %s\n", city)

	apiKey := os.Getenv("WEATHER_API_KEY")
	fmt.Printf("Debug: apiKey: %s\n", apiKey)
	url := "http://api.openweathermap.org/data/2.5/weather?q=" + escapeCity + "&APPID=" + apiKey

	resp, err := http.Get(url)
	if err != nil {
		return weatherData{}, err
	}
	// Close the body when the function returns
	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return weatherData{}, err
	}

	fmt.Printf("Debug: City: %s\n Raw Response: %s\n", city, string(response))
	data := weatherData{}
	err = json.Unmarshal(response, &data)
	return data, err
}

func fetchWeatherByCoordinates(lat, lon string) (weatherData, error) {
	escapedLat := url.QueryEscape(lat)
	escapedLon := url.QueryEscape(lon)
	fmt.Printf("Debug: Coordinates: lat=%s lon=%s\n", lat, lon)

	apiKey := os.Getenv("WEATHER_API_KEY")
	fmt.Printf("Debug: apiKey: %s\n", apiKey)
	url := "http://api.openweathermap.org/data/2.5/weather?lat=" + escapedLat + "&lon=" + escapedLon + "&APPID=" + apiKey

	resp, err := http.Get(url)
	if err != nil {
		return weatherData{}, err
	}
	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return weatherData{}, err
	}

	fmt.Printf("Debug: Coordinates: lat=%s lon=%s\n Raw Response: %s\n", lat, lon, string(response))
	data := weatherData{}
	err = json.Unmarshal(response, &data)
	return data, err
}

func formatWeatherResponse(wdata weatherData) map[string]interface{} {
	tempC := wdata.Main.Kelvin - 273.15
	tempF := (tempC * 1.8) + 32

	locationTime := time.Unix(wdata.Dt+int64(wdata.Timezone), 0).UTC()

	condition := ""
	icon := ""
	if len(wdata.Weather) > 0 {
		condition = wdata.Weather[0].Description
		icon = wdata.Weather[0].Icon
	}

	return map[string]interface{}{
		"city":      wdata.Name,
		"country":   wdata.Sys.Country,
		"localTime": locationTime.Format("15:04"),
		"temperature": map[string]float64{
			"celsius":    tempC,
			"fahrenheit": tempF,
		},
		"humidity": wdata.Main.Humidity,
		"wind": map[string]float64{
			"speed": wdata.Wind.Speed,
			"deg":   wdata.Wind.Deg,
		},
		"condition": condition,
		"icon":      icon,
	}
}
