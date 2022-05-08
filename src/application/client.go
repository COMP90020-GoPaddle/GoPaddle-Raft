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

package application

import (
	"sync/atomic"
	"time"

	"GoPaddle-Raft/labrpc"
)

type Clerk struct {
	Servers       []*labrpc.ClientEnd
	leaderId      int
	clientId      int64
	lastRequestId int
	timeoutId     int
}

var Cid int64 = 0

func makeCid() int64 {
	atomic.AddInt64(&Cid, 1)
	return Cid
}

// set random cilentId
func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.Servers = servers
	ck.clientId = makeCid()
	return ck
}

// Get method, it will call RPC Get function in the server
func (ck *Clerk) Get(key string) string {
	// requestId ready to update
	updatedRequestId := ck.lastRequestId + 1
	if ck.timeoutId >= updatedRequestId {
		updatedRequestId = ck.timeoutId + 1
	}

	DPrintf("IDs: TimeOut ID:%v | LastID:%v | NowID:%v", ck.timeoutId, ck.lastRequestId, updatedRequestId)

	args := GetArgs{
		Key:       key,
		ClientId:  ck.clientId,
		RequestId: updatedRequestId,
	}
	DPrintf("Client[%d], Request[%d] Get, Key=%s ", ck.clientId, updatedRequestId, key)
	cnt := 0
	for {
		if cnt >= len(ck.Servers) {
			ck.timeoutId = updatedRequestId
			return "Timeout"
		}
		savedLeaderId := ck.leaderId % len(ck.Servers)
		// make a new reply in every loop
		var reply GetReply

		// RPC Call
		if ck.Servers[savedLeaderId].Call("KVServer.Get", &args, &reply) {
			// Success or No Key -> update request id, return GET value
			if reply.Err == OK {
				DPrintf("Get Success: %s", reply.Value)
				ck.lastRequestId = updatedRequestId
				return reply.Value
			} else if reply.Err == ErrNoKey {
				DPrintf("Get Success, but key not found")
				ck.lastRequestId = updatedRequestId
				return "Key no found"
			}
		}
		// Fail -> leaderId + 1 and retry
		DPrintf("Wrong leader[%d], try another one", savedLeaderId)
		ck.leaderId = (savedLeaderId + 1) % len(ck.Servers)
		cnt++
		time.Sleep(300 * time.Millisecond)
	}
}

//	Shared by Put and Append, it will call RPC PutAppend function in the server
func (ck *Clerk) PutAppend(key string, value string, op string) string {
	// requestId ready to update
	updatedRequestId := ck.lastRequestId + 1
	if ck.timeoutId >= updatedRequestId {
		updatedRequestId = ck.timeoutId + 1
	}

	DPrintf("IDs: TimeOut ID:%v | LastID:%v | NowID:%v", ck.timeoutId, ck.lastRequestId, updatedRequestId)
	args := PutAppendArgs{
		Key:       key,
		Value:     value,
		Op:        op,
		ClientId:  ck.clientId,
		RequestId: updatedRequestId,
	}
	DPrintf("Client[%d], Request[%d] PutAppend, Key=%s Value=%s", ck.clientId, updatedRequestId, key, value)
	cnt := 0
	for {
		if cnt >= len(ck.Servers) {
			ck.timeoutId = updatedRequestId
			DPrintf("%v Timeout", op)
			return "Timeout"
		}
		savedLeaderId := ck.leaderId % len(ck.Servers)
		// make a new reply in every loop
		var reply PutAppendReply

		// RPC Call
		if ck.Servers[savedLeaderId].Call("KVServer.PutAppend", &args, &reply) {
			// Success -> update requestId, no value return
			if reply.Err == OK {
				DPrintf("%v Success", op)
				ck.lastRequestId = updatedRequestId
				return "OK"
			}
		}

		// Fail -> leaderId + 1 and retry
		DPrintf("Wrong leader[%d], try another one", savedLeaderId)
		ck.leaderId = (savedLeaderId + 1) % len(ck.Servers)
		cnt++
		time.Sleep(300 * time.Millisecond)
	}
}

// Put method, will call PutAppends
func (ck *Clerk) Put(key string, value string) string {
	return ck.PutAppend(key, value, "Put")
}

// Append method, will call PutAppend
func (ck *Clerk) Append(key string, value string) string {
	return ck.PutAppend(key, value, "Append")
}
