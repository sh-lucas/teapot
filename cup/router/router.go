package router

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sh-lucas/mug/pkg/spout"
	"github.com/sh-lucas/teapot/handlers/logs"
)

func Route(addr string) {
	router := chi.NewRouter()
	router.Use(logs.CORSMiddleware)

	// handlers:
	router.HandleFunc("POST /log", logs.SaveLog)
	router.HandleFunc("GET /logs/{clientName}", logs.GetLog)

	// Swagger docs
	spout.ServeDocs(router)
	fmt.Printf("\033[36mSwagger UI available at http://localhost:%s/docs\033[0m\n", addr)

	fmt.Printf("\033[32mStarting server on :%s\033[0m\n", addr)
	if err := http.ListenAndServe(":"+addr, router); err != nil {
		log.Fatalf("❌ Could not start server: %s\n", err)
	}
}
