package main

const (
	SystemCmd int32 = 0
	NormalCmd int32 = 1
)

// SystemCmd
const (
	ServerHeartbeatService int32 = 0
	ClientHeartbeatService int32 = 1
	IntroductionService    int32 = 2
)

// Normal
const (
	TimerRequestService  int32 = 0
	TimerResponseService int32 = 1
)
