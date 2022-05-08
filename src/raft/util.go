/*
Team: GoPaddle

Team Members:
Zixin Wang   1047486  zixin3@student.unimelb.edu.au
Wenjun Wang  1249890  www4@student.unimelb.edu.au
Xinhao Chen  1230696  xinhchen1@student.unimelb.edu.au
Bocan Yang   1152078  bocany@student.unimelb.edu.au

RPC Module and Raft Framework:
MIT 6.824 Lab2:	“6.824 Lab 2: Raft,” Mit.edu. [Online]. Available: https://pdos.csail.mit.edu/6.824/labs/lab-raft.html.
MIT 6.824 Lab3:	“6.824 lab 3: Fault-tolerant key/value service,” Mit.edu. [Online]. Available: https://pdos.csail.mit.edu/6.824/labs/lab-kvraft.html.

Algorithm implementation，variable names，and any optimization ideas is following:
Raft Paper:	D. Ongaro and J. Ousterhout, “In search of an understandable consensus algorithm (extended version),” Mit.edu. [Online]. Available: https://pdos.csail.mit.edu/6.824/papers/raft-extended.pdf.
*/

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

func logEntryToStr(logIdx int, newEntry LogEntry) string {
	rowCommStr := fmt.Sprintf("%v", newEntry.Command)
	split := strings.Split(rowCommStr[1:len(rowCommStr)-1], " ")
	var result string
	if len(split) == 4 {
		//result = fmt.Sprintf("T:[%v] K:[%v] V:[] O:[%v] CID:[%v] ReqID:[%v]\n", newEntry.Term,
		//	split[0], split[1], split[2], split[3])
		result = fmt.Sprintf("I:[%v] T:[%v] K:[%v] V:[] O:[%v] CID:[%v]\n", logIdx, newEntry.Term,
			split[0], split[1], split[2])
	} else if len(split) == 5 {
		//result = fmt.Sprintf("T:[%v] K:[%v] V:[%v] O:[%v] CID:[%v] ReqID:[%v]\n", newEntry.Term,
		//	split[0], split[1], split[2], split[3], split[4])
		result = fmt.Sprintf("I:[%v] T:[%v] K:[%v] V:[%v] O:[%v] CID:[%v]\n", logIdx, newEntry.Term,
			split[0], split[1], split[2], split[3])
	}

	return result
}
