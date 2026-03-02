package main

import (
	"log"
	"os"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
		os.Exit(1)
	}

	srv := NewServer(cfg, "")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
