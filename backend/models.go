package main

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type weatherData struct {
	Name     string `json:"name"`
	Timezone int    `json:"timezone"` // Offset in seconds from UTC
	Dt       int64  `json:"dt"`       // Current Unix timestamp
	Sys      struct {
		Country string `json:"country"`
	} `json:"sys"`
	Main struct {
		Kelvin   float64 `json:"temp"`
		Humidity int     `json:"humidity"`
	} `json:"main"`

	Wind struct {
		Speed float64 `json:"speed"`
		Deg   float64 `json:"deg"`
	} `json:"wind"`

	Weather []struct {
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
}

type WeatherCache struct {
	client *redis.Client
	ctx    context.Context
}