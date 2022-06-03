package main

import "time"

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
	DriverNo   string
	Class      string
	Sessions   []Session
	TotalTime  time.Duration
}

type TAClass struct {
	Name    string
	Drivers []DriverResult
}

type DriverResultsAscendingLapTimeSort []DriverResult

func (e DriverResultsAscendingLapTimeSort) Len() int {
	return len(e)
}

func (e DriverResultsAscendingLapTimeSort) Less(i, j int) bool {
	return e[i].TotalTime < e[j].TotalTime
}

func (e DriverResultsAscendingLapTimeSort) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
