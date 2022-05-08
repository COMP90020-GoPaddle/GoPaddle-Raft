package application

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"GoPaddle-Raft/labgob"
	"GoPaddle-Raft/labrpc"
	"GoPaddle-Raft/raft"

	"fyne.io/fyne/v2/data/binding"
)

const Debug = true

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
	// Field names must start with capital letters,
	// otherwise RPC will break.
	Key       string
	Value     string
	Name      string
	ClientId  int64
	RequestId int
}

type KVServer struct {
	mu      sync.Mutex
	me      int // the id of current server
	Rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big

	kvStore    map[string]string
	requestMap map[int64]int
	dispatcher map[int]chan Notification

	disconn     bool // record whether it's been disconnected
	ServerStore binding.ExternalStringList
}

// show kv store of one server
func (kv *KVServer) ShowDB() map[string]string {
	return kv.kvStore
}

// RPC Get
func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	operation := Op{
		Key:       args.Key,
		Name:      "Get",
		ClientId:  args.ClientId,
		RequestId: args.RequestId,
	}
	DPrintf("Client[%v] wants to Get %v from %v", args.ClientId, args.Key, kv.me)

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

// Check whether the requests are duplicate compared to the records
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

// Send operation to raft and wait
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
			kv.updateServerStore()
		case "Append":
			kv.kvStore[operation.Key] += operation.Value
			kv.updateServerStore()

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

//  Start a kv server, including making a new raft instance
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

	// For demo app: init server store as a binding string
	kv.ServerStore = binding.BindStringList(
		&[]string{},
	)

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.Rf = raft.Make(servers, me, persister, kv.applyCh, consoleBinding)

	DPrintf("Server Start: %v", kv.me)
	// You may need initialization code here.
	go kv.Listener()

	return kv
}

//  Update binding data for GUI to display
func (kv *KVServer) updateServerStore() {
	kvPair := make([]string, 0)
	keys := make([]string, 0)
	for k, _ := range kv.kvStore {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		kvPair = append(kvPair, fmt.Sprintf("[%v]:[%v]\n", k, kv.kvStore[k]))
	}

	err := kv.ServerStore.Set(kvPair)
	if err != nil {
		//fmt.Println("Here", err)
		return
	}
}
