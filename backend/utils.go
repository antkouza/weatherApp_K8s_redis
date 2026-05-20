package main

import (
	"net/http"
	"strings"
)

func getCityName(r *http.Request) string {
	cityName := r.URL.Query().Get("city")
	if cityName != "" {
		return cityName
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) > 1 { // /weather/London -> parts is ["weather", "London"]
		return parts[len(parts)-1]
	}
	return ""
}