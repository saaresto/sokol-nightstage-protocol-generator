package main

import (
	"encoding/csv"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/xuri/excelize/v2"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const LAPS_IN_SESSION int = 3
const SESSION_COUNT int = 3

var LAPTIME_THRESHOLD, _ = time.ParseDuration("3m")

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

type Timespan time.Duration

func (t Timespan) Format(format string) string {
	z := time.Unix(0, 0).UTC()
	return z.Add(time.Duration(t)).Format(format)
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
