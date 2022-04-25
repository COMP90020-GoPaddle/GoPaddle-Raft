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
