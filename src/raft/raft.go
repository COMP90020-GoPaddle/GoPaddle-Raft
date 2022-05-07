package raft

import (
	"GoPaddle-Raft/labgob"
	"GoPaddle-Raft/labrpc"
	"bytes"
	"fmt"
	"fyne.io/fyne/v2/data/binding"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2/data/binding"
)

// Election timer (Only used when not leader)
func (rf *Raft) electionTimer() {
	// use goroutine to keep running
	for {
		rf.mu.Lock()
		if rf.State == LEADER { // if is leader now, block the electionTimer
			rf.nonLeaderCond.Wait()
		}
		// whenever find the cur State is not leader
		elapse := time.Now().UnixMilli() - rf.lastResetElectionTime
		// notify the raft server to initialize election when election timeout
		if elapse > rf.electionTimeout {
			DPrintf("[electionTimer] | raft %d election timeout %d | current term: %d | current State: %d\n",
				rf.Me, rf.electionTimeout, rf.CurrentTerm, rf.State)
			rf.updateConsoleLogs(DLog("Raft Server[%v]: Timeout %d, Starting election | current term: %d",
				rf.Me+1, rf.electionTimeout, rf.CurrentTerm))
			rf.mu.Unlock()
			rf.electionSignalChan <- true
			rf.mu.Lock()
		}
		rf.mu.Unlock()
		// use sleep to avoid frequently checking
		time.Sleep(ElectionTimerInterval)
	}
}

// Election timer reset
func (rf *Raft) electionTimerReset() {
	// set last reset time to now
	rf.lastResetElectionTime = time.Now().UnixMilli()
	// create new random timeout after reset
	rf.electionTimeout = rf.broadcastTimeout*2 + rand.Int63n(250)
}

// broadcast timer (Only used when leader)
func (rf *Raft) broadcastTimer() {
	// use goroutine to keep running
	for {
		rf.mu.Lock()
		if rf.State != LEADER { // if is not leader now, block the broadcastTimer
			rf.leaderCond.Wait()
		}
		elapse := time.Now().UnixMilli() - rf.lastResetBroadcastTime
		// notify the raft server(leader) to broadcast when broadcast timeout
		if elapse > rf.broadcastTimeout {
			DPrintf("[broadcastTimer] | leader raft %d  broadcast timeout | current term: %d | current State: %d\n",
				rf.Me, rf.CurrentTerm, rf.State)
			rf.mu.Unlock()
			rf.broadcastSignalChan <- true
			rf.mu.Lock()
		}
		rf.mu.Unlock()
		// use sleep to avoid frequently checking
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
			rf.mu.Lock()
			DPrintf("[mainLoop-startBroadCast] | raft %d  start broadcast | current term: %d | current State: %d\n",
				rf.Me, rf.CurrentTerm, rf.State)
			rf.mu.Unlock()
			go rf.broadcast()
		case <-rf.electionSignalChan:
			rf.mu.Lock()
			DPrintf("[mainLoop-Election] | raft %d  start Election | current term: %d | current State: %d\n",
				rf.Me, rf.CurrentTerm, rf.State)
			rf.mu.Unlock()
			go rf.startElection()
		}
	}
}

// candidate start election
func (rf *Raft) startElection() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// convertTo candidate including reset timeout and make currentTerm+1
	rf.convertTo(CANDIDATE)
	// update server info
	rf.updateServerInfo()
	//rf.InfoCh <- true

	// persist the State
	rf.persist()
	// already vote for itself
	voteCnt := 1
	lastLogIndex := len(rf.log) - 1
	args := &RequestVoteArgs{
		Term:         rf.CurrentTerm,
		CandidateId:  rf.Me,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  rf.log[lastLogIndex].Term,
	}
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.Me {
			continue
		}
		// for each peer start a goroutine to send RequestVote
		go func(id int, args *RequestVoteArgs) {
			//rf.mu.RLock()
			//lastLogIndex := len(rf.log) - 1
			//args := &RequestVoteArgs{
			//	Term:         rf.CurrentTerm,
			//	CandidateId:  rf.Me,
			//	LastLogIndex: lastLogIndex,
			//	LastLogTerm:  rf.log[lastLogIndex].Term,
			//}
			//rf.mu.RUnlock()
			reply := &RequestVoteReply{}
			// State of sending
			if rf.sendRequestVote(id, args, reply) {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				// check whether term changed during election
				if rf.CurrentTerm != args.Term {
					DPrintf("[startElection| changed term] raft %d term changed | current term: %d | current State: %d\n",
						rf.Me, rf.CurrentTerm, rf.State)
					return
				}
				// check whether State changed during election
				if rf.State != CANDIDATE {
					DPrintf("[startElection| changed State] raft %d State changed | current term: %d | current State: %d\n",
						rf.Me, rf.CurrentTerm, rf.State)
					return
				}
				// get vote from peer
				if reply.VoteGranted == true {
					voteCnt++
					DPrintf("[startElection | reply true] raft %d get accept vote from %d | current term: %d | current State: %d | reply term: %d | voteCnt: %d\n",
						rf.Me, id, rf.CurrentTerm, rf.State, reply.Term, voteCnt)
					// win the majority
					if voteCnt > len(rf.peers)/2 && rf.State == CANDIDATE {
						rf.convertTo(LEADER)

						// update server info
						rf.updateServerInfo()
						//rf.InfoCh <- true

						DPrintf("[startElection | become leader] raft %d convert to leader | current term: %d | current State: %d\n",
							rf.Me, rf.CurrentTerm, rf.State)
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
					DPrintf("[startElection | reply false] raft %d get reject vote from %d | current term: %d | current State: %d | reply term: %d | VoteCnt: %d\n",
						rf.Me, id, rf.CurrentTerm, rf.State, reply.Term, voteCnt)
					// get higher term, convert to follower and match the term
					if rf.CurrentTerm < reply.Term {
						rf.convertTo(FOLLOWER)
						rf.CurrentTerm = reply.Term

						// update server info
						rf.updateServerInfo()
						//rf.InfoCh <- true

						rf.persist()
					}
				}
			} else {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				DPrintf("[startElection | no reply] raft %d RPC to %d failed | current term: %d | current State: %d | reply term: %d\n",
					rf.Me, id, rf.CurrentTerm, rf.State, reply.Term)
			}
		}(i, args)
	}

}

// leader broadcast heartbeat/appendEntries
func (rf *Raft) broadcast() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// if not leader anymore
	if rf.State != LEADER {
		DPrintf("[broadcast | not leader] raft %d lost leadership | current term: %d | current State: %d\n",
			rf.Me, rf.CurrentTerm, rf.State)
		return
	}
	rf.broadcastTimerReset()
	curTerm := rf.CurrentTerm

	for i := 0; i < len(rf.peers); i++ {
		if i == rf.Me {
			continue
		}
		// for each follower start a goroutine to broadcast heartbeat and appendEntries
		go func(id int, curTerm int) {
		RETRY:
			rf.mu.Lock()
			// check whether current server's term is changed
			if curTerm != rf.CurrentTerm {
				rf.mu.Unlock()
				return
			}
			args := &AppendEntriesArgs{
				Term:         curTerm,
				PrevLogIndex: rf.nextIndex[id] - 1,
				PrevLogTerm:  rf.log[rf.nextIndex[id]-1].Term,
				// send AppendEntries RPC with log entries starting at nextIndex
				// if heartbeat Entries will be nil
				Entries:      rf.log[rf.nextIndex[id]:],
				LeaderCommit: rf.CommitIndex,
			}

			if len(args.Entries) > 0 {
				rf.updateConsoleLogs(DLog("Raft Server[%v]: Command[%v] Send to Server[%d]: %v",
					rf.Me+1, rf.CommitIndex+1, id+1, args.Entries))
			}

			rf.mu.Unlock()
			reply := &AppendEntriesReply{}
			// state of broadcasting

			if rf.sendAppendEntries(id, args, reply) {
				rf.mu.Lock()
				// check State whether changed during broadcasting
				if rf.State != LEADER {
					DPrintf("[broadcast| changed State] raft %d lost leadership | current term: %d | current State: %d\n",
						rf.Me, rf.CurrentTerm, rf.State)
					rf.mu.Unlock()
					return
				}
				// whether the appendEntries are accepted
				if reply.Success {
					if len(args.Entries) > 0 {
						rf.updateConsoleLogs(DLog("Raft Server[%v]: Command[%v] Accepted by Server[%v]: | current term: %d",
							rf.Me+1, args.LeaderCommit+1, id+1, rf.CurrentTerm))
					}
					// update the matchIndex and nextIndex, check if the logEntry can be committed
					DPrintf("[broadcast | reply true] raft %d broadcast to %d accepted | current term: %d | current State: %d\n",
						rf.Me, id, rf.CurrentTerm, rf.State)
					// index of the highest log entry known to be replicated
					rf.matchIndex[id] = args.PrevLogIndex + len(args.Entries)
					// index of the next log entry to send
					rf.nextIndex[id] = rf.matchIndex[id] + 1
					rf.checkN()
				} else {
					if len(args.Entries) > 0 {
						rf.updateConsoleLogs(DLog("Raft Server[%v]: Command[%v] Rejected by Server[%v] | current term: %d | reply term: %d",
							rf.Me+1, args.LeaderCommit+1, id+1, rf.CurrentTerm, reply.Term))
					}
					DPrintf("[broadcast | reply false] raft %d broadcast to %d rejected | current term: %d | current state: %d | reply term: %d\n",
						rf.Me, id, rf.CurrentTerm, rf.State, reply.Term)
					// get higher term, convert to follower and match the term
					if rf.CurrentTerm < reply.Term {
						rf.convertTo(FOLLOWER)
						rf.CurrentTerm = reply.Term
						// update server info
						rf.updateServerInfo()
						//rf.InfoCh <- true

						rf.persist()
						rf.mu.Unlock()
						return
					}
					// update the AppendEntriesArgs and retry
					if reply.ConflictIndex == 0 {
						rf.nextIndex[id] = 1
					} else {
						rf.nextIndex[id] = reply.ConflictIndex
					}
					if len(args.Entries) > 0 {
						rf.updateConsoleLogs(DLog("Raft Server[%v]: Command[%v] Send Retry to Server[%v] | nextIndex: %d",
							rf.Me+1, args.LeaderCommit+1, id+1, rf.nextIndex[id]))
					}
					DPrintf("[appendEntriesAsync] raft %d append entries to %d rejected: decrement nextIndex and retry | nextIndex: %d\n",
						rf.Me, id, rf.nextIndex[id])
					rf.mu.Unlock()
					goto RETRY
				}
				rf.mu.Unlock()
			} else {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				// failed broadcasting
				if len(args.Entries) > 0 {
					rf.updateConsoleLogs(DLog("Raft Server[%v]: Command[%v] RPC to %d failed | current term: %d",
						rf.Me+1, args.LeaderCommit+1, id+1, rf.CurrentTerm))
				}
				DPrintf("[broadcast | no reply] raft %d RPC to %d failed | current term: %d | current state: %d \n",
					rf.Me, id, rf.CurrentTerm, rf.State)
			}
		}(i, curTerm)
	}
}

// check if the logEntry can be committed, call with lock, only leader can call while handling AppendEntries response
func (rf *Raft) checkN() {
	// if N > CommitIndex, a majority of matchIndex[i] ≥ N, and log[N].term == CurrentTerm: set CommitIndex = N
	for N := len(rf.log) - 1; N > rf.CommitIndex && rf.log[N].Term == rf.CurrentTerm; N-- {
		nReplicated := 0
		for i := 0; i < len(rf.peers); i++ {
			if rf.matchIndex[i] >= N && rf.log[N].Term == rf.CurrentTerm {
				nReplicated += 1
			}
			// check the majority
			if nReplicated > len(rf.peers)/2 {
				rf.CommitIndex = N

				//update serverInfo
				rf.updateServerInfo()
				//rf.InfoCh <- true

				// logEntry can be committed, append to applyCh
				rf.applyCond.Broadcast()
				break
			}
		}
	}
}

// GetState return CurrentTerm and whether this server believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term := rf.CurrentTerm
	isLeader := rf.State == LEADER
	return term, isLeader
}

// Save Raft's persistent State to stable storage,
func (rf *Raft) persist() {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.CurrentTerm)
	e.Encode(rf.VotedFor)
	e.Encode(rf.log)
	// try save serverlog in persistent state
	//e.Encode(rf.ServerLog)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
	DPrintf("[persist] raft: %d || currentTerm: %d || votedFor: %d || log len: %d\n", rf.Me, rf.CurrentTerm, rf.VotedFor, len(rf.log))
}

// restore previously persisted State.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any State?
		return
	}
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm int
	var votedFor int
	var log []LogEntry

	if d.Decode(&currentTerm) != nil ||
		d.Decode(&votedFor) != nil ||
		d.Decode(&log) != nil {
		DPrintf("[readPersist] error\n")
	} else {
		rf.CurrentTerm = currentTerm
		rf.VotedFor = votedFor
		rf.log = log
		//update server info
		fmt.Println("readPersist success! -----", rf.CurrentTerm, rf.VotedFor, rf.log)
		rf.updateServerInfo()
		ss, _ := rf.ServerInfo.Get()
		fmt.Println("updateServerInfo success! ------, serverInfo:", ss)
		//rf.InfoCh <- true
	}

	DPrintf("[readPersist] raft: %d || CurrentTerm: %d || VotedFor: %d || log len: %d\n", rf.Me, rf.CurrentTerm, rf.VotedFor, len(log))
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
	if args.Term > rf.CurrentTerm {
		rf.CurrentTerm = args.Term
		rf.convertTo(FOLLOWER)

		// update server info
		rf.updateServerInfo()
		//rf.InfoCh <- true

		rf.persist()
	}
	// do not grant vote due to smaller term or already voted for another one
	if args.Term < rf.CurrentTerm || (rf.VotedFor != -1 && rf.VotedFor != args.CandidateId) {
		reply.Term = rf.CurrentTerm
		reply.VoteGranted = false
		DPrintf("[RequestVote] raft %d reject vote for %d | current term: %d | current State: %d | recieved term: %d | voteFor: %d\n",
			rf.Me, args.CandidateId, rf.CurrentTerm, rf.State, args.Term, rf.VotedFor)
		return
	}
	// not vote yet or already voted for it before
	if rf.VotedFor == -1 || rf.VotedFor == args.CandidateId {
		// check if candidate's log is at least as up-to-date as its log
		lastLogIndex := len(rf.log) - 1
		if rf.log[lastLogIndex].Term > args.LastLogTerm ||
			(rf.log[lastLogIndex].Term == args.LastLogTerm && args.LastLogIndex < lastLogIndex) {
			reply.Term = rf.CurrentTerm
			reply.VoteGranted = false
			return
		}
		// grant vote
		rf.VotedFor = args.CandidateId

		// update server info
		rf.updateServerInfo()
		//rf.InfoCh <- true

		// avoid two election proceeding in parallel
		rf.electionTimerReset()
		rf.persist()
		reply.Term = rf.CurrentTerm
		reply.VoteGranted = true
		DPrintf("[RequestVote] raft %d accept vote for %d | current term: %d | current State: %d | recieved term: %d\n",
			rf.Me, args.CandidateId, rf.CurrentTerm, rf.State, args.Term)
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
	if args.Term < rf.CurrentTerm {
		reply.Term = rf.CurrentTerm
		reply.Success = false
		DPrintf("[AppendEntries| small term] raft %d reject append entries | current term: %d | current State: %d | received term: %d\n",
			rf.Me, rf.CurrentTerm, rf.State, args.Term)
		return
	}

	// do not change voteFor except step down
	if args.Term == rf.CurrentTerm {
		if rf.State == CANDIDATE {
			rf.State = FOLLOWER
		}

		rf.electionTimerReset()
	}

	// get higher term, convert to follower and match the term
	if args.Term > rf.CurrentTerm {
		rf.convertTo(FOLLOWER)
		rf.CurrentTerm = args.Term
		DPrintf("[AppendEntries| big term or has leader] raft %d update term or State | current term: %d | current State: %d | recieved term: %d\n",
			rf.Me, rf.CurrentTerm, rf.State, args.Term)
	}
	// update serverInfo when any variable changes
	rf.updateServerInfo()
	//rf.InfoCh <- true
	rf.persist()

	//// get higher term, convert to follower and match the term
	//if args.Term >= rf.CurrentTerm {
	//	rf.convertTo(FOLLOWER)
	//	rf.CurrentTerm = args.Term
	//	rf.persist()
	//	DPrintf("[AppendEntries| big term or has leader] raft %d update term or State | current term: %d | current State: %d | recieved term: %d\n",
	//		rf.Me, rf.CurrentTerm, rf.State, args.Term)
	//}

	//smaller length or unmatched term at PrevLogIndex
	if len(rf.log) <= args.PrevLogIndex || rf.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		DPrintf("[AppendEntries| inconsistent log] raft %d reject append entry | log len: %d | args.PrevLogIndex: %d | args.prevLogTerm %d\n",
			rf.Me, len(rf.log), args.PrevLogIndex, args.PrevLogTerm)
		reply.Term = rf.CurrentTerm
		reply.Success = false
		if len(rf.log) <= args.PrevLogIndex {
			reply.ConflictIndex = len(rf.log)
		} else {
			// search for the ConflictIndex forward
			for i := args.PrevLogIndex; i > 0; i-- {
				if rf.log[i].Term != rf.log[i-1].Term {
					reply.ConflictIndex = i
					break
				}
			}
		}
	} else {
		including := true
		nextIndex := args.PrevLogIndex + 1
		conflictIndex := 0
		logLen := len(rf.log)
		entLen := len(args.Entries)
		// check the including of current new log entries
		for i := 0; i < entLen; i++ {
			if ((logLen - 1) < (nextIndex + i)) || rf.log[nextIndex+i].Term != args.Entries[i].Term {
				including = false
				conflictIndex = i
				break
			}
		}
		if !including {
			// can not directly call append() since there will be a data race
			//newLog := make([]LogEntry, 0, len(rf.log[:nextIndex+conflictIndex]))
			//newLog = append(newLog, rf.log[:nextIndex+conflictIndex]...)
			//newLog = append(rf.log[:nextIndex+conflictIndex], args.Entries[conflictIndex:]...)
			newEntries := make([]LogEntry, len(args.Entries[conflictIndex:]))
			copy(newEntries, args.Entries[conflictIndex:])
			rf.log = append(rf.log[:nextIndex+conflictIndex], newEntries...)
			//fmt.Printf("Rf log after append: %v\n", rf.log)
			fmt.Printf(fmt.Sprintf("Changed log%v\n", newEntries))
			// Serverlog update
			rf.updateServerLogs(fmt.Sprintf("%v\n", newEntries))
			//rf.log = newLog
			rf.persist()
			rf.updateConsoleLogs(DLog("Raft Server[%v]: Receive %v Rrom Leader | Log Length: %d",
				rf.Me+1, newEntries, len(rf.log)))
			DPrintf("[AppendEntries] raft %d appended entries from leader | log length: %d\n", rf.Me, len(rf.log))
		}
		lastNewEntryIndex := args.PrevLogIndex + entLen
		if args.LeaderCommit > rf.CommitIndex {
			rf.CommitIndex = min(args.LeaderCommit, lastNewEntryIndex)

			// update server info
			rf.updateServerInfo()
			//rf.InfoCh <- true

			// apply entries after update CommitIndex
			rf.applyCond.Broadcast()
		}
		reply.Term = rf.CurrentTerm
		reply.Success = true
	}
}

// apply entries and set CommandValid to true
//func (rf *Raft) applyEntries() {
//	rf.mu.Lock()
//	defer rf.mu.Unlock()
//	// if CommitIndex > LastApplied: increment LastApplied, apply log[LastApplied] to State machine
//	for i := rf.LastApplied + 1; i <= rf.CommitIndex; i++ {
//		applyMsg := ApplyMsg{
//			CommandValid: true,
//			Command:      rf.log[i].Command,
//			CommandIndex: i,
//		}
//		rf.applyCh <- applyMsg
//		rf.LastApplied += 1
//		DPrintf("[applyEntries] raft %d applied entry | LastApplied: %d | CommitIndex: %d\n",
//			rf.Me, rf.LastApplied, rf.CommitIndex)
//	}
//}

// long-running goroutine function, which keep applying new entry to application
func (rf *Raft) applyEntries() {
	for {
		rf.mu.Lock()
		commitIndex := rf.CommitIndex
		lastApplied := rf.LastApplied
		DPrintf("[applyEntries]: Id %d Term %d State %d\t||\tlastApplied %d and CommitIndex %d\n",
			rf.Me, rf.CurrentTerm, rf.State, lastApplied, commitIndex)
		rf.mu.Unlock()

		if lastApplied == commitIndex {
			rf.mu.Lock()
			rf.applyCond.Wait()
			rf.mu.Unlock()
		} else {
			for i := lastApplied + 1; i <= commitIndex; i++ {
				rf.mu.Lock()
				applyMsg := ApplyMsg{
					CommandValid: true,
					Command:      rf.log[i].Command,
					CommandIndex: i,
				}
				rf.LastApplied = i
				// update server info
				rf.updateServerInfo()
				// update server apply command
				rf.updateServerApplies(applyMsg.CommandValid, applyMsg.Command, applyMsg.CommandIndex)

				DPrintf("[applyEntries]: Id %d Term %d State %d\t||\tapply command %v of index %d and term %d to applyCh\n",
					rf.Me, rf.CurrentTerm, rf.State, applyMsg.Command, applyMsg.CommandIndex, rf.log[i].Term)
				rf.updateConsoleLogs(DLog("Raft Server[%v]: Command[%v] Successful Apply, commit now: %v",
					rf.Me+1, lastApplied+1, applyMsg.Command))
				rf.mu.Unlock()
				rf.applyCh <- applyMsg

			}
		}
	}
}

// State conversion, should be within writeLock
func (rf *Raft) convertTo(state int) {
	oldState := rf.State
	newState := state
	switch state {
	case FOLLOWER:
		rf.electionTimerReset()
		rf.VotedFor = -1
		rf.State = FOLLOWER
	case CANDIDATE:
		rf.electionTimerReset()
		rf.CurrentTerm++
		rf.VotedFor = rf.Me
		rf.State = CANDIDATE
	case LEADER:
		// broadcast includes heartbeat and appendEntries
		rf.broadcastTimerReset()
		rf.State = LEADER
		rf.updateConsoleLogs(DLog("Raft Server[%v]: I am Leader | current term: %d",
			rf.Me+1, rf.CurrentTerm))
	}

	// update serverInfo when any variable changes
	rf.updateServerInfo()
	//rf.InfoCh <- true

	// send signal to awake timer
	if oldState == LEADER && newState == FOLLOWER {
		rf.nonLeaderCond.Broadcast()
	} else if oldState == CANDIDATE && newState == LEADER {
		rf.leaderCond.Broadcast()
	}
}

// Start take a command as input and
//return the index of next logEntry, current term and if this server believes it is the leader
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true
	if term, isLeader = rf.GetState(); isLeader {
		rf.mu.Lock()
		rf.log = append(rf.log, LogEntry{Command: command, Term: rf.CurrentTerm})
		rf.updateServerLogs(fmt.Sprintf("%v", LogEntry{Command: command, Term: rf.CurrentTerm}))
		rf.persist()
		rf.matchIndex[rf.Me] = len(rf.log) - 1
		index = len(rf.log) - 1
		rf.updateConsoleLogs(DLog("Raft Server[%v]: Command[%v] Replicated | current term: %d",
			rf.Me+1, rf.matchIndex[rf.Me], rf.CurrentTerm))
		DPrintf("[Start] raft %d replicate command %v to log | current term: %d | current State: %d | log length: %d\n",
			rf.Me, command, rf.CurrentTerm, rf.State, len(rf.log))
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
	persister *Persister, applyCh chan ApplyMsg, consoleBinding binding.ExternalStringList) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.Me = me

	rf.State = FOLLOWER
	rf.CurrentTerm = 0
	rf.VotedFor = -1

	rf.broadcastSignalChan = make(chan bool)
	rf.electionSignalChan = make(chan bool)
	rf.broadcastTimeout = BroadcastTimeout
	rf.electionTimerReset()

	rf.log = append(rf.log, LogEntry{Term: 0})

	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	rf.CommitIndex = 0
	rf.LastApplied = 0
	rf.applyCh = applyCh
	rf.consoleLogs = consoleBinding
	//rf.InfoCh = make(chan bool)

	info := make([]string, 5)

	// init server log
	rf.ServerLog = binding.BindStringList(
		&[]string{},
	)

	// init server apply
	rf.ServerApply = binding.BindStringList(
		&[]string{},
	)

	//rf.InfoCh <- true

	rf.applyCond = sync.NewCond(&rf.mu)
	rf.nonLeaderCond = sync.NewCond(&rf.mu)
	rf.leaderCond = sync.NewCond(&rf.mu)

	// bootstrap from a persisted State
	rf.ServerInfo = binding.BindStringList(&info)
	rf.readPersist(persister.ReadRaftState())
	rf.updateServerInfo()

	DPrintf("Starting raft %d\n", me)
	// do correspond action according to the channel
	go rf.mainLoop()
	// used when raft server is not a leader
	go rf.electionTimer()
	// used when raft server is a leader
	go rf.broadcastTimer()
	// long-running function to apply new entry
	go rf.applyEntries()

	return rf
}

func (rf *Raft) updateServerInfo() {
	switch rf.State {
	case 0:
		err := rf.ServerInfo.SetValue(0, "Follower")
		if err != nil {
			fmt.Println("err state0")
		}
	case 1:
		err := rf.ServerInfo.SetValue(0, "Candidate")
		if err != nil {
			fmt.Println("err state1")
		}
	case 2:
		err := rf.ServerInfo.SetValue(0, "Leader")
		if err != nil {
			fmt.Println("err state2")
		}
	}
	err1 := rf.ServerInfo.SetValue(1, strconv.Itoa(rf.CurrentTerm))
	if err1 != nil {
		fmt.Println(err1)
	}
	err2 := rf.ServerInfo.SetValue(2, strconv.Itoa(rf.VotedFor+1))
	if err2 != nil {
		fmt.Println(err2)
	}
	err3 := rf.ServerInfo.SetValue(3, strconv.Itoa(rf.CommitIndex))
	if err3 != nil {
		fmt.Println(err3)
	}
	err4 := rf.ServerInfo.SetValue(4, strconv.Itoa(rf.LastApplied))
	if err4 != nil {
		fmt.Println(err4)
	}

	//if rf.Me == 4 {
	//	v0, _ := rf.ServerInfo.GetValue(0)
	//	v1, _ := rf.ServerInfo.GetValue(1)
	//	v2, _ := rf.ServerInfo.GetValue(2)
	//	v3, _ := rf.ServerInfo.GetValue(3)
	//	v4, _ := rf.ServerInfo.GetValue(4)
	//	fmt.Println("updateServerInfo", v0, v1, v2, v3, v4)
	//}
	//err := rf.ServerInfo.Reload()
	//if err != nil {
	//	return
	//}
}


func (rf *Raft) updateConsoleLogs(newLog string) {
	// rf.consoleLogs = append(rf.consoleLogs, newLog+"\n")
	rf.consoleLogs.Append(newLog + "\n")
}

func (rf *Raft) updateServerLogs(log string) {
	rf.ServerLog.Append(log + "\n")
	//results, _ := rf.ServerLog.Get()
	//fmt.Printf("Demo log: %v\n", results)
}

func (rf *Raft) updateServerApplies(a ...interface{}) {
	rf.ServerApply.Append(fmt.Sprintf("%v: [%v] commit index: [%v]\n", a...))
	//results, _ := rf.ServerApply.Get()
	//fmt.Printf("Demo Apply: %v\n", results)
}
