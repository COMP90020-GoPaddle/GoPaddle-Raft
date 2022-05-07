package raft

import (
	"fmt"
	"log"
	"strings"
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

func logEntryToStr(newEntry LogEntry) string {
	rowCommStr := fmt.Sprintf("%v", newEntry.Command)
	split := strings.Split(rowCommStr[1:len(rowCommStr)-1], " ")
	var result string
	if len(split) == 4 {
		result = fmt.Sprintf("T:[%v] K:[%v] V:[] O:[%v] CID:[%v] ReqID:[%v]\n", newEntry.Term,
			split[0], split[1], split[2], split[3])
	} else if len(split) == 5 {
		result = fmt.Sprintf("T:[%v] K:[%v] V:[%v] Oper:[%v] CID:[%v] ReqID:[%v]\n", newEntry.Term,
			split[0], split[1], split[2], split[3], split[4])
	}

	return result
}
