/*
Team: GoPaddle

Team Members:
Zixin Wang   1047486  zixin3@student.unimelb.edu.au
Wenjun Wang  1249890  www4@student.unimelb.edu.au
Xinhao Chen  1230696  xinhchen1@student.unimelb.edu.au
Bocan Yang   1152078  bocany@student.unimelb.edu.au

RPC Module and Raft Framework:
MIT 6.824 Lab2:	“6.824 Lab 2: Raft,” Mit.edu. [Online]. Available: https://pdos.csail.mit.edu/6.824/labs/lab-raft.html.
MIT 6.824 Lab3:	“6.824 lab 3: Fault-tolerant key/value service,” Mit.edu. [Online]. Available: https://pdos.csail.mit.edu/6.824/labs/lab-kvraft.html.

Algorithm implementation，variable names，and any optimization ideas is following:
Raft Paper:	D. Ongaro and J. Ousterhout, “In search of an understandable consensus algorithm (extended version),” Mit.edu. [Online]. Available: https://pdos.csail.mit.edu/6.824/papers/raft-extended.pdf.
*/

package main

import (
	"GoPaddle-Raft/application"
	"fmt"
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

func mian() {
	a := app.New()
	w := a.NewWindow("GoPaddle's Application")
	w.Resize(fyne.NewSize(800, 600))
	w.SetFixedSize(true)

	//data := binding.BindStringList(
	//	&[]string{"Item 1", "Item 2", "Item 3"},
	//)
	//
	//list := widget.NewListWithData(data,
	//	func() fyne.CanvasObject {
	//		return widget.NewLabel("template")
	//	},
	//	func(i binding.DataItem, o fyne.CanvasObject) {
	//		o.(*widget.Label).Bind(i.(binding.String))
	//	})
	//
	//add := widget.NewButton("Append", func() {
	//	val := fmt.Sprintf("Item %d", data.Length()+1)
	//	data.Append(val)
	//})
	//
	//w.SetContent(container.NewBorder(nil, add, nil, nil, list))

	manager := &application.Manager{}
	manager.StartSevers(3, true)

	labels := []string{"State", "currentTerm", "votedFor", "commitIndex", "lastApplied"}
	//values := make([]binding.ExternalStringList, 1)

	//str := strServerInfo(manager.Cfg.Kvservers[1].Rf)

	go func() {
		for {
			//manager.ShowSingleServer(manager.Cfg.Kvservers[0].Rf)
			fmt.Printf("%v\n", manager.Cfg.Kvservers[0].Rf.ServerLog)
			//str = strServerInfo(manager.Cfg.Kvservers[1].Rf)
			//err := values[0].Reload()
			//if err != nil {
			//	return
			//}
			//select {
			//case <-manager.Cfg.Kvservers[0].Rf.InfoCh:
			//	manager.ShowSingleServer(manager.Cfg.Kvservers[0].Rf)
			//	//str = strServerInfo(manager.Cfg.Kvservers[1].Rf)
			//	err := values[0].Reload()
			//	if err != nil {
			//		return
			//	}
			//}
		}
	}()

	//values[0] = binding.BindStringList(
	//	&manager.Cfg.Kvservers[0].Rf.ServerInfo,
	//)

	text1 := canvas.NewText("Raft Server No."+strconv.Itoa(0), color.White)
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

	valueList := widget.NewListWithData(manager.Cfg.Kvservers[0].Rf.ServerInfo,
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(i.(binding.String))
		})
	labelList.Resize(fyne.NewSize(200, 200))
	valueList.Resize(fyne.NewSize(60, 200))
	//serverContainer := container.New(layout.NewVBoxLayout(), text1, labelList, valueList)
	w.SetContent(container.NewBorder(text1, nil, labelList, valueList))

	w.ShowAndRun()
}
