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

import "time"

const (
	FOLLOWER               = 0
	CANDIDATE              = 1
	LEADER                 = 2
	ElectionTimerInterval  = 10 * time.Millisecond
	BroadcastTimerInterval = 10 * time.Millisecond
	BroadcastTimeout       = 100 //Millisecond
)
