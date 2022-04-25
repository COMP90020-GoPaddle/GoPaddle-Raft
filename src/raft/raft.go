package raft

import (
	"6.824/labrpc"
	"math/rand"
	"sync/atomic"
	"time"
)

// Election timer (Only used when not leader)
func (rf *Raft) electionTimer() {
	// use goroutine to keep running
	for {
		rf.mu.RLock()
		// whenever find the cur state is not leader
		if rf.state != LEADER {
			elapse := time.Now().UnixMilli() - rf.lastResetElectionTime
			// notify the raft server to initialize election when election timeout
			if elapse > rf.electionTimeout {
				DPrintf("[electionTimer] | raft %d election timeout %d | current term: %d | current state: %d\n",
					rf.me, rf.electionTimeout, rf.currentTerm, rf.state)
				rf.electionSignalChan <- true
			}
		}
		rf.mu.RUnlock()
		// use sleep to avoid holding the lock
		time.Sleep(ElectionTimerInterval)
	}
}

// Election timer reset
func (rf *Raft) electionTimerReset() {
	// set last reset time to now
	rf.lastResetElectionTime = time.Now().UnixMilli()
	// create new random timeout after reset
	rf.electionTimeout = rf.broadcastTimeout*3 + rand.Int63n(150)
}

// broadcast timer (Only used when leader)
func (rf *Raft) broadcastTimer() {
	// use goroutine to keep running
	for {
		rf.mu.RLock()
		// confirm that the current state is leader
		if rf.state == LEADER {
			elapse := time.Now().UnixMilli() - rf.lastResetBroadcastTime
			// notify the raft server(leader) to broadcast when broadcast timeout
			if elapse > rf.broadcastTimeout {
				DPrintf("[broadcastTimer] | leader raft %d  broadcast timeout | current term: %d | current state: %d\n",
					rf.me, rf.currentTerm, rf.state)
				rf.broadcastSignalChan <- true
			}
		}
		rf.mu.RUnlock()
		// use sleep to avoid holding the lock
		time.Sleep(BroadcastTimerInterval)
	}
}

// broadcast timer reset
func (rf *Raft) broadcastTimerReset() {
	rf.lastResetBroadcastTime = time.Now().UnixMilli()
}

// The main loop of the raft server
func (rf *Raft) mainLoop() {
	for !rf.killed() {
		select {
		// only one of the cases will be satisfied
		case <-rf.broadcastSignalChan:
			rf.broadcast()
		case <-rf.electionSignalChan:
			rf.startElection()
		}
	}
}

// candidate start election
func (rf *Raft) startElection() {
	rf.mu.Lock()
	// convertTo candidate including reset timeout and make currentTerm+1
	rf.convertTo(CANDIDATE)
	// already vote for itself
	voteCnt := 1
	rf.mu.Unlock()
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}
		// for each peer start a goroutine to send RequestVote
		go func(id int) {
			rf.mu.RLock()
			lastLogIndex := len(rf.log) - 1
			args := &RequestVoteArgs{
				Term:         rf.currentTerm,
				CandidateId:  rf.me,
				LastLogIndex: lastLogIndex,
				LastLogTerm:  rf.log[lastLogIndex].Term,
			}
			rf.mu.RUnlock()
			reply := &RequestVoteReply{}
			// state of sending
			if rf.sendRequestVote(id, args, reply) {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				// check whether state changed during broadcasting
				if rf.state != CANDIDATE {
					DPrintf("[startElection| changed state] raft %d state changed | current term: %d | current state: %d\n",
						rf.me, rf.currentTerm, rf.state)
					return
				}
				// get vote from peer
				if reply.VoteGranted == true {
					voteCnt++
					DPrintf("[startElection | reply true] raft %d get accept vote from %d | current term: %d | current state: %d | reply term: %d | voteCnt: %d\n",
						rf.me, id, rf.currentTerm, rf.state, reply.Term, voteCnt)
					// win the majority
					if voteCnt > len(rf.peers)/2 && rf.state == CANDIDATE {
						rf.convertTo(LEADER)
						DPrintf("[startElection | become leader] raft %d convert to leader | current term: %d | current state: %d\n",
							rf.me, rf.currentTerm, rf.state)
						// reinitialize after election
						for i := 0; i < len(rf.peers); i++ {
							// initialized to leader last log index + 1
							rf.nextIndex[i] = len(rf.log)
							rf.matchIndex[i] = 0
						}
						//broadcast immediately
						rf.broadcastSignalChan <- true
					}
				} else {
					DPrintf("[startElection | reply false] raft %d get reject vote from %d | current term: %d | current state: %d | reply term: %d | VoteCnt: %d\n",
						rf.me, id, rf.currentTerm, rf.state, reply.Term, voteCnt)
					// get higher term, convert to follower and match the term
					if rf.currentTerm < reply.Term {
						rf.convertTo(FOLLOWER)
						rf.currentTerm = reply.Term
					}
				}
			} else {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				DPrintf("[startElection | no reply] raft %d RPC to %d failed | current term: %d | current state: %d | reply term: %d\n",
					rf.me, id, rf.currentTerm, rf.state, reply.Term)
			}
		}(i)
	}

}

// leader broadcast heartbeat/appendEntries
func (rf *Raft) broadcast() {
	rf.mu.Lock()
	// if not leader anymore
	if rf.state != LEADER {
		DPrintf("[broadcast | not leader] raft %d lost leadership | current term: %d | current state: %d\n",
			rf.me, rf.currentTerm, rf.state)
		return
	}
	rf.broadcastTimerReset()
	rf.mu.Unlock()
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}
		// for each follower start a goroutine to broadcast heartbeat and appendEntries
		go func(id int) {
		RETRY:
			rf.mu.RLock()
			args := &AppendEntriesArgs{
				Term:         rf.currentTerm,
				PrevLogIndex: rf.nextIndex[id] - 1,
				PrevLogTerm:  rf.log[rf.nextIndex[id]-1].Term,
				// send AppendEntries RPC with log entries starting at nextIndex
				// if heartbeat Entries will be nil
				Entries:      rf.log[rf.nextIndex[id]:],
				LeaderCommit: rf.commitIndex,
			}
			rf.mu.RUnlock()
			reply := &AppendEntriesReply{}
			// state of broadcasting
			if rf.sendAppendEntries(id, args, reply) {
				rf.mu.Lock()
				// check state whether changed during broadcasting
				if rf.state != LEADER {
					DPrintf("[broadcast| changed state] raft %d lost leadership | current term: %d | current state: %d\n",
						rf.me, rf.currentTerm, rf.state)
					rf.mu.Unlock()
					return
				}
				// whether the appendEntries are accepted
				if reply.Success {
					// update the matchIndex and nextIndex, check if the logEntry can be committed
					DPrintf("[broadcast | reply true] raft %d broadcast to %d accepted | current term: %d | current state: %d\n",
						rf.me, id, rf.currentTerm, rf.state)
					// index of the highest log entry known to be replicated
					rf.matchIndex[id] = args.PrevLogIndex + len(args.Entries)
					// index of the next log entry to send
					rf.nextIndex[id] = rf.matchIndex[id] + 1
					rf.checkN()
				} else {
					DPrintf("[broadcast | reply false] raft %d broadcast to %d rejected | current term: %d | current state: %d | reply term: %d\n",
						rf.me, id, rf.currentTerm, rf.state, reply.Term)
					// get higher term, convert to follower and match the term
					if rf.currentTerm < reply.Term {
						rf.convertTo(FOLLOWER)
						rf.currentTerm = reply.Term
						rf.mu.Unlock()
						return
					}
					// update the AppendEntriesArgs and retry
					rf.nextIndex[id] = reply.ConflictIndex
					DPrintf("[appendEntriesAsync] raft %d append entries to %d rejected: decrement nextIndex and retry | nextIndex: %d\n",
						rf.me, id, rf.nextIndex[id])
					rf.mu.Unlock()
					goto RETRY
				}
				rf.mu.Unlock()
			} else {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				// failed broadcasting
				DPrintf("[broadcast | no reply] raft %d RPC to %d failed | current term: %d | current state: %d | reply term: %d\n",
					rf.me, id, rf.currentTerm, rf.state, reply.Term)
			}
		}(i)
	}
}

// check if the logEntry can be committed, call with lock
func (rf *Raft) checkN() {
	// if N > commitIndex, a majority of matchIndex[i] ≥ N, and log[N].term == currentTerm: set commitIndex = N
	for N := len(rf.log) - 1; N > rf.commitIndex && rf.log[N].Term == rf.currentTerm; N-- {
		nReplicated := 0
		for i := 0; i < len(rf.peers); i++ {
			if rf.matchIndex[i] >= N && rf.log[N].Term == rf.currentTerm {
				nReplicated += 1
			}
			// check the majority
			if nReplicated > len(rf.peers)/2 {
				rf.commitIndex = N
				// logEntry can be committed, append to applyCh
				go rf.applyEntries()
				break
			}
		}
	}
}

// GetState return currentTerm and whether this server believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	term := rf.currentTerm
	isLeader := rf.state == LEADER
	return term, isLeader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {
	// Your code here (2D).
	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).
}

// RequestVote RPC handler
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// get high term, convert to follower and match the term
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.convertTo(FOLLOWER)
	}
	// do not grant vote due to smaller term or already voted for another one
	if args.Term < rf.currentTerm || (rf.votedFor != -1 && rf.votedFor != args.CandidateId) {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		DPrintf("[RequestVote] raft %d reject vote for %d | current term: %d | current state: %d | recieved term: %d | voteFor: %d\n",
			rf.me, args.CandidateId, rf.currentTerm, rf.state, args.Term, rf.votedFor)
		return
	}
	// not vote yet or already voted for it before
	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		// check if candidate's log is at least as up-to-date as its log
		lastLogIndex := len(rf.log) - 1
		if rf.log[lastLogIndex].Term > args.LastLogTerm ||
			(rf.log[lastLogIndex].Term == args.LastLogTerm && args.LastLogIndex < lastLogIndex) {
			reply.Term = rf.currentTerm
			reply.VoteGranted = false
			return
		}
		// grant vote
		rf.votedFor = args.CandidateId
		// avoid two election proceeding in parallel
		rf.electionTimerReset()
		reply.Term = rf.currentTerm
		reply.VoteGranted = true
		DPrintf("[RequestVote] raft %d accept vote for %d | current term: %d | current state: %d | recieved term: %d\n",
			rf.me, args.CandidateId, rf.currentTerm, rf.state, args.Term)
	}
}

// RequestVote RPC
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// AppendEntries RPC
func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

// AppendEntries RPC handler
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// get lower term, refuse the AppendEntries and send back its term
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		DPrintf("[AppendEntries| small term] raft %d reject append entries | current term: %d | current state: %d | received term: %d\n",
			rf.me, rf.currentTerm, rf.state, args.Term)
		return
	}
	// get higher term, convert to follower and match the term
	if args.Term >= rf.currentTerm {
		rf.convertTo(FOLLOWER)
		rf.currentTerm = args.Term
		DPrintf("[AppendEntries| big term or has leader] raft %d update term or state | current term: %d | current state: %d | recieved term: %d\n",
			rf.me, rf.currentTerm, rf.state, args.Term)
	}
	//smaller length or unmatched term at PrevLogIndex
	if len(rf.log) <= args.PrevLogIndex || rf.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		DPrintf("[AppendEntries| inconsistent log] raft %d reject append entry | log len: %d | args.PrevLogIndex: %d | args.prevLogTerm %d\n",
			rf.me, len(rf.log), args.PrevLogIndex, args.PrevLogTerm)
		reply.Term = rf.currentTerm
		reply.Success = false
		if len(rf.log) <= args.PrevLogIndex {
			reply.ConflictIndex = len(rf.log)
		} else {
			// search for the ConflictIndex forward
			for i := args.PrevLogIndex; i >= 0; i-- {
				if rf.log[i].Term != rf.log[i-1].Term {
					reply.ConflictIndex = i
					break
				}
			}
		}
	} else {
		isMatch := true
		nextIndex := args.PrevLogIndex + 1
		conflictIndex := 0
		logLen := len(rf.log)
		entLen := len(args.Entries)
		// do the matching
		for i := 0; isMatch && i < entLen; i++ {
			if ((logLen - 1) < (nextIndex + i)) || rf.log[nextIndex+i].Term != args.Entries[i].Term {
				isMatch = false
				conflictIndex = i
				break
			}
		}
		if !isMatch {
			rf.log = append(rf.log[:nextIndex+conflictIndex], args.Entries[conflictIndex:]...)
			DPrintf("[AppendEntries] raft %d appended entries from leader | log length: %d\n", rf.me, len(rf.log))
		}
		lastNewEntryIndex := args.PrevLogIndex + entLen
		if args.LeaderCommit > rf.commitIndex {
			rf.commitIndex = min(args.LeaderCommit, lastNewEntryIndex)
			// apply entries after update commitIndex
			go rf.applyEntries()
		}
		reply.Term = rf.currentTerm
		reply.Success = true
	}
}

// apply entries and set CommandValid to true
func (rf *Raft) applyEntries() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// if commitIndex > lastApplied: increment lastApplied, apply log[lastApplied] to state machine
	for i := rf.lastApplied + 1; i <= rf.commitIndex; i++ {
		applyMsg := ApplyMsg{
			CommandValid: true,
			Command:      rf.log[i].Command,
			CommandIndex: i,
		}
		rf.applyCh <- applyMsg
		rf.lastApplied += 1
		DPrintf("[applyEntries] raft %d applied entry | lastApplied: %d | commitIndex: %d\n",
			rf.me, rf.lastApplied, rf.commitIndex)
	}
}

// State conversion, should be within writeLock
func (rf *Raft) convertTo(state int) {
	switch state {
	case FOLLOWER:
		rf.electionTimerReset()
		rf.votedFor = -1
		rf.state = FOLLOWER
	case CANDIDATE:
		rf.electionTimerReset()
		rf.state = CANDIDATE
		rf.currentTerm++
		rf.votedFor = rf.me
	case LEADER:
		// broadcast includes heartbeat and appendEntries
		rf.broadcastTimerReset()
		rf.state = LEADER
	}
}

// Start take a command as input and return the index of next logEntry, current term and if this server believes it is the leader
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true
	if term, isLeader = rf.GetState(); isLeader {
		rf.mu.Lock()
		rf.log = append(rf.log, LogEntry{Command: command, Term: rf.currentTerm})
		rf.matchIndex[rf.me] = len(rf.log) - 1
		index = len(rf.log) - 1
		DPrintf("[Start] raft %d replicate command to log | current term: %d | current state: %d | log length: %d\n",
			rf.me, rf.currentTerm, rf.state, len(rf.log))
		rf.mu.Unlock()
	}
	return index, term, isLeader
}

// Kill the raft server
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
}

// check whether the raft server is killed
func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// Make a raft server and do initialization
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	rf.state = FOLLOWER
	rf.currentTerm = 0
	rf.votedFor = -1

	rf.broadcastSignalChan = make(chan bool)
	rf.electionSignalChan = make(chan bool)
	rf.broadcastTimeout = BroadcastTimeout
	rf.electionTimerReset()

	rf.log = append(rf.log, LogEntry{Term: 0})
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.applyCh = applyCh

	DPrintf("Starting raft %d\n", me)
	// do correspond action according to the channel
	go rf.mainLoop()
	// used when raft server is not a leader
	go rf.electionTimer()
	// used when raft server is a leader
	go rf.broadcastTimer()

	return rf
}
