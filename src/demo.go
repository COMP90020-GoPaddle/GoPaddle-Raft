package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"strconv"
	"time"
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
	w := a.NewWindow("GoPaddle's Application")
	w.Resize(fyne.NewSize(1440, 800))
	w.SetFixedSize(true)
	w2 := a.NewWindow("New Cluster")
	w2.Resize(fyne.NewSize(220, 240))
	w2.SetFixedSize(true)

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
		connectBtn := widget.NewButton("Connect", func() {
			clientArray[serverIndex].SetValue(0, "new id")
		})
		connectBtn.Resize(fyne.NewSize(120, 40))
		commands := widget.NewTextGridFromString(clientConsoleArray[serverIndex])
		commandsScroll := container.NewScroll(commands)
		commandsScroll.Resize(fyne.NewSize(300, 150))
		commandsScroll.ScrollToBottom()

		input := widget.NewEntry()
		input.SetPlaceHolder("Enter text...")
		input.Resize(fyne.NewSize(300, 40))
		getBtn := widget.NewButton("Get", func() {
			fmt.Println("Get " + input.Text)
			clientConsoleArray[serverIndex] += "Get " + input.Text + "\n"
			fmt.Println(clientConsoleArray[serverIndex])
			input.SetPlaceHolder("Enter text...")
			input.SetText("")
			commands.SetText(clientConsoleArray[serverIndex])
			commandsScroll.ScrollToBottom()
		})
		getBtn.Resize(fyne.NewSize(120, 40))
		putBtn := widget.NewButton("Put", func() {
			clientConsoleArray[serverIndex] += "Put " + input.Text + "\n"
			input.SetPlaceHolder("Enter text...")
			input.SetText("")
			commands.SetText(clientConsoleArray[serverIndex])
			commandsScroll.ScrollToBottom()
		})
		responseText := binding.NewString()
		response := widget.NewEntryWithData(responseText)
		responseScroll := container.NewScroll(response)
		responseScroll.Resize(fyne.NewSize(300, 150))
		putBtn.Resize(fyne.NewSize(120, 40))
		clientLabel.Move(fyne.NewPos(35, 35))
		clientId.Move(fyne.NewPos(125, 35))
		connectBtn.Move(fyne.NewPos(140, 80))
		input.Move(fyne.NewPos(50, 140))
		getBtn.Move(fyne.NewPos(60, 200))
		putBtn.Move(fyne.NewPos(220, 200))
		commandsScroll.Move(fyne.NewPos(50, 250))
		responseScroll.Move(fyne.NewPos(50, 410))
		clientContainer.Add(clientLabel)
		clientContainer.Add(clientId)
		clientContainer.Add(connectBtn)
		clientContainer.Add(input)
		clientContainer.Add(getBtn)
		clientContainer.Add(putBtn)
		clientContainer.Add(commandsScroll)
		clientContainer.Add(responseScroll)
		w3.SetContent(clientContainer)
		w3.Show()
		go func() {
			for {
				time.Sleep(5000 * time.Millisecond)
				str, _ := responseText.Get()
				responseText.Set(str + "new line\n")
			}
		}()
	})
	partitionBtn := widget.NewButton("Make Partition", func() {
		if btnArray[2].Text == "Make Partition" {
			btnArray[2].SetText("Reconnect All")
		} else {
			btnArray[2].SetText("Make Partition")
		}
	})
	btnArray[2] = partitionBtn
	exitBtn := widget.NewButton("Exit", func() {
		w.Close()
		w2.Close()
	})
	controlContainer := container.New(layout.NewGridWrapLayout(fyne.NewSize(240, 80)), layout.NewSpacer(), image, text2, text3, newClusterBtn, newClientBtn, partitionBtn, exitBtn)
	content := container.New(layout.NewHBoxLayout(), serverContainer, controlContainer)

	//w2
	label := canvas.NewText("How many server to create?", color.White)
	selectNum := widget.NewSelect([]string{"1", "2", "3", "4", "5"}, nil)
	reliable := widget.NewCheck("Reliable Network", nil)
	confirmBtn := widget.NewButton("Confirm", func() {
		for idx, item := range serverContainer.Objects {
			if idx > 1 {
				serverContainer.Remove(item)
			}
		}
		num, _ := strconv.Atoi(selectNum.Selected)
		// create raft server here and store it into corresponding index
		serverArray = make([]string, num+1)
		btn1Array := make([]*widget.Button, num+1)
		btn2Array := make([]*widget.Button, num+1)
		//unchanged
		labels := []string{"State", "currentTerm", "votedFor", "commitIndex", "lastApplied"}
		values := make([]binding.ExternalStringList, num+1)
		for i := 1; i <= num; i++ {
			index := i
			// bind each widget to its raft server
			serverArray[index] = "raft server" + strconv.Itoa(index)
			values[index] = binding.BindStringList(
				&[]string{"Item 1", "Item 2", "Item 3", "Item 4", "Item 5"},
			)
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

			valueList := widget.NewListWithData(values[index],
				func() fyne.CanvasObject {
					return widget.NewLabel("template")
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
			logEntries := widget.NewTextGrid()
			logEntries.SetText("Tiring......\nTiring......\nTiring......\nTiring......\nTiring......\nTiring......\nTiring......\n")
			logScroll := container.NewScroll(logEntries)
			logScroll.Resize(fyne.NewSize(200, 80))
			logScroll.ScrollToBottom()
			text3 := canvas.NewText("Applies", color.White)
			text3.TextSize = 10
			text3.Alignment = fyne.TextAlignCenter
			rect2 := canvas.NewRectangle(color.White)
			rect2.Resize(fyne.NewSize(200, 80))
			rect2.StrokeColor = color.White
			rect2.StrokeWidth = 1
			rect2.FillColor = color.Transparent
			applies := widget.NewTextGrid()
			applies.SetText("Sleeping......\nSleeping......\nSleeping......\nSleeping......\nSleeping......\nSleeping......\nSleeping......\nSleeping......\n")
			applyScroll := container.NewScroll(applies)
			applyScroll.Resize(fyne.NewSize(200, 80))
			applyScroll.ScrollToBottom()
			btn1 := widget.NewButton("Disconnect", func() {
				fmt.Println("should disconnect " + serverArray[index])
				if btn1Array[index].Text == "Disconnect" {
					btn1Array[index].SetText("Reconnect")
				} else {
					btn1Array[index].SetText("Disconnect")
				}
			})
			btn1Array[index] = btn1
			btn1.Resize(fyne.NewSize(120, 40))
			btn2 := widget.NewButton("Shutdown", func() {
				fmt.Println("should Shutdown " + serverArray[index])
				if btn2Array[index].Text == "Shutdown" {
					btn2Array[index].SetText("Restart")
				} else {
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
			logScroll.Move(fyne.NewPos(float32(45+(index-1)*230), 280))
			text3.Move(fyne.NewPos(float32(50+(index-1)*230), 360))
			rect2.Move(fyne.NewPos(float32(45+(index-1)*230), 380))
			applyScroll.Move(fyne.NewPos(float32(45+(index-1)*230), 380))
			btn1.Move(fyne.NewPos(float32(80+(index-1)*230), 680))
			btn2.Move(fyne.NewPos(float32(80+(index-1)*230), 720))
			serverContainer.Add(text1)
			serverContainer.Add(labelList)
			serverContainer.Add(valueList)
			serverContainer.Add(text2)
			serverContainer.Add(logScroll)
			serverContainer.Add(rect1)
			serverContainer.Add(text3)
			serverContainer.Add(applyScroll)
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
		console := widget.NewTextGrid()
		console.SetText(
			"Raft Server[1]: Uh sama lamaa duma lamaa you assuming I'm a human. What I gotta do to get it through to you I'm superhuman. Innovative and I'm made of rubber. So that anything you saying ricocheting off of me and it'll glue to you\n" +
				"Raft Server[2]: I'm never stating more than never demonstrating. How to give a motherfuckin' audience a feeling like it's levitating. Never fading and I know that the haters are forever waiting\n" +
				"Raft Server[3]: For the day that they can say I fell off they'd be celebrating. Cause I know the way to get 'em motivated. I make elevating music you make elevator music\n" +
				"Raft Server[4]: Well that's what they do when they get jealous they confuse it. It's not hip hop it's pop cause I found a hella way to fuse it. With rock shock rap with Doc. Throw on Lose Yourself and make 'em lose it\n")
		consoleScroll := container.NewScroll(console)
		consoleScroll.Resize(fyne.NewSize(1100, 180))
		consoleScroll.Move(fyne.NewPos(50, 480))
		consoleScroll.ScrollToBottom()
		serverContainer.Add(rect3)
		serverContainer.Add(consoleScroll)
		w2.Hide()
	})
	clientParamsContainer := container.New(layout.NewGridWrapLayout(fyne.NewSize(220, 60)), label, selectNum, reliable, confirmBtn)

	w2.SetContent(clientParamsContainer)
	w.SetContent(content)
	w.ShowAndRun()
}
