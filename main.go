package main

import (
	"encoding/csv"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const LAPS_IN_SESSION int = 3

type Lap struct {
	Num         int
	DriverNo    string
	DriverName  string
	LapTime     time.Duration
	Transponder string
	Class       string
}

type Session struct {
	LapTimes []time.Duration
	BestLap  time.Duration
}

type DriverResult struct {
	DriverName string
	DriverNum  string
	Sessions   []Session
	TotalTime  time.Duration
}

type Class struct {
	Name    string
	Drivers []DriverResult
}

func convert(rec []string) (Lap, error) {
	laptime, err := time.ParseDuration(formatTime(rec[5]))
	if err != nil {
		return Lap{}, err
	}
	num, _ := strconv.Atoi(rec[0])
	var class string
	if len(rec[16]) > 1 {
		class = rec[16]
	} else {
		class = "UNDEFINED"
	}

	return Lap{
		Num:         num,
		DriverNo:    rec[1],
		DriverName:  rec[2],
		LapTime:     laptime,
		Transponder: rec[13],
		Class:       class,
	}, nil
}

func formatTime(s string) string {
	formatted := strings.ReplaceAll(s, ":", "m")
	formatted = strings.ReplaceAll(formatted, ".", "s")
	return formatted + "ms"
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("file")
	fmt.Printf("File name: %s\n", header.Filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer file.Close()

	// read csv
	csvReader := csv.NewReader(file)
	csvReader.LazyQuotes = true
	// skip header line
	csvReader.Read()

	var laps []Lap
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		lap, err := convert(rec)
		if err == nil {
			laps = append(laps, lap)
		}

	}

	sheetsData := processLaps(laps)

	fmt.Println(sheetsData)
}

func processLaps(laps []Lap) []Class {
	return nil
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	http.ServeFile(w, r, "template/index.html")
}

func main() {

	r := mux.NewRouter()
	r.HandleFunc("/", handleIndex)
	r.HandleFunc("/upload", handleUpload)

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal(err)
		return
	}
}
