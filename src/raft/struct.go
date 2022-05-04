package raft

import (
	"GoPaddle-Raft/labrpc"
	"sync"
)

type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's State
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted State
	Me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	applyCh chan ApplyMsg // channel to send commit

	// Persistent State on all servers
	CurrentTerm int
	VotedFor    int
	log         []LogEntry
	State       int
	leaderId    int

	// Volatile State on all servers
	CommitIndex int
	LastApplied int

	// Volatile State on leaders
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
	// signals of appendEntriesTimer and electionTimer
	leaderCond    *sync.Cond
	nonLeaderCond *sync.Cond

	// Apply signal for new committed entry when updating the CommitIndex
	applyCond *sync.Cond

	// Relevant server info in a string list
	ServerInfo []string
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
	LeaderCommit int        // leader's CommitIndex
}

type AppendEntriesReply struct {
	Term          int // CurrentTerm, for leader to update itself
	Success       bool
	ConflictIndex int
}
