package main

import (
	"GoPaddle-Raft/application"
	"GoPaddle-Raft/kvraft"
	"GoPaddle-Raft/models"
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

func Put(cfg *application.Config, ck *kvraft.Clerk, key string, value string, log *OpLog, cli int) {
	start := int64(time.Since(t0))
	ck.Put(key, value)
	end := int64(time.Since(t0))
	cfg.Op()
	if log != nil {
		log.Append(porcupine.Operation{
			Input:    models.KvInput{Op: 1, Key: key, Value: value},
			Output:   models.KvOutput{},
			Call:     start,
			Return:   end,
			ClientId: cli,
		})

	}
}

// get/put/putappend that keep counts
func Get(cfg *application.Config, ck *kvraft.Clerk, key string, log *OpLog, cli int) string {
	start := int64(time.Since(t0))
	v := ck.Get(key)
	end := int64(time.Since(t0))
	cfg.Op()
	if log != nil {
		log.Append(porcupine.Operation{
			Input:    models.KvInput{Op: 0, Key: key},
			Output:   models.KvOutput{Value: v},
			Call:     start,
			Return:   end,
			ClientId: cli,
		})
	}

	return v
}

func main() {
	var nServers string
	fmt.Println("Enter the number of servers: ")
	fmt.Scanln(&nServers)
	num, _ := strconv.Atoi(nServers)
	manager := &application.Manager{}
	manager.StartSevers(num, true)
	manager.ShowServerInfo()
	var operation string
	for operation != "exit" {
		fmt.Println("Enter your command: ")
		fmt.Scanln(&operation)
		//var key, value string
		//if operation == "put" {
		//	fmt.Println("Enter key: ")
		//	fmt.Scanln(&key)
		//	fmt.Println("Enter value: ")
		//	fmt.Scanln(&value)
		//	Put(cfg, ck, key, value, opLog, 1)
		//}
		//if operation == "get" {
		//	fmt.Println("Enter key: ")
		//	fmt.Scanln(&key)
		//	v := Get(cfg, ck, key, opLog, 1)
		//	fmt.Printf("Value is: %v\n", v)
		//}
		//if operation == "show" {
		//	fmt.Println("Database: ")
		//	for _, server := range cfg.Kvservers {
		//		for k, v := range server.KvStore {
		//			fmt.Printf("%v : %v \n", k, v)
		//		}
		//	}
		//}
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

		if operation == "reconnect" {
			var serverid string
			fmt.Println("Enter Server ID: ")
			fmt.Scanln(&serverid)
			id, _ := strconv.Atoi(serverid)
			manager.Reconnect(id)
		}

	}

}
