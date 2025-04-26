package utils

import "time"

func GetSafetyTimerChan(t *time.Timer) <-chan time.Time {
	if t != nil {
		return t.C
	}
	// 若 timer 沒啟用，回傳一個永不觸發的 channel
	return make(<-chan time.Time)
}
