package application

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"GoPaddle-Raft/labgob"
	"GoPaddle-Raft/labrpc"
	"GoPaddle-Raft/raft"

	"fyne.io/fyne/v2/data/binding"
)

const Debug = true
const Demo = true

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type Notification struct {
	ClientId  int64
	RequestId int
}

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	Key       string
	Value     string
	Name      string
	ClientId  int64
	RequestId int
}

type KVServer struct {
	mu           sync.Mutex
	me           int // the id of current server
	Rf           *raft.Raft
	applyCh      chan raft.ApplyMsg
	consoleLogCh chan string
	dead         int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	kvStore    map[string]string
	requestMap map[int64]int
	dispatcher map[int]chan Notification

	disconn bool // record where it's been disconnected
}

func (kv *KVServer) ShowDB() map[string]string {
	return kv.kvStore
}

// RPC Get
func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	operation := Op{
		Key:       args.Key,
		Name:      "Get",
		ClientId:  args.ClientId,
		RequestId: args.RequestId,
	}
	DPrintf("Client[%v] wants to Get %v from %v", args.ClientId, args.Key, kv.me)
	fmt.Printf("Client[%v] wants to Get %v from %v\n", args.ClientId, args.Key, kv.me)

	// Return True -> Get request fails
	if kv.waitRaft(operation) {
		reply.Err = ErrWrongLeader
		DPrintf("Operation: %v", reply.Err)
	} else {
		kv.mu.Lock()
		// Search key in KV-database
		value, ok := kv.kvStore[operation.Key]
		kv.mu.Unlock()

		// Find key -> reply value
		if ok {
			reply.Err = OK
			reply.Value = value
			DPrintf("Operation: %v", reply.Err)
			return
		}

		// No key -> reply Err
		reply.Err = ErrNoKey
		DPrintf("Operation: %v", reply.Err)
	}
}

// RPC PutAppend
func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	operation := Op{
		Key:       args.Key,
		Value:     args.Value,
		Name:      args.Op,
		ClientId:  args.ClientId,
		RequestId: args.RequestId,
	}

	if kv.waitRaft(operation) {
		reply.Err = ErrWrongLeader
	} else {
		reply.Err = OK
	}
}

func (kv *KVServer) isDuplicate(clientId int64, requestId int) bool {
	oldRequestId, ok := kv.requestMap[clientId]

	// RequestId already in requestMap -> check if it is larger than the old one
	// Not in requestMap -> not duplicate
	if !ok || oldRequestId < requestId {
		DPrintf("OK[%v], Not Duplicate: Old request[%v] from Client[%v], new request:[%v]", ok, oldRequestId, clientId, requestId)
		return false
	}
	DPrintf("Duplicate: Old request[%v] from Client[%v], new request:[%v]", oldRequestId, clientId, requestId)
	return true
}

// Send operation tp raft and wait
func (kv *KVServer) waitRaft(operation Op) bool {
	// Send to raft
	index, _, isLeader := kv.Rf.Start(operation)
	if !isLeader {
		return true
	}
	wrongLeader := false
	kv.mu.Lock()

	// Create notification channel for this request
	if _, ok := kv.dispatcher[index]; !ok {
		kv.dispatcher[index] = make(chan Notification, 1)
	}
	ch := kv.dispatcher[index]
	kv.mu.Unlock()

	// Wait channel Msg
	select {
	// Receive Msg, check client and request ids
	case notification := <-ch:
		if notification.ClientId != operation.ClientId || notification.RequestId != operation.RequestId {
			DPrintf("Raft Client[%v] -> My Client[%v], Raft Request[%v] -> My Request[%v]",
				notification.ClientId, operation.ClientId, notification.RequestId, operation.RequestId)
			wrongLeader = true
		}
	// Timeout
	case <-time.After(500 * time.Millisecond):
		kv.mu.Lock()
		DPrintf("From Wait Raft...")
		if !kv.isDuplicate(operation.ClientId, operation.RequestId) {
			wrongLeader = true
		}
		kv.mu.Unlock()
	}
	kv.mu.Lock()

	// Remove the notification channel after a request processed
	delete(kv.dispatcher, index)
	kv.mu.Unlock()
	DPrintf("Delete Operation[%d]: %v", index, operation)
	return wrongLeader
}

//
// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
//
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.Rf.Kill()
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

// Receive applyMSG from Raft
func (kv *KVServer) Listener() {
	for applyMsg := range kv.applyCh {
		// Skip not valid Msg
		if !applyMsg.CommandValid {
			continue
		}
		operation := applyMsg.Command.(Op)
		kv.mu.Lock()

		// Skip duplicate Requests
		if kv.isDuplicate(operation.ClientId, operation.RequestId) {
			kv.mu.Unlock()
			continue
		}

		// Update KV-database
		switch operation.Name {
		case "Put":
			kv.kvStore[operation.Key] = operation.Value
		case "Append":
			kv.kvStore[operation.Key] += operation.Value
		}
		DPrintf("ApplyMsg[%v] Operation: %v, Database: %v", applyMsg, operation, kv.kvStore[operation.Key])
		//fmt.Printf("ApplyMsg[%v] Operation: %v, Database: %v\n", applyMsg, operation, kv.kvStore[operation.Key])
		//fmt.Printf("ApplyMsg: %v, Operation: %v\n", applyMsg.Command.(Op), operation)
		DPrintf("Client[%v]: RequestId: %v", operation.ClientId, operation.RequestId)

		// Update requestId
		kv.requestMap[operation.ClientId] = operation.RequestId
		DPrintf("Client[%v]: RequestId: %v", operation.ClientId, kv.requestMap[operation.ClientId])

		// If client is waiting for raft response, send notification to that channel
		if ch, ok := kv.dispatcher[applyMsg.CommandIndex]; ok {
			notification := Notification{
				ClientId:  operation.ClientId,
				RequestId: operation.RequestId,
			}
			ch <- notification
		}
		kv.mu.Unlock()
	}
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int, consoleBinding binding.ExternalStringList) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.
	kv.kvStore = make(map[string]string)
	kv.dispatcher = make(map[int]chan Notification)
	kv.requestMap = make(map[int64]int)

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.Rf = raft.Make(servers, me, persister, kv.applyCh, consoleBinding)

	DPrintf("Server Start: %v", kv.me)
	// You may need initialization code here.
	go kv.Listener()

	return kv
}
