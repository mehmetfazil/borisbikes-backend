package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mehmetfazil/borisbikes-backend/db"
	"github.com/mehmetfazil/borisbikes-backend/handlers"
)

func main() {
	db.Init()
	defer db.Close()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Assigning routes directly to handlers
	r.Get("/stations", handlers.GetAllStationsInfo)
	r.Get("/station/{terminalName}", handlers.GetStationStatus)

	// Handle graceful shutdown
	srv := &http.Server{Addr: ":8080", Handler: r}
	go func() {
		log.Println("Starting server on :8080...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on :8080: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	srv.Close() // Close the server to ensure deferred db.CloseDB() is called
}
