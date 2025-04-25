package base

import "github.com/j32u4ukh/gos/define"

type OnEventsFunc map[define.EventType]func(data any)
