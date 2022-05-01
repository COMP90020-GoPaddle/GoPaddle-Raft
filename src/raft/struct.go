package raft

import (
	"GoPaddle-Raft/labrpc"
	"sync"
)

type Raft struct {
	mu        sync.RWMutex        // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	applyCh chan ApplyMsg // channel to send commit

	// Persistent state on all servers
	currentTerm int
	votedFor    int
	log         []LogEntry
	state       int
	leaderId    int

	// Volatile state on all servers
	commitIndex int
	lastApplied int

	// Volatile state on leaders
	nextIndex  []int // for each server, index of the next log entry to send to that server
	matchIndex []int // for each server, index of highest log entry known to be replicated on server

	// Timer
	electionSignalChan chan bool
	// heartbeat and appendEntries are handled together
	broadcastSignalChan    chan bool
	lastResetElectionTime  int64
	lastResetBroadcastTime int64
	electionTimeout        int64
	broadcastTimeout       int64

	// Apply signal for new committed entry when updating the commitIndex
	applyCond *sync.Cond
}

type LogEntry struct {
	Term    int
	Command interface{}
}

type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

type RequestVoteArgs struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term         int
	PrevLogIndex int        // index of log entry immediately preceding new ones
	PrevLogTerm  int        // term of PreLogIndex entry
	Entries      []LogEntry // log entries to store
	LeaderCommit int        // leader's commitIndex
}

type AppendEntriesReply struct {
	Term          int // currentTerm, for leader to update itself
	Success       bool
	ConflictIndex int
}
