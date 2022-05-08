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
	"GoPaddle-Raft/labrpc"
	"sync"

	"fyne.io/fyne/v2/data/binding"
)

type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's State
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted State
	Me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	applyCh     chan ApplyMsg // channel to send commit
	consoleLogs binding.ExternalStringList

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
	ServerInfo binding.ExternalStringList
	ServerLog  binding.ExternalStringList
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
