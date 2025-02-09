package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	sqlitecloud "github.com/sqlitecloud/sqlitecloud-go"
)

var db *sqlitecloud.SQCloud

func main() {
	loadEnv()
	initDB()
	defer db.Close()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/stations", getAllStations)

	log.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func loadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}
}

func initDB() {
	connStr := os.Getenv("SQLITECLOUD_CONN_STR")
	if connStr == "" {
		log.Fatal("SQLITECLOUD_CONN_STR environment variable not set")
	}

	var err error
	db, err = sqlitecloud.Connect(connStr)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite Cloud: %v", err)
	}
}

func getAllStations(w http.ResponseWriter, r *http.Request) {
	query := `SELECT DISTINCT terminal_name FROM livecyclehireupdates;`
	result, err := db.Select(query)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Query error: %v", err)
		return
	}

	var stations []string
	for r := uint64(0); r < result.GetNumberOfRows(); r++ {
		stationName, _ := result.GetStringValue(r, 0)
		stations = append(stations, stationName)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stations); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Encoding error: %v", err)
	}
}
