package handlers

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mehmetfazil/borisbikes-backend/db"
)

type Station struct {
	Name         string  `json:"name"`
	TerminalName string  `json:"terminal_name"`
	Latitude     float64 `json:"lat"`
	Longitude    float64 `json:"long"`
}

func GetAllStationsInfo(w http.ResponseWriter, r *http.Request) {
	url := "https://tfl.gov.uk/tfl/syndication/feeds/cycle-hire/livecyclehireupdates.xml"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CustomAgent/1.0)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch station data", http.StatusInternalServerError)
		log.Printf("Error fetching TFL feed: %v", err)
		return
	}
	defer resp.Body.Close()

	type StationXML struct {
		Name         string `xml:"name"`
		TerminalName string `xml:"terminalName"`
		Lat          string `xml:"lat"`
		Long         string `xml:"long"`
	}

	type TFLFeed struct {
		Stations []StationXML `xml:"station"`
	}

	var feed TFLFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		http.Error(w, "Failed to parse station data", http.StatusInternalServerError)
		log.Printf("Error parsing XML: %v", err)
		return
	}

	var stations []Station
	for _, station := range feed.Stations {
		lat, _ := strconv.ParseFloat(strings.TrimSpace(station.Lat), 64)
		long, _ := strconv.ParseFloat(strings.TrimSpace(station.Long), 64)

		stations = append(stations, Station{
			Name:         strings.TrimSpace(station.Name),
			TerminalName: strings.TrimSpace(station.TerminalName),
			Latitude:     lat,
			Longitude:    long,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stations); err != nil {
		http.Error(w, "Failed to encode station data", http.StatusInternalServerError)
		log.Printf("Error encoding JSON: %v", err)
	}
}

type StationStatus struct {
	LastUpdate      string `json:"last_update"`
	NbEbikes        int    `json:"nb_ebikes"`
	NbStandardBikes int    `json:"nb_standard_bikes"`
	NbEmptyDocks    int    `json:"nb_empty_docks"`
}

func GetStationStatus(w http.ResponseWriter, r *http.Request) {
	terminalName := chi.URLParam(r, "terminalName")

	// 1) Check length constraint
	if len(terminalName) == 0 || len(terminalName) > 10 {
		http.Error(w, "Invalid station identifier", http.StatusBadRequest)
		return
	}

	// 2) Whitelist valid characters (adjust as needed: alpha, numeric, etc.)
	isValid := regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString
	if !isValid(terminalName) {
		http.Error(w, "Invalid station identifier", http.StatusBadRequest)
		return
	}

	// 3) Safely build the query with the validated terminalName
	//    The library does not support parameterized queries, so we validate carefully.
	query := fmt.Sprintf(`
        SELECT 
            last_update, 
            nb_ebikes, 
            nb_standard_bikes, 
            nb_empty_docks 
        FROM livecyclehireupdates
        WHERE terminal_name = '%s'
        ORDER BY last_update DESC 
        LIMIT 1;
    `, terminalName)

	// Pass the values in a slice to match the placeholders
	result, err := db.Db.Select(query)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		log.Printf("Database query error: %v", err)
		return
	}

	if result.GetNumberOfRows() == 0 {
		http.Error(w, "Station not found", http.StatusNotFound)
		log.Printf("No data found for station: %s", terminalName)
		return
	}

	// Extract each column
	lastUpdate, err := result.GetStringValue(0, 0)
	if err != nil {
		http.Error(w, "Failed to retrieve last_update", http.StatusInternalServerError)
		log.Printf("Error retrieving last_update: %v", err)
		return
	}
	nbEbikes, err := result.GetInt64Value(0, 1)
	if err != nil {
		http.Error(w, "Failed to retrieve nb_ebikes", http.StatusInternalServerError)
		log.Printf("Error retrieving nb_ebikes: %v", err)
		return
	}
	nbStandardBikes, err := result.GetInt64Value(0, 2)
	if err != nil {
		http.Error(w, "Failed to retrieve nb_standard_bikes", http.StatusInternalServerError)
		log.Printf("Error retrieving nb_standard_bikes: %v", err)
		return
	}
	nbEmptyDocks, err := result.GetInt64Value(0, 3)
	if err != nil {
		http.Error(w, "Failed to retrieve nb_empty_docks", http.StatusInternalServerError)
		log.Printf("Error retrieving nb_empty_docks: %v", err)
		return
	}

	// Build your response
	response := StationStatus{
		LastUpdate:      lastUpdate,
		NbEbikes:        int(nbEbikes),
		NbStandardBikes: int(nbStandardBikes),
		NbEmptyDocks:    int(nbEmptyDocks),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Encoding error: %v", err)
	}
}

type StationHistory struct {
	LastUpdate      string `json:"last_update"`
	NbStandardBikes int    `json:"nb_standard_bikes"`
	NbEbikes        int    `json:"nb_ebikes"`
}

func GetStationHistory(w http.ResponseWriter, r *http.Request) {
	terminalName := chi.URLParam(r, "terminalName")

	// 1) Check length constraint
	if len(terminalName) == 0 || len(terminalName) > 10 {
		http.Error(w, "Invalid station identifier", http.StatusBadRequest)
		return
	}

	// 2) Whitelist valid characters
	isValid := regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString
	if !isValid(terminalName) {
		http.Error(w, "Invalid station identifier", http.StatusBadRequest)
		return
	}

	// 3) Calculate date range for the past 7 days
	sevenDaysAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02 15:04:05")

	// 4) Build the query string
	query := fmt.Sprintf(`
        SELECT 
            last_update,
            nb_standard_bikes, 
            nb_ebikes
        FROM livecyclehireupdates
        WHERE terminal_name = '%s' AND last_update >= '%s'
        ORDER BY last_update DESC
		LIMIT 1000;
    `, terminalName, sevenDaysAgo)

	// Execute the query
	result, err := db.Db.Select(query)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		log.Printf("Database query error: %v", err)
		return
	}

	// Convert the result to JSON using the ToJSON() method
	jsonData := result.ToJSON()

	// Respond with the JSON data
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(jsonData))
}
