package base

type LoopState byte

const (
	// 迴圈繼續往下執行
	KEEPGOING LoopState = 0
	// 呼叫 continue
	CONTINUE LoopState = 1
	// 呼叫 break
	BREAK LoopState = 2
)
