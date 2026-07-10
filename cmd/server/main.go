package main

import (
	"log"
	"net/http"

	"github.com/johnnyc20/myproject/internal/api"
	"github.com/johnnyc20/myproject/internal/config"
	"github.com/johnnyc20/myproject/internal/store"
)

func main() {
	cfg := config.Load()

	s, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer s.Close()

	a := api.New(s)

	log.Printf("listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, a.Routes()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
