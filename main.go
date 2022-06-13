package main

import (
	_ "embed"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	//go:embed template/index.html
	indexPage string

	LAPS_IN_SESSION      int = 3
	SESSION_COUNT        int = 3
	LAPTIME_THRESHOLD, _     = time.ParseDuration("3m")
	EMPTY_LAP, _             = time.ParseDuration("0s")
)

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

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	http.ServeFile(w, r, indexPage) // https://stackoverflow.com/questions/70068302/how-to-serve-file-from-go-embed
}

func main() {

	r := mux.NewRouter()
	r.HandleFunc("/", handleIndex)
	r.HandleFunc("/upload-protocol", handleProtocolUpload)
	r.HandleFunc("/upload-trackday", handleTrackdayUpload)

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal(err)
		return
	}
}
