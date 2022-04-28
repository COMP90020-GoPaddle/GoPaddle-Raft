package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"strconv"
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

func main() {
	a := app.New()
	win := a.NewWindow("GoPaddle")
	win.Resize(fyne.NewSize(500, 600))
	baseContainer := container.NewHBox(layout.NewSpacer())

	title := canvas.NewText("GoPaddle", color.Black)
	title.TextSize = 50

	count := 0
	serverBtn := widget.NewButton("New Server", func() {
		baseContainer.Add(makeServerUI(baseContainer, &count))
		fmt.Println(len(baseContainer.Objects))
	})

	clientBtn := widget.NewButton("New Client", func() {

	})

	exitBtn := widget.NewButton("EXIT", func() {

	})

	controlBox := container.NewVBox(layout.NewSpacer(), title, serverBtn, clientBtn, exitBtn, layout.NewSpacer())
	baseContainer.Add(controlBox)
	baseContainer.Add(layout.NewSpacer())
	fmt.Println(len(baseContainer.Objects))
	win.SetContent(baseContainer)

	win.Show()
	a.Run()
}
