package kvraft

import (
	"crypto/rand"
	"math/big"
	"time"

	"GoPaddle-Raft/labrpc"
)

type Clerk struct {
	Servers []*labrpc.ClientEnd
	// You will have to modify this struct.
	leaderId      int
	clientId      int64
	lastRequestId int
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

// set random cilentId
func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.Servers = servers
	ck.clientId = nrand()
	// ck.
	// You'll have to add code here.
	return ck
}

//
// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) Get(key string) string {
	// You will have to modify this function.
	// requestId ready to update
	updatedRequestId := ck.lastRequestId + 1

	args := GetArgs{
		Key:       key,
		ClientId:  ck.clientId,
		RequestId: updatedRequestId,
	}
	DPrintf("Client[%d], Request[%d] Get, Key=%s ", ck.clientId, updatedRequestId, key)
	for {
		savedLeaderId := ck.leaderId
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
				return ""
			}
		}
		// Fail -> leaderId + 1 and retry
		DPrintf("Wrong leader[%d], try another one", savedLeaderId)
		ck.leaderId = (savedLeaderId + 1) % len(ck.Servers)
		time.Sleep(10 * time.Millisecond)
	}
}

//
// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) PutAppend(key string, value string, op string) {
	// You will have to modify this function.
	// requestId ready to update
	updatedRequestId := ck.lastRequestId + 1
	args := PutAppendArgs{
		Key:       key,
		Value:     value,
		Op:        op,
		ClientId:  ck.clientId,
		RequestId: updatedRequestId,
	}
	DPrintf("Client[%d], Request[%d] PutAppend, Key=%s Value=%s", ck.clientId, updatedRequestId, key, value)
	for {
		savedLeaderId := ck.leaderId
		// make a new reply in every loop
		var reply PutAppendReply

		// RPC Call
		if ck.Servers[savedLeaderId].Call("KVServer.PutAppend", &args, &reply) {
			// Success -> update requestId, no value return
			if reply.Err == OK {
				DPrintf("%v Success", op)
				ck.lastRequestId = updatedRequestId
				return
			}
		}

		// Fail -> leaderId + 1 and retry
		DPrintf("Wrong leader[%d], try another one", savedLeaderId)
		ck.leaderId = (savedLeaderId + 1) % len(ck.Servers)
		time.Sleep(10 * time.Millisecond)
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
