package main

import (
	"encoding/csv"
	"fmt"
	"github.com/xuri/excelize/v2"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func handleTrackdayUpload(w http.ResponseWriter, r *http.Request) {
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
			if len(lap.DriverName) > 1 {
				laps = append(laps, lap)
			}
		}

	}

	sheetsData := processTrackdayLaps(laps)

	excelFile := createTrackdayProtocol(sheetsData)
	defer excelFile.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=protocol.xlsx")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	excelFile.Write(w)
}

func createTrackdayProtocol(data []TrackDayClass) *excelize.File {
	protocol := excelize.NewFile()
	var initialY = 'A'
	var initialX = 2

	for _, taClass := range data {
		protocol.NewSheet(taClass.Name)
		// header row
		protocol.SetCellValue(taClass.Name, "B1", "Пилот")
		protocol.SetCellValue(taClass.Name, "C1", "Автомобиль")
		protocol.SetCellValue(taClass.Name, "D1", "Лучшее время")

		for di, lap := range taClass.Laps {
			nameAndCar := strings.Split(lap.DriverName, "(")
			if len(nameAndCar) == 1 {
				nameAndCar = append(nameAndCar, ")")
			}
			i := 0
			protocol.SetCellValue(taClass.Name, string(initialY+int32(i))+strconv.Itoa(initialX+di), di+1)
			i++
			protocol.SetCellValue(taClass.Name, string(initialY+int32(i))+strconv.Itoa(initialX+di), nameAndCar[0])
			i++
			protocol.SetCellValue(taClass.Name, string(initialY+int32(i))+strconv.Itoa(initialX+di), nameAndCar[1][0:len(nameAndCar[1])-1])
			i++
			protocol.SetCellValue(taClass.Name, string(initialY+int32(i))+strconv.Itoa(initialX+di), Timespan(lap.LapTime).Format("04:05.000"))
			i++
		}
	}

	protocol.DeleteSheet("Sheet1")
	return protocol
}

func processTrackdayLaps(laps []Lap) []TrackDayClass {
	bestLapsByDriver := make(map[string]Lap)
	for _, l := range laps {
		bestLap, exists := bestLapsByDriver[l.DriverName]
		if !exists {
			bestLapsByDriver[l.DriverName] = l
		} else {
			if l.LapTime < bestLap.LapTime {
				bestLapsByDriver[l.DriverName] = l
			}
		}
	}

	lapsByClasses := make(map[string][]Lap)
	for _, lap := range bestLapsByDriver {
		classLaps, exists := lapsByClasses[lap.Class]
		if exists {
			lapsByClasses[lap.Class] = append(classLaps, lap)
		} else {
			lapsByClasses[lap.Class] = append(make([]Lap, 0), lap)
		}
	}

	trackDayClasses := make([]TrackDayClass, 0)
	for class, classLaps := range lapsByClasses {
		sort.Sort(TrackDayClassAscendingLapTimeSort(classLaps))
		trackDayClasses = append(trackDayClasses, TrackDayClass{
			Name: class,
			Laps: classLaps,
		})
	}
	return trackDayClasses
}
