package application

import (
	"GoPaddle-Raft/raft"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Manager struct {
	Cfg *Config
}

func (manager *Manager) StartSevers(num int, unreliable bool) {
	// create config
	manager.Cfg = Make_config(num, unreliable, -1)
	// initialize an empty operation log
	//manager.OpLog = &OpLog{}
	// initialize a Clerk with Clerk specific server names.
	//manager.Clerk = manager.Cfg.makeClient(manager.Cfg.All())
}

func (manager *Manager) ShutDown(serverID int) {
	manager.Cfg.ShutdownServer(serverID)
}

func (manager *Manager) Disconnect(serverID int) {
	pa := make([][]int, 2)
	pa[0] = make([]int, 0)
	pa[1] = make([]int, 0)
	for j := 0; j < manager.Cfg.n; j++ {
		if j == serverID {
			pa[0] = append(pa[0], j)
		} else {
			pa[1] = append(pa[1], j)
		}
	}
	manager.Cfg.partition(pa[0], pa[1])
}

func (manager *Manager) Restart(serverID int) {
	manager.Cfg.StartServer(serverID)
}

func (manager *Manager) Reconnect(serverID int) {
	manager.Cfg.connectUnlocked(serverID, manager.Cfg.All())
}

func (manager *Manager) ReconnectAll() {
	manager.Cfg.ConnectAll()
}

func (manager *Manager) MakePartition() {
	a := make([]int, manager.Cfg.n)
	for i := 0; i < manager.Cfg.n; i++ {
		a[i] = (rand.Int() % 2)
	}
	pa := make([][]int, 2)
	for i := 0; i < 2; i++ {
		pa[i] = make([]int, 0)
		for j := 0; j < manager.Cfg.n; j++ {
			if a[j] == i {
				pa[i] = append(pa[i], j)
			}
		}
	}
	manager.Cfg.partition(pa[0], pa[1])
}

func (manager *Manager) ShowServerInfo() {
	fmt.Printf("Total Server num %v\n", manager.Cfg.n)
	for _, server := range manager.Cfg.Kvservers {
		if server != nil {
			raftState := server.Rf
			fmt.Printf("Server[%v]: State [%v], Current Term [%v]\n", raftState.Me, raftState.State, raftState.CurrentTerm)
			//fmt.Println("Log:")
			//for _, log := range manager.OpLog.operations {
			//	fmt.Printf("%v; ", log)
			//	fmt.Println()
			//}
			fmt.Println("---------------")
		} else {
			// action when the server is shutdown
		}

	}
}

func (manager *Manager) ShowSingleServer(rf *raft.Raft) {
	fmt.Printf("Server[%v]: State [%v], voteFor [%v] Current Term [%v]\n", rf.Me, rf.State, rf.VotedFor, rf.CurrentTerm)
	fmt.Println("Log:")
	//for _, log := range manager.OpLog.operations {
	//	fmt.Printf("%v; ", log)
	//	fmt.Println()
	//}
	fmt.Println("---------------")

}

func (manager *Manager) GetAllServers() []*KVServer {
	return manager.Cfg.Kvservers
}

/** Client APIs **/

// OpLog use by recoding client's operation
type OpLog struct {
	operations []Operation
	sync.Mutex
}

type Operation struct {
	ClientId int64
	Input    interface{}
	Call     time.Time // invocation time
	Output   interface{}
	Return   time.Time // response time
}

type KvInput struct {
	Op    uint8 // 0 => get, 1 => put, 2 => append
	Key   string
	Value string
}

type KvOutput struct {
	Value string
}

func (log *OpLog) Append(op Operation) {
	log.Lock()
	defer log.Unlock()
	log.operations = append(log.operations, op)
}

func (log *OpLog) Read() []Operation {
	log.Lock()
	defer log.Unlock()
	ops := make([]Operation, len(log.operations))
	copy(ops, log.operations)
	return ops
}

// Client Each client's GUI hold one and use it to interact with servers
type Client struct {
	ck  *Clerk
	Log *OpLog
}

// Call by connect button
func (manager *Manager) StartClient() *Client {
	client := &Client{}
	ck := manager.Cfg.MakeClient(manager.Cfg.All())
	opLog := &OpLog{}
	client.ck = ck
	client.Log = opLog
	return client
}

// Call by close client's GUI
// after calling, drop the client's pointer?
func (manager *Manager) CloseClient(client *Client) {
	manager.Cfg.deleteClient(client.ck)
}

func (client *Client) Get(cfg *Config, key string) string {
	start := time.Now()
	v := client.ck.Get(key)
	end := time.Now()
	cfg.op()
	if client.Log != nil {
		client.Log.Append(Operation{
			Input:    KvInput{Op: 0, Key: key},
			Output:   KvOutput{Value: v},
			Call:     start,
			Return:   end,
			ClientId: client.ck.clientId,
		})
	}
	return v
}

func (client *Client) Put(cfg *Config, key string, value string) {
	start := time.Now()
	client.ck.Put(key, value)
	end := time.Now()
	cfg.op()
	if client.Log != nil {
		client.Log.Append(Operation{
			Input:    KvInput{Op: 1, Key: key, Value: value},
			Output:   KvOutput{},
			Call:     start,
			Return:   end,
			ClientId: client.ck.clientId,
		})
	}
}

func (client *Client) Append(cfg *Config, key string, value string) {
	start := time.Now()
	client.ck.Append(key, value)
	end := time.Now()
	cfg.op()
	if client.Log != nil {
		client.Log.Append(Operation{
			Input:    KvInput{Op: 2, Key: key, Value: value},
			Output:   KvOutput{},
			Call:     start,
			Return:   end,
			ClientId: client.ck.clientId,
		})
	}
}
