package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/sdomino/scribble"
)

type Register struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	Amount   float64   `json:"amount"`
	Currency string    `json:"currency"`
	Date     time.Time `json:"date"`
	Note     string    `json:"note"`
}

var db *scribble.Driver

func main() {
	var err error
	db, err = scribble.New("data", nil)
	if err != nil {
		log.Fatal("Error creating scribble database: ", err)
		return
	}

	r := chi.NewRouter()

	r.Post("/addRegister", addRegister)
	r.Get("/getRegister", getRegisterByFilter)
	r.Get("/getMonthConsumption", getMonthConsumption)

	addr := os.Getenv("ADDR")
	log.Println("Starting server on ", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func addRegister(w http.ResponseWriter, r *http.Request) {
	var newRegister Register

	if err := json.NewDecoder(r.Body).Decode(&newRegister); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate an ID based on the current time (for simplicity)
	newRegister.ID = int(time.Now().Unix())

	if err := db.Write("registers", strconv.Itoa(newRegister.ID), newRegister); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(newRegister)
}

func getRegisterByFilter(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	currency := r.URL.Query().Get("currency")
	startDate, _ := time.Parse(time.RFC3339, r.URL.Query().Get("startDate"))
	endDate, _ := time.Parse(time.RFC3339, r.URL.Query().Get("endDate"))

	data, err := db.ReadAll("registers")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var records []Register
	for _, d := range data {
		var r Register
		if err := json.Unmarshal(d, &r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		records = append(records, r)
	}

	var filteredRegisters []Register
	for _, record := range records {
		if (name == "" || record.Name == name) &&
			(currency == "" || record.Currency == currency) &&
			(startDate.IsZero() || record.Date.After(startDate)) &&
			(endDate.IsZero() || record.Date.Before(endDate)) {
			filteredRegisters = append(filteredRegisters, record)
		}
	}

	json.NewEncoder(w).Encode(filteredRegisters)
}

func getMonthConsumption(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	currency := r.URL.Query().Get("currency")

	data, err := db.ReadAll("registers")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var records []Register
	for _, d := range data {
		var r Register
		if err := json.Unmarshal(d, &r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		records = append(records, r)
	}

	var totalConsumption float64
	for _, record := range records {
		if record.Name == name && record.Currency == currency &&
			time.Since(record.Date).Hours() < 30*24 {
			totalConsumption += record.Amount
		}
	}

	result := map[string]float64{
		"totalConsumption": totalConsumption,
	}

	json.NewEncoder(w).Encode(result)
}
