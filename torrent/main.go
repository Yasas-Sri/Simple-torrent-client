package main

import (
	"fmt"
	"time"

	"log"
	"torrent-client/internal/api"
	"torrent-client/internal/gui"
	"torrent-client/internal/p2p"
	"torrent-client/internal/tracker"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {

	myID, err := tracker.GeneratePeerID()
	if err != nil {
		log.Fatalf("Critical Error: Could not generate Peer ID: %v", err)
	}

	manager := p2p.NewManager(myID)

	server := api.NewServer(manager)
	go server.Start()
	gui.StartUI(manager)

}

func runGUI(m *p2p.Manager) {
	myApp := app.New()
	myWindow := myApp.NewWindow("Gemini Torrent Client")
	myWindow.Resize(fyne.NewSize(600, 400))

	// UI Components
	statusList := container.NewVBox()
	scroll := container.NewScroll(statusList)

	addInput := widget.NewEntry()
	addInput.SetPlaceHolder("Enter path to .torrent file...")

	addBtn := widget.NewButton("Add Torrent", func() {

		fmt.Println("Attempting to add:", addInput.Text)

	})

	go func() {
		for {
			time.Sleep(time.Second)
			stats := m.GetStats()

			statusList.Objects = nil
			for _, s := range stats {
				progress := widget.NewProgressBar()
				progress.SetValue(s.Percent / 100)

				info := widget.NewLabel(fmt.Sprintf("%s - %.2f%%", s.Name, s.Percent))
				statusList.Add(info)
				statusList.Add(progress)
			}
			statusList.Refresh()
		}
	}()

	myWindow.SetContent(container.NewBorder(
		container.NewVBox(addInput, addBtn),
		nil, nil, nil,
		scroll,
	))

	myWindow.ShowAndRun()
}
