package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Declare the custom counters (Keeping them package-visible)
var (
	cacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "weather_cache_hits_total",
			Help: "Total number of successful Redis cache reads.",
		},
	)
	cacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "weather_cache_misses_total",
			Help: "Total number of cache misses requiring an external API call.",
		},
	)
)

func init() {
	// Register counters automatically during application startup
	prometheus.MustRegister(cacheHits)
	prometheus.MustRegister(cacheMisses)
}