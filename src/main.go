package main

import (
	"GoPaddle-Raft/application"
	"GoPaddle-Raft/porcupine"
	"fmt"
	"strconv"
	"sync"
	"time"
)

func (log *OpLog) Append(op porcupine.Operation) {
	log.Lock()
	defer log.Unlock()
	log.operations = append(log.operations, op)
}

type OpLog struct {
	operations []porcupine.Operation
	sync.Mutex
}

var t0 = time.Now()

//func showServerInfo(cfg *application.Config) {
//	for _, server := range cfg.Kvservers {
//		raftState := server.Rf
//		fmt.Printf("Server[%v]: State [%v], Current Term [%v]\n", raftState.Me, raftState.State, raftState.CurrentTerm)
//		fmt.Println("Log:")
//		for _, log := range raftState.Log {
//			fmt.Printf("%v; ", log)
//			fmt.Println()
//		}
//		fmt.Println("---------------")
//	}
//
//}

func main() {
	var nServers string
	fmt.Println("Enter the number of servers: ")
	fmt.Scanln(&nServers)
	num, _ := strconv.Atoi(nServers)
	manager := &application.Manager{}
	manager.StartSevers(num, true)
	manager.ShowServerInfo()

	client := manager.StartClient()

	//cfg := application.Make_config(num, true, -1)
	//showServerInfo(cfg)
	//opLog := &OpLog{}
	//ck := cfg.MakeClient(cfg.All())
	var operation string
	for operation != "exit" {
		fmt.Println("Enter your command: ")
		fmt.Scanln(&operation)
		var key, value string
		if operation == "put" {
			fmt.Println("Enter key: ")
			fmt.Scanln(&key)
			fmt.Println("Enter value: ")
			fmt.Scanln(&value)
			client.Put(manager.Cfg, key, value)
		}
		if operation == "get" {
			fmt.Println("Enter key: ")
			fmt.Scanln(&key)
			client.Get(manager.Cfg, key)
			fmt.Println("Client log:", client.Log)
		}
		//if operation == "show" {
		//	fmt.Println("Database: ")
		//	for _, server := range cfg.Kvservers {
		//		for k, v := range server.KvStore {
		//			fmt.Printf("%v : %v \n", k, v)
		//		}
		//	}
		//}
		if operation == "log" {
			fmt.Printf("log: %v\n", manager.Cfg.Kvservers[0].Rf.ServerLog)
		}

		if operation == "apply" {
			fmt.Printf("apply: %v\n", manager.Cfg.Kvservers[0].Rf.ServerLog)
		}

		if operation == "info" {
			manager.ShowServerInfo()
		}

		if operation == "shutdown" {
			var serverid string
			fmt.Println("Enter Server ID: ")
			fmt.Scanln(&serverid)
			id, _ := strconv.Atoi(serverid)
			manager.ShutDown(id)
		}
		if operation == "makep" {
			manager.MakePartition()
		}
		if operation == "restart" {
			// crash and re-start all
			//for i := 0; i < num; i++ {
			//	cfg.StartServer(i)
			//}
			var serverid string
			fmt.Println("Enter Server ID: ")
			fmt.Scanln(&serverid)
			id, _ := strconv.Atoi(serverid)
			manager.Restart(id)
		}

		if operation == "disconnect" {
			var serverid string
			fmt.Println("Enter Server ID: ")
			fmt.Scanln(&serverid)
			id, _ := strconv.Atoi(serverid)
			manager.Disconnect(id)
		}

		if operation == "reconnect" {
			var serverid string
			fmt.Println("Enter Server ID: ")
			fmt.Scanln(&serverid)
			id, _ := strconv.Atoi(serverid)
			manager.Reconnect(id)
		}

		if operation == "reconnectAll" {
			manager.ReconnectAll()
		}

	}

}
