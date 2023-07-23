package main

const (
	SystemCmd     int32 = 0
	NormalCmd     int32 = 1
	CommissionCmd int32 = 2
)

// SystemCmd
const (
	HeartbeatService    int32 = 0
	IntroductionService int32 = 1
)

// NormalCmd
const (
	TimerService int32 = 0
)