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
)

type serverBox struct {
}

func (sB *serverBox) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(1300, 800)
}

func (sB *serverBox) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	pos := fyne.NewPos(25, 25+containerSize.Height-sB.MinSize(objects).Height)
	objects[0].Move(pos)
}

func main() {
	a := app.New()
	w := a.NewWindow("GoPaddle's Application")
	w.Resize(fyne.NewSize(1600, 800))
	w.SetFixedSize(true)
	w2 := a.NewWindow("New Server")
	w2.Resize(fyne.NewSize(220, 240))
	w2.SetFixedSize(true)

	//w
	text1 := canvas.NewText("GoPaddle", color.White)
	text1.Alignment = fyne.TextAlignCenter
	text1.TextSize = 40
	text2 := canvas.NewText("Raft Server", color.White)
	text2.Alignment = fyne.TextAlignCenter
	text2.TextSize = 30
	text3 := canvas.NewText("Monitor Platform", color.White)
	text3.Alignment = fyne.TextAlignCenter
	text3.TextSize = 30
	newServerBtn := widget.NewButton("New Server", func() {
		w2.Show()
	})
	newClientBtn := widget.NewButton("New Client", func() {
	})
	exitBtn := widget.NewButton("Exit", func() {
		w.Close()
		w2.Close()
	})
	controlContainer := container.New(layout.NewGridWrapLayout(fyne.NewSize(300, 100)), layout.NewSpacer(), text1, text2, text3, newServerBtn, newClientBtn, exitBtn)
	rect := canvas.NewRectangle(color.White)
	rect.Resize(fyne.NewSize(1250, 750))
	rect.StrokeColor = color.White
	rect.StrokeWidth = 3
	rect.FillColor = color.Transparent
	serverContainer := container.New(&serverBox{}, rect)
	content := container.New(layout.NewHBoxLayout(), serverContainer, controlContainer)

	//w2
	label := canvas.NewText("How many server to create?", color.White)
	selectNum := widget.NewSelect([]string{"1", "2", "3", "4", "5"}, nil)
	reliable := widget.NewCheck("Reliable Network", nil)
	confirmBtn := widget.NewButton("Confirm", func() {
		num, _ := strconv.Atoi(selectNum.Selected)
		for i := 1; i <= num; i++ {
			index := i
			go func() {
				text1 := canvas.NewText("Raft Server No."+strconv.Itoa(index), color.White)
				text1.TextSize = 20
				text1.Alignment = fyne.TextAlignCenter
				text2 := canvas.NewText("Server Info", color.White)
				text2.TextSize = 20
				text2.Alignment = fyne.TextAlignCenter
				btn1 := widget.NewButton("Check Log", func() {
					fmt.Println("this is check log button " + strconv.Itoa(index))
				})
				btn1.Resize(fyne.NewSize(150, 50))
				btn2 := widget.NewButton("Disconnect", func() {
					fmt.Println("this is disconnect button " + strconv.Itoa(index))
				})
				btn2.Resize(fyne.NewSize(150, 50))

				text1.Move(fyne.NewPos(float32(150+(index-1)*250), 25))
				text2.Move(fyne.NewPos(float32(150+(index-1)*250), 55))
				btn1.Move(fyne.NewPos(float32(75+(index-1)*250), 650))
				btn2.Move(fyne.NewPos(float32(75+(index-1)*250), 700))
				serverContainer.Add(text1)
				serverContainer.Add(text2)
				serverContainer.Add(btn1)
				serverContainer.Add(btn2)
				serverContainer.Refresh()
			}()
		}
		w2.Hide()
	})
	clientParamsContainer := container.New(layout.NewGridWrapLayout(fyne.NewSize(220, 60)), label, selectNum, reliable, confirmBtn)

	w2.SetContent(clientParamsContainer)
	w.SetContent(content)

	w.ShowAndRun()
}
