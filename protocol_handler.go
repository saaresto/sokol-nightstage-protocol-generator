package main

import (
	"encoding/csv"
	"github.com/xuri/excelize/v2"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"
)

func handleProtocolUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, _, err := r.FormFile("file")
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

	excelFile := createProtocol(sheetsData)
	defer excelFile.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=protocol.xlsx")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	excelFile.Write(w)
}

func createProtocol(data []TAClass) *excelize.File {
	protocol := excelize.NewFile()
	//timeFormatStyle, _ := protocol.NewStyle(&excelize.Style{NumFmt: 47})
	//bestLapStyle, _ := protocol.NewStyle(&excelize.Style{Fill: excelize.Fill{
	//	Pattern: 1,
	//	Color:   []string{"yellow"},
	//	Shading: 0,
	//}})
	var initialY = 'A'
	var initialX = 2

	for _, taClass := range data {
		protocol.NewSheet(taClass.Name)
		// header row
		protocol.SetCellValue(taClass.Name, "B1", "Пилот")
		protocol.SetCellValue(taClass.Name, "C1", "No")
		protocol.SetCellValue(taClass.Name, "D1", "1 Круг")
		protocol.SetCellValue(taClass.Name, "E1", "2 Круг")
		protocol.SetCellValue(taClass.Name, "F1", "3 Круг")
		protocol.SetCellValue(taClass.Name, "G1", "1 Сессия")
		protocol.SetCellValue(taClass.Name, "H1", "1 Круг")
		protocol.SetCellValue(taClass.Name, "I1", "2 Круг")
		protocol.SetCellValue(taClass.Name, "J1", "3 Круг")
		protocol.SetCellValue(taClass.Name, "K1", "2 Сессия")
		protocol.SetCellValue(taClass.Name, "L1", "1 Круг")
		protocol.SetCellValue(taClass.Name, "M1", "2 Круг")
		protocol.SetCellValue(taClass.Name, "N1", "3 Круг")
		protocol.SetCellValue(taClass.Name, "O1", "3 Сессия")
		protocol.SetCellValue(taClass.Name, "P1", "Итог")

		for di, driver := range taClass.Drivers {
			//protocol.SetRowStyle(taClass.Name, di, di, timeFormatStyle)
			i := 0
			protocol.SetCellValue(taClass.Name, string(initialY+int32(i))+strconv.Itoa(initialX+di), di+1)
			i++
			protocol.SetCellValue(taClass.Name, string(initialY+int32(i))+strconv.Itoa(initialX+di), driver.DriverName)
			i++
			protocol.SetCellValue(taClass.Name, string(initialY+int32(i))+strconv.Itoa(initialX+di), driver.DriverNo)
			i++

			for _, session := range driver.Sessions {
				for _, lapTime := range session.LapTimes {
					protocol.SetCellValue(taClass.Name, string(initialY+int32(i))+strconv.Itoa(initialX+di), Timespan(lapTime).Format("04:05.000"))
					i++
				}
				for l := len(session.LapTimes); l < LAPS_IN_SESSION; l++ {
					protocol.SetCellValue(taClass.Name, string(initialY+int32(i))+strconv.Itoa(initialX+di), "–")
					i++
				}
				protocol.SetCellValue(taClass.Name, string(initialY+int32(i))+strconv.Itoa(initialX+di), Timespan(session.BestLap).Format("04:05.000"))
				//protocol.SetCellStyle(taClass.Name, string(initialY+int32(i)), strconv.Itoa(initialX+di), bestLapStyle)
				i++
			}

			protocol.SetCellValue(taClass.Name, string(initialY+int32(i))+strconv.Itoa(initialX+di), Timespan(driver.TotalTime).Format("04:05.000"))
			i++
		}
	}

	protocol.DeleteSheet("Sheet1")
	return protocol
}

func processLaps(laps []Lap) []TAClass {
	lapsByDriver := make(map[string][]Lap)
	for _, l := range laps {
		driverLaps := lapsByDriver[l.DriverName]
		if len(driverLaps) < 1 {
			lapsByDriver[l.DriverName] = append([]Lap{}, l)
		} else {
			lapsByDriver[l.DriverName] = append(driverLaps, l)
		}
	}

	driverResults := make([]DriverResult, 0)
	for name, laps := range lapsByDriver {
		driverResult := DriverResult{
			DriverName: name,
			DriverNo:   laps[0].DriverNo,
		}
		sessions := make([]Session, 0)
		session := Session{LapTimes: make([]time.Duration, 0)}
		for _, lap := range laps {
			if lap.LapTime > LAPTIME_THRESHOLD {
				if len(session.LapTimes) > 0 {
					bestLap := LAPTIME_THRESHOLD
					for _, l := range session.LapTimes {
						if l < bestLap {
							bestLap = l
						}
					}
					session.BestLap = bestLap
					sessions = append(sessions, session)
				}
				session = Session{}
				continue
			}
			session.LapTimes = append(session.LapTimes, lap.LapTime)
		}
		bestLap := LAPTIME_THRESHOLD
		for _, l := range session.LapTimes {
			if l < bestLap {
				bestLap = l
			}
		}
		session.BestLap = bestLap
		sessions = append(sessions, session)

		var totalTime time.Duration = 0
		for _, s := range sessions {
			totalTime += s.BestLap
		}

		for i := len(sessions); i < SESSION_COUNT; i++ {
			emptySession := Session{
				LapTimes: make([]time.Duration, LAPS_IN_SESSION),
				BestLap:  EMPTY_LAP,
			}
			for i := 0; i < LAPS_IN_SESSION; i++ {
				emptySession.LapTimes = append(emptySession.LapTimes, EMPTY_LAP)
			}
			sessions = append(sessions, emptySession)
		}

		driverResult.TotalTime = totalTime
		driverResult.Sessions = sessions
		driverResult.Class = laps[0].Class
		driverResults = append(driverResults, driverResult)
	}

	sort.Sort(DriverResultsAscendingLapTimeSort(driverResults))
	classMap := make(map[string][]DriverResult)
	for _, dr := range driverResults {
		c := classMap[dr.Class]
		if c == nil {
			c = make([]DriverResult, 0)
		}
		c = append(c, dr)
		classMap[dr.Class] = c
	}
	classes := make([]TAClass, 0)
	for c, drs := range classMap {
		classes = append(classes, TAClass{
			Name:    c,
			Drivers: drs,
		})
	}
	return classes
}
