package application

import (
	"GoPaddle-Raft/kvraft"
	"GoPaddle-Raft/porcupine"
	"fmt"
	"math/rand"
	"sync"
)

type OpLog struct {
	operations []porcupine.Operation
	sync.Mutex
}

type Manager struct {
	Cfg   *Config
	OpLog *OpLog
	Clerk *kvraft.Clerk
}

func (manager *Manager) StartSevers(num int, unreliable bool) {
	// create config
	manager.Cfg = make_config(num, unreliable, -1)
	// initialize an empty operation log
	manager.OpLog = &OpLog{}
	// initialize a Clerk with Clerk specific server names.
	manager.Clerk = manager.Cfg.MakeClient(manager.Cfg.All())
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
	for _, server := range manager.Cfg.kvservers {
		if server != nil {
			raftState := server.rf
			fmt.Printf("Server[%v]: State [%v], Current Term [%v]\n", raftState.Me, raftState.State, raftState.CurrentTerm)
			fmt.Println("Log:")
			for _, log := range manager.OpLog.operations {
				fmt.Printf("%v; ", log)
				fmt.Println()
			}
			fmt.Println("---------------")
		} else {
			// action when the server is shutdown
		}

	}

}
