package raft

import (
	"fmt"
	"log"
	"time"
)

// Debugging
const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

func DLog(format string, a ...interface{}) string {
	return time.Now().Format("2006/01/02 15:04:05 ") + fmt.Sprintf(format, a...)
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}
