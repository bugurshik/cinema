package main

import (
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                   string
	MonolithURL            string
	MoviesServiceURL       string
	EventsServiceURL       string
	GradualMigration       bool
	MoviesMigrationPercent int
}

var config Config

func loadConfig() {
	config = Config{
		Port:                   getEnv("PORT", "8000"),
		MonolithURL:            getEnv("MONOLITH_URL", "http://localhost:8080"),
		MoviesServiceURL:       getEnv("MOVIES_SERVICE_URL", "http://localhost:8081"),
		EventsServiceURL:       getEnv("EVENTS_SERVICE_URL", "http://localhost:8082"),
		GradualMigration:       getEnvAsBool("GRADUAL_MIGRATION", false),
		MoviesMigrationPercent: getEnvAsInt("MOVIES_MIGRATION_PERCENT", 50),
	}
}

func shouldMigrate(path string) bool {
	if !config.GradualMigration {
		return false
	}

	if strings.HasPrefix(path, "/api/movies") {
		random := rand.Intn(100)
		return random < config.MoviesMigrationPercent
	}

	return false
}

func NewProxy(targetURL string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	log.Printf("redirecting to: %s", targetURL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	return proxy, nil
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request for: %s", r.URL.Path)

	var targetProxy *httputil.ReverseProxy
	var err error

	switch {
	case shouldMigrate(r.URL.Path):
		if strings.HasPrefix(r.URL.Path, "/api/movies") {
			targetProxy, err = NewProxy(config.MoviesServiceURL)
		} else if strings.HasPrefix(r.URL.Path, "/api/events") {
			targetProxy, err = NewProxy(config.EventsServiceURL)
		}
	default:
		targetProxy, err = NewProxy(config.MonolithURL)
	}

	if err != nil {
		log.Printf("Error creating proxy: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	targetProxy.ServeHTTP(w, r)
}

func main() {
	loadConfig()

	http.HandleFunc("/", handleRequest)

	log.Printf("Starting proxy server on port %s", config.Port)
	log.Printf("Configuration: %+v", config)

	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsBool(name string, defaultVal bool) bool {
	valStr := getEnv(name, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsInt(name string, defaultVal int) int {
	valStr := getEnv(name, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}
