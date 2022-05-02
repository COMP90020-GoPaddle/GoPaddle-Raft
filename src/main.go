package main

import (
	"GoPaddle-Raft/application"
	"GoPaddle-Raft/kvraft"
	"GoPaddle-Raft/models"
	"GoPaddle-Raft/porcupine"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"strconv"
	"sync"
	"time"
)

func updateTime(clock *widget.Label, command <-chan string) {
	for {
		select {
		case cmd := <-command:
			fmt.Println(cmd)
			switch cmd {
			case "Pause":
				return
			case "Play":
				for range time.Tick(time.Second) {
					formatted := time.Now().Format("Time: 03:04:05")
					clock.SetText(formatted)
				}
			}
		default:
			return
		}

	}
}

func makeUI() (*widget.Label, *widget.Entry) {
	out := widget.NewLabel("Hello ")
	in := widget.NewEntry()

	in.OnChanged = func(s string) {
		out.SetText("Hello " + s)
	}
	return out, in
}

func makeServerUI(main *fyne.Container, count *int) *fyne.Container {
	serverTitle := widget.NewLabel("Raft Server" + strconv.Itoa(*count))
	*count++
	sep := widget.NewSeparator()
	clock := widget.NewLabel("")
	clock.SetText(time.Now().Format("Time: 03:04:05"))
	command := make(chan string)
	status := "Pause"
	showBtn := widget.NewButton("show log", func() {
		go updateTime(clock, command)
		if status == "Pause" {
			status = "Play"
		} else {
			status = "Pause"
		}
		command <- status
	})
	discBtn := widget.NewButton("disconnect", func() {
	})
	delBtn := widget.NewButton("delete", func() {
		main.Objects = append(main.Objects[:3+*count-1], main.Objects[3+*count:]...)
		*count--
		main.Refresh()
	})

	vbox := container.NewVBox(layout.NewSpacer(), serverTitle, sep, clock, showBtn, discBtn, delBtn, layout.NewSpacer())
	return vbox
}

func makeClientUI() {

}

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

func showServerInfo(cfg *application.Config) {
	for _, server := range cfg.Kvservers {
		raftState := server.Rf
		fmt.Printf("Server[%v]: State [%v], Current Term [%v]\n", raftState.Me, raftState.State, raftState.CurrentTerm)
		fmt.Println("Log:")
		for _, log := range raftState.Log {
			fmt.Printf("%v; ", log)
			fmt.Println()
		}
		fmt.Println("---------------")
	}

}

func main() {
	//a := app.New()
	//win := a.NewWindow("GoPaddle")
	//win.Resize(fyne.NewSize(500, 600))
	//baseContainer := container.NewHBox(layout.NewSpacer())
	//
	//title := canvas.NewText("GoPaddle", color.Black)
	//title.TextSize = 50
	//
	//count := 0
	//serverBtn := widget.NewButton("New Server", func() {
	//	baseContainer.Add(makeServerUI(baseContainer, &count))
	//	fmt.Println(len(baseContainer.Objects))
	//})
	//
	//clientBtn := widget.NewButton("New Client", func() {
	//
	//})
	//
	//exitBtn := widget.NewButton("EXIT", func() {
	//
	//})
	//
	//controlBox := container.NewVBox(layout.NewSpacer(), title, serverBtn, clientBtn, exitBtn, layout.NewSpacer())
	//baseContainer.Add(controlBox)
	//baseContainer.Add(layout.NewSpacer())
	//fmt.Println(len(baseContainer.Objects))
	//win.SetContent(baseContainer)
	//
	//win.Show()
	//a.Run()
	var nServers string
	fmt.Println("Enter the number of servers: ")
	fmt.Scanln(&nServers)
	num, _ := strconv.Atoi(nServers)
	cfg := application.Make_config(num, true, -1)
	showServerInfo(cfg)
	opLog := &OpLog{}
	ck := cfg.MakeClient(cfg.All())
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
			Put(cfg, ck, key, value, opLog, 1)
		}
		if operation == "get" {
			fmt.Println("Enter key: ")
			fmt.Scanln(&key)
			v := Get(cfg, ck, key, opLog, 1)
			fmt.Printf("Value is: %v\n", v)
		}
		if operation == "show" {
			fmt.Println("Database: ")
			for _, server := range cfg.Kvservers {
				for k, v := range server.KvStore {
					fmt.Printf("%v : %v \n", k, v)
				}
			}
		}
		if operation == "info" {
			showServerInfo(cfg)
		}

		if operation == "shutdown" {
			for i := 0; i < num; i++ {
				cfg.ShutdownServer(i)
			}
		}

		if operation == "restart" {
			// crash and re-start all
			for i := 0; i < num; i++ {
				cfg.StartServer(i)
			}
		}

		if operation == "reconnect" {
			cfg.ConnectAll()
		}

	}

}
