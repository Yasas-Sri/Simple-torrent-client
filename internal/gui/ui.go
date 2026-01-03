package gui

import (
	"fmt"
	"io"
	"time"

	"torrent-client/internal/p2p"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func StartUI(m *p2p.Manager) {
	a := app.New()
	w := a.NewWindow("Beta Torrent")
	w.Resize(fyne.NewSize(600, 400))

	boundList := binding.NewUntypedList()

	list := widget.NewListWithData(
		boundList,
		func() fyne.CanvasObject {

			nameLabel := widget.NewLabelWithStyle("Torrent Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			progBar := widget.NewProgressBar()
			detailsLabel := widget.NewLabelWithStyle("0 MB / 0 MB", fyne.TextAlignTrailing, fyne.TextStyle{Italic: true})

			return container.NewPadded(
				container.NewVBox(
					nameLabel,
					progBar,
					detailsLabel,
				),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			val, _ := item.(binding.Untyped).Get()
			if val == nil {
				return
			}
			stats := val.(p2p.TorrentStats)

			padded := obj.(*fyne.Container)
			vbox := padded.Objects[0].(*fyne.Container)

			nameLbl := vbox.Objects[0].(*widget.Label)
			pBar := vbox.Objects[1].(*widget.ProgressBar)
			detailLbl := vbox.Objects[2].(*widget.Label)

			downloadedMB := float64(stats.Downloaded) / 1024 / 1024
			totalMB := float64(stats.TotalLength) / 1024 / 1024

			nameLbl.SetText(stats.Name)
			pBar.SetValue(stats.Percent / 100)
			detailLbl.SetText(fmt.Sprintf("%.2f MB / %.2f MB (%.2f%%)", downloadedMB, totalMB, stats.Percent))
		},
	)

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err != nil || reader == nil {
					return
				}
				defer reader.Close()
				data, _ := io.ReadAll(reader)
				m.AddTorrent(data) // Add to engine
			}, w)
			fd.SetFilter(storage.NewExtensionFileFilter([]string{".torrent"}))
			fd.Show()
		}),
	)

	go func() {
		for {
			time.Sleep(time.Second)
			newData := m.GetStats()

			var interfaceList []interface{}
			for _, s := range newData {
				interfaceList = append(interfaceList, s)
			}
			boundList.Set(interfaceList)
		}
	}()

	title := widget.NewLabelWithStyle("Active Downloads", fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Italic: true})

	header := container.NewVBox(
		container.NewHBox(title, layout.NewSpacer(), toolbar),
		widget.NewSeparator(),
	)

	w.SetContent(container.NewBorder(header, nil, nil, nil, list))

	content := container.NewBorder(toolbar, nil, nil, nil, list)
	w.SetContent(content)

	w.ShowAndRun()
}
