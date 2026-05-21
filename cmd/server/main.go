package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/wevibe-network/wevibe-social-graph/internal/server"
	"github.com/wevibe-network/wevibe-social-graph/internal/store"
)

func main() {
	dbPath := getenv("SOCIAL_GRAPH_DB_PATH", "/data/social-graph.db")
	port := getenvInt("SOCIAL_GRAPH_PORT", 4470)

	profileStore, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("social-graph: initialize store: %v", err)
	}
	defer profileStore.Close()

	handler := server.New(profileStore)
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("wevibe-social-graph listening on %s (db=%s)", httpServer.Addr, dbPath)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("social-graph: server failed: %v", err)
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
