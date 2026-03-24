package main

import (
	"log"
	"net/http"

	"antigravity/internal/api"
	"antigravity/internal/indexer"
	"antigravity/internal/search"
	"antigravity/internal/state"
)

func main() {
	log.Println("=== ANTIGRAVITY Search Engine ===")
	log.Println("Starting on http://localhost:3600")

	// Initialize subsystems
	st := state.New()
	idx := indexer.New()
	searchEngine := search.New(idx)

	storage, err := indexer.NewStorage("data/storage/p.data", "data/storage/pages.jsonl")
	if err != nil {
		log.Fatalf("failed to initialize storage: %v", err)
	}
	defer storage.Close()

	handlers := &api.Handlers{
		Indexer: idx,
		Search:  searchEngine,
		State:   st,
		Storage: storage,
	}

	mux := http.NewServeMux()
	api.RegisterRoutes(mux, handlers)

	log.Fatal(http.ListenAndServe(":3600", mux))
}
