package raft

import "time"

const (
	FOLLOWER               = 0
	CANDIDATE              = 1
	LEADER                 = 2
	ElectionTimerInterval  = 10 * time.Millisecond
	HeartbeatTimerInterval = 10 * time.Millisecond
	HeartbeatTimeout       = 100 //Millisecond
)
