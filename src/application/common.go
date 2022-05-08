package application

//  Err reply for client
const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongLeader = "ErrWrongLeader"
)

type Err string

// Put or Append Args
type PutAppendArgs struct {
	Key   string
	Value string
	Op    string // "Put" or "Append"
	// Field names must start with capital letters,
	// otherwise RPC will break.
	ClientId  int64
	RequestId int
}

//  Put or Append Reply
type PutAppendReply struct {
	Err Err
}

//  Get Args
type GetArgs struct {
	Key       string
	ClientId  int64
	RequestId int
}

//  Get Args
type GetReply struct {
	Err   Err
	Value string
}
