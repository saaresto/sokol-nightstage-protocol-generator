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
	Sessions   []Session
	TotalTime  time.Duration
}

type TAClass struct {
	Name    string
	Drivers []DriverResult
}
