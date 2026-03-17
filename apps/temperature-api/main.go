package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type TemperatureResponse struct {
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	Timestamp   time.Time `json:"timestamp"`
	Location    string    `json:"location"`
	Status      string    `json:"status"`
	SensorID    string    `json:"sensor_id"`
	SensorType  string    `json:"sensor_type"`
	Description string    `json:"description"`
}

func locationBySensorID(sensorID string) string {
	switch sensorID {
	case "1":
		return "Living Room"
	case "2":
		return "Bedroom"
	case "3":
		return "Kitchen"
	default:
		return "Unknown"
	}
}

func sensorIDByLocation(location string) string {
	switch location {
	case "Living Room":
		return "1"
	case "Bedroom":
		return "2"
	case "Kitchen":
		return "3"
	default:
		return "0"
	}
}

func generateResponse(sensorID, location string) TemperatureResponse {
	value := 15.0 + rand.Float64()*15.0 // 15.0 — 30.0 °C

	return TemperatureResponse{
		Value:       math.Round(value*100) / 100,
		Unit:        "°C",
		Timestamp:   time.Now(),
		Location:    location,
		Status:      "active",
		SensorID:    sensorID,
		SensorType:  "temperature",
		Description: fmt.Sprintf("Temperature in %s", location),
	}
}

func handleTemperature(w http.ResponseWriter, r *http.Request) {
	location := r.URL.Query().Get("location")
	sensorID := sensorIDByLocation(location)

	if location == "" {
		location = "Unknown"
	}

	resp := generateResponse(sensorID, location)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleTemperatureByID(w http.ResponseWriter, r *http.Request) {
	// path: /temperature/{id}
	parts := strings.Split(r.URL.Path, "/")
	sensorID := parts[len(parts)-1]
	location := locationBySensorID(sensorID)

	resp := generateResponse(sensorID, location)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/temperature", handleTemperature)
	http.HandleFunc("/temperature/", handleTemperatureByID)

	log.Println("Temperature API starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
