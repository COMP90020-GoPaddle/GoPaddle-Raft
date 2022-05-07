package main

import (
	"fmt"
	"fyne.io/fyne/v2/theme"
	"image/color"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"GoPaddle-Raft/application"
)

type serverBox struct {
}

func (sB *serverBox) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(1200, 800)
}

func (sB *serverBox) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	pos := fyne.NewPos(25, 25+containerSize.Height-sB.MinSize(objects).Height)
	objects[0].Move(pos)
}

type clientBox struct {
}

func (cB *clientBox) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(400, 600)
}

func (cB *clientBox) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	pos := fyne.NewPos(25, 25+containerSize.Height-cB.MinSize(objects).Height)
	objects[0].Move(pos)
}

func main() {
	a := app.New()
	a.Settings().SetTheme(theme.DarkTheme())
	manager := &application.Manager{}
	w := a.NewWindow("GoPaddle's Application")
	w.Resize(fyne.NewSize(1440, 800))
	w.SetFixedSize(true)
	w2 := a.NewWindow("New Cluster")
	w2.Resize(fyne.NewSize(220, 240))
	w2.SetFixedSize(true)

	templateStr := "this is a template string to simulate a very long log entry or console output that might appear, if this is not long enough then make it longer :)"
	rect := canvas.NewRectangle(color.White)
	rect.Resize(fyne.NewSize(1150, 750))
	rect.StrokeColor = color.White
	rect.StrokeWidth = 3
	rect.FillColor = color.Transparent
	serverContainer := container.New(&serverBox{}, rect)
	clientArray := make([]binding.ExternalStringList, 10)
	clientCount := 0
	clientConsoleArray := make([]string, 10)
	var serverArray []string
	var btnArray = make([]*widget.Button, 3)
	image := canvas.NewImageFromFile("img.png")
	image.FillMode = canvas.ImageFillOriginal
	text2 := canvas.NewText("Raft Server", color.White)
	text2.Alignment = fyne.TextAlignCenter
	text2.TextSize = 30
	text3 := canvas.NewText("Monitor Platform", color.White)
	text3.Alignment = fyne.TextAlignCenter
	text3.TextSize = 25
	newClusterBtn := widget.NewButton("New Cluster", func() {
		w2.Show()
	})

	newClientBtn := widget.NewButton("New Client", func() {
		serverIndex := clientCount
		clientArray[serverIndex] = binding.BindStringList(&[]string{"Invalid"})
		clientCount++
		w3 := a.NewWindow("New Client")
		w3.Resize(fyne.NewSize(400, 600))
		w3.SetFixedSize(true)
		rect := canvas.NewRectangle(color.White)
		rect.Resize(fyne.NewSize(350, 550))
		rect.StrokeColor = color.White
		rect.StrokeWidth = 3
		rect.FillColor = color.Transparent
		clientContainer := container.New(&clientBox{}, rect)
		str := make([]string, 1)
		str[0] = "Client ID:"
		clientLabel := widget.NewList(
			func() int {
				return len(str)
			},
			func() fyne.CanvasObject {
				return widget.NewLabel("template")
			},
			func(i widget.ListItemID, o fyne.CanvasObject) {
				o.(*widget.Label).SetText(str[i])
			})
		clientLabel.Resize(fyne.NewSize(80, 40))
		clientId := widget.NewListWithData(clientArray[serverIndex],
			func() fyne.CanvasObject {
				return widget.NewLabel("template")
			},
			func(i binding.DataItem, o fyne.CanvasObject) {
				o.(*widget.Label).Bind(i.(binding.String))
			})
		clientId.Resize(fyne.NewSize(160, 40))
		// start client
		var client *application.Client = nil
		connectBtn := widget.NewButton("Connect", func() {
			client = manager.StartClient()
			clientArray[serverIndex].SetValue(0, strconv.Itoa(int(client.Cid)))
		})
		connectBtn.Resize(fyne.NewSize(120, 40))
		commands := widget.NewTextGridFromString(clientConsoleArray[serverIndex])
		commandsScroll := container.NewScroll(commands)
		commandsScroll.Resize(fyne.NewSize(300, 150))
		commandsScroll.ScrollToBottom()

		input := widget.NewEntry()
		input.SetPlaceHolder("\"key,value\" for PUT and \"key\" for GET")
		input.Resize(fyne.NewSize(300, 40))
		var rsp = ""
		responseText := binding.NewString()
		getBtn := widget.NewButton("Get", func() {
			id, _ := clientArray[serverIndex].GetValue(0)
			if id == "Invalid" {
				str, _ := responseText.Get()
				responseText.Set(str + "No connection now!\n")
				clientConsoleArray[serverIndex] += "No connection now!\n"
			} else {
				clientConsoleArray[serverIndex] += "Get " + input.Text
				if input.Text != "" {
					rsp = client.Get(manager.Cfg, input.Text)
					str, _ := responseText.Get()
					responseText.Set(str + rsp + "\n")
					clientConsoleArray[serverIndex] += "\n"
				} else {
					str, _ := responseText.Get()
					responseText.Set(str + "Invalid command\n")
					clientConsoleArray[serverIndex] += "Invalid command\n"
				}
			}
			input.SetPlaceHolder("\"key,value\" for PUT and \"key\" for GET")
			input.SetText("")
			commands.SetText(clientConsoleArray[serverIndex])
			commandsScroll.ScrollToBottom()
		})
		getBtn.Resize(fyne.NewSize(120, 40))
		putBtn := widget.NewButton("Put", func() {
			id, _ := clientArray[serverIndex].GetValue(0)
			if id == "Invalid" {
				str, _ := responseText.Get()
				responseText.Set(str + "No connection now!\n")
				clientConsoleArray[serverIndex] += "No connection now!\n"
			} else {
				clientConsoleArray[serverIndex] += "Put " + input.Text
				s := strings.Split(input.Text, ",")
				if len(s) == 2 {
					rsp := client.Put(manager.Cfg, s[0], s[1])
					clientConsoleArray[serverIndex] += " " + rsp + "\n"
				} else {
					responseText.Set("Invalid command\n")
					clientConsoleArray[serverIndex] += " Invalid command\n"
				}
			}
			input.SetPlaceHolder("\"key,value\" for PUT and \"key\" for GET")
			input.SetText("")
			commands.SetText(clientConsoleArray[serverIndex])
			commandsScroll.ScrollToBottom()
		})
		response := widget.NewEntryWithData(responseText)
		response.Resize(fyne.NewSize(300, 150))
		putBtn.Resize(fyne.NewSize(120, 40))
		clientLabel.Move(fyne.NewPos(35, 35))
		clientId.Move(fyne.NewPos(125, 35))
		connectBtn.Move(fyne.NewPos(140, 80))
		input.Move(fyne.NewPos(50, 140))
		getBtn.Move(fyne.NewPos(60, 200))
		putBtn.Move(fyne.NewPos(220, 200))
		commandsScroll.Move(fyne.NewPos(50, 250))
		response.Move(fyne.NewPos(50, 410))
		clientContainer.Add(clientLabel)
		clientContainer.Add(clientId)
		clientContainer.Add(connectBtn)
		clientContainer.Add(input)
		clientContainer.Add(getBtn)
		clientContainer.Add(putBtn)
		clientContainer.Add(commandsScroll)
		clientContainer.Add(response)
		w3.SetContent(clientContainer)
		w3.Show()
		w3.SetCloseIntercept(func() {
			if client != nil {
				manager.CloseClient(client)
			}
			w3.Close()
		})
	})

	partitionBtn := widget.NewButton("Make Partition", func() {
		if btnArray[2].Text == "Make Partition" {
			// Server API: make partition
			manager.MakePartition()
			btnArray[2].SetText("Reconnect All")
		} else {
			// Server API: reconnect All
			manager.ReconnectAll()
			btnArray[2].SetText("Make Partition")
		}
	})

	btnArray[2] = partitionBtn
	exitBtn := widget.NewButton("Exit", func() {
		a.Quit()
	})

	// w.close by click x
	w.SetCloseIntercept(func() {
		a.Quit()
	})

	// w2.close by click x
	w2.SetCloseIntercept(func() {
		w2.Hide()
	})

	controlContainer := container.New(layout.NewGridWrapLayout(fyne.NewSize(240, 80)), layout.NewSpacer(), image, text2, text3, newClusterBtn, newClientBtn, partitionBtn, exitBtn)
	content := container.New(layout.NewHBoxLayout(), serverContainer, controlContainer)

	//w2
	label := canvas.NewText("How many server to create?", color.White)
	selectNum := widget.NewSelect([]string{"2", "3", "4", "5"}, nil)
	reliable := widget.NewCheck("Reliable Network", nil)
	confirmBtn := widget.NewButton("Confirm", func() {
		for idx, item := range serverContainer.Objects {
			if idx > 1 {
				serverContainer.Remove(item)
			}
		}
		num, _ := strconv.Atoi(selectNum.Selected)
		// create raft server here and store it into corresponding index

		// Server API: startServers
		manager.StartSevers(num, !reliable.Checked)

		serverArray = make([]string, num+1)
		btn1Array := make([]*widget.Button, num+1)
		btn2Array := make([]*widget.Button, num+1)

		//unchanged
		labels := []string{"State", "currentTerm", "votedFor", "commitIndex", "lastApplied"}
		serverInfos := make([]binding.ExternalStringList, num+1)
		serverLogEntries := make([]binding.ExternalStringList, num+1)
		serverDatabases := make([]binding.ExternalStringList, num+1)

		// test part
		//cfg := application.Make_config(num, !reliable.Checked, -1)
		//fmt.Println(cfg)
		//go func(cfg *application.Config) {
		//	ck := cfg.MakeClient(cfg.All())
		//	for i := 1; i < 100; i++ {
		//		time.Sleep(10 * time.Second)
		//		servers := cfg.ShowServerInfo()
		//		fmt.Println(servers[0].ShowDB())
		//		ck.Put(fmt.Sprint(i), "put operation")
		//	}
		//}(cfg)

		for i := 1; i <= num; i++ {
			index := i
			// bind each widget to its raft server
			serverArray[index] = "raft server" + strconv.Itoa(index)

			// Server API: since index start from 1, so for kvservers: index-1
			serverInfos[index] = manager.Cfg.Kvservers[index-1].Rf.ServerInfo // init server info binding
			serverLogEntries[index] = manager.Cfg.Kvservers[index-1].Rf.ServerLog
			serverDatabases[index] = manager.Cfg.Kvservers[index-1].ServerStore

			text1 := canvas.NewText("Raft Server No."+strconv.Itoa(index), color.White)
			text1.TextSize = 20
			text1.Alignment = fyne.TextAlignCenter
			labelList := widget.NewList(
				func() int {
					return len(labels)
				},
				func() fyne.CanvasObject {
					return widget.NewLabel("template")
				},
				func(i widget.ListItemID, o fyne.CanvasObject) {
					o.(*widget.Label).SetText(labels[i])
				})

			valueList := widget.NewListWithData(serverInfos[index],
				func() fyne.CanvasObject {
					return widget.NewLabel("follower")
				},
				func(i binding.DataItem, o fyne.CanvasObject) {
					o.(*widget.Label).Bind(i.(binding.String))
				})
			labelList.Resize(fyne.NewSize(120, 200))
			valueList.Resize(fyne.NewSize(60, 200))
			text2 := canvas.NewText("LogEntries", color.White)
			text2.TextSize = 10
			text2.Alignment = fyne.TextAlignCenter
			rect1 := canvas.NewRectangle(color.White)
			rect1.Resize(fyne.NewSize(200, 80))
			rect1.StrokeColor = color.White
			rect1.StrokeWidth = 1
			rect1.FillColor = color.Transparent

			logEntries := widget.NewListWithData(serverLogEntries[index],
				func() fyne.CanvasObject {
					return widget.NewLabel(templateStr)
				},
				func(i binding.DataItem, o fyne.CanvasObject) {
					o.(*widget.Label).Bind(i.(binding.String))
				})
			logEntries.Resize(fyne.NewSize(200, 80))
			text3 := canvas.NewText("KV Database", color.White)
			text3.TextSize = 10
			text3.Alignment = fyne.TextAlignCenter
			rect2 := canvas.NewRectangle(color.White)
			rect2.Resize(fyne.NewSize(200, 80))
			rect2.StrokeColor = color.White
			rect2.StrokeWidth = 1
			rect2.FillColor = color.Transparent

			applies := widget.NewListWithData(serverDatabases[index],
				func() fyne.CanvasObject {
					return widget.NewLabel(templateStr)
				},
				func(i binding.DataItem, o fyne.CanvasObject) {
					o.(*widget.Label).Bind(i.(binding.String))
				})
			applies.Resize(fyne.NewSize(200, 80))
			applies.ScrollToBottom()

			btn1 := widget.NewButton("Disconnect", func() {
				fmt.Println("should disconnect " + serverArray[index])
				if btn1Array[index].Text == "Disconnect" {
					// Server API: disconnect current server
					manager.Disconnect(index - 1)
					btn1Array[index].SetText("Reconnect")
				} else {
					// Server API: reconnect current server
					manager.Reconnect(index - 1)
					btn1Array[index].SetText("Disconnect")
				}
			})
			btn1Array[index] = btn1
			btn1.Resize(fyne.NewSize(120, 40))
			btn2 := widget.NewButton("Shutdown", func() {
				fmt.Println("should Shutdown " + serverArray[index])
				if btn2Array[index].Text == "Shutdown" {
					// Server API: shutdown current server
					manager.ShutDown(index - 1)
					btn2Array[index].SetText("Restart")
					fmt.Printf("Restart: -----------------%v,  index: %d\n", &serverInfos[index], index)
				} else if btn2Array[index].Text == "Restart" {
					// Server API: restart current server
					manager.Restart(index - 1)
					time.Sleep(2 * time.Second)
					ss, _ := manager.Cfg.Kvservers[index-1].Rf.ServerInfo.Get()
					fmt.Println("Restart--before serverInfos[index]=: -----------------,  info: ", ss)
					serverInfos[index] = manager.Cfg.Kvservers[index-1].Rf.ServerInfo
					ss1, _ := serverInfos[index].Get()
					fmt.Println("Restart--after serverInfos[index]=: -----------------,  info: ", ss1)
					serverContainer.Remove(valueList)
					newValueList := widget.NewListWithData(serverInfos[index],
						func() fyne.CanvasObject {
							return widget.NewLabel("template")
						},
						func(i binding.DataItem, o fyne.CanvasObject) {
							o.(*widget.Label).Bind(i.(binding.String))
						})
					newValueList.Resize(fyne.NewSize(60, 200))
					newValueList.Move(fyne.NewPos(float32(180+(index-1)*230), 65))
					valueList = newValueList
					serverContainer.Add(valueList)
					btn2Array[index].SetText("Reconnect")
					//TODO valueList
				} else {
					// Server API: reconnect current server
					manager.Reconnect(index - 1)
					btn2Array[index].SetText("Shutdown")
				}
			})
			btn2Array[index] = btn2
			btn2.Resize(fyne.NewSize(120, 40))
			text1.Move(fyne.NewPos(float32(145+(index-1)*230), 25))
			labelList.Move(fyne.NewPos(float32(45+(index-1)*230), 65))
			valueList.Move(fyne.NewPos(float32(180+(index-1)*230), 65))
			text2.Move(fyne.NewPos(float32(60+(index-1)*230), 260))
			rect1.Move(fyne.NewPos(float32(45+(index-1)*230), 280))
			logEntries.Move(fyne.NewPos(float32(45+(index-1)*230), 280))
			text3.Move(fyne.NewPos(float32(50+(index-1)*230), 360))
			rect2.Move(fyne.NewPos(float32(45+(index-1)*230), 380))
			applies.Move(fyne.NewPos(float32(45+(index-1)*230), 380))
			btn1.Move(fyne.NewPos(float32(80+(index-1)*230), 680))
			btn2.Move(fyne.NewPos(float32(80+(index-1)*230), 720))
			serverContainer.Add(text1)
			serverContainer.Add(labelList)
			serverContainer.Add(valueList)
			serverContainer.Add(text2)
			serverContainer.Add(logEntries)
			serverContainer.Add(rect1)
			serverContainer.Add(text3)
			serverContainer.Add(applies)
			serverContainer.Add(rect2)
			serverContainer.Add(btn1)
			serverContainer.Add(btn2)
			serverContainer.Refresh()
		}
		rect3 := canvas.NewRectangle(color.White)
		rect3.Resize(fyne.NewSize(1100, 180))
		rect3.Move(fyne.NewPos(50, 480))
		rect3.StrokeColor = color.White
		rect3.StrokeWidth = 1
		rect3.FillColor = color.Transparent

		console := widget.NewListWithData(manager.Cfg.ConsoleBinding,
			func() fyne.CanvasObject {
				return widget.NewLabel(templateStr)
			},
			func(i binding.DataItem, o fyne.CanvasObject) {
				o.(*widget.Label).Bind(i.(binding.String))
			})
		console.Resize(fyne.NewSize(1100, 180))
		console.Move(fyne.NewPos(50, 480))
		console.ScrollToBottom()
		serverContainer.Add(rect3)
		serverContainer.Add(console)
		w2.Hide()
	})
	clientParamsContainer := container.New(layout.NewGridWrapLayout(fyne.NewSize(220, 60)), label, selectNum, reliable, confirmBtn)

	w2.SetContent(clientParamsContainer)
	w.SetContent(content)

	w.ShowAndRun()
}
