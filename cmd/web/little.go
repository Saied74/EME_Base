package main

import (
	"fmt"
	"image/color"
	"log"
	"runtime/debug"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	ap "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const (
	red     = "red"
	notRed  = "white"
	turnOn  = "Turn Remote On"
	turnOff = "Turn Remote Off"
)

func (app *application) little() {
	buttonText := turnOn
	a := ap.New()
	w := a.NewWindow("Temperature and Power")
	dvra, err := fyne.LoadResourceFromPath("ui/static/img/dvra.jpeg")
	if err != nil {
		log.Fatal(err)
	}
	w.SetIcon(dvra)
	row1Col1 := widget.NewLabel("Amp")
	row1Col2 := widget.NewLabel("Air")
	row1Col3 := widget.NewLabel("Heatsink")
	row1 := container.New(layout.NewGridLayout(3), row1Col1, row1Col2, row1Col3)

	td, err := app.updateSensors()
	if err != nil {
		trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
		app.errorLog.Output(2, trace)
		// app.serverError(w, err)
		return
	}

	row2Col1 := widget.NewLabel(td.AmpPower)
	row2Col2 := widget.NewLabel(td.AirTemp)
	row2Col3 := widget.NewLabel(td.SinkTemp)
	button := widget.NewButton(buttonText, func() {
		if app.remoteOn {
			app.getRemote("f")
			//buttonText = turnOn
			app.remoteOn = false
		} else {
			app.getRemote("t")
			//	buttonText = turnOff
			app.remoteOn = true
		}
	})
	row2 := container.New(layout.NewGridLayout(3), row2Col1, row2Col2, row2Col3)

	grid := container.New(layout.NewVBoxLayout(), row1, row2, button)
	w.SetContent(grid)
	w.Resize(fyne.NewSize(180, 80))

	go func() {
		colorCode := notRed
		for range time.Tick(time.Second) {
			td, err := app.updateSensors()
			if err != nil {
				trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
				app.errorLog.Output(2, trace)
			}
			row2Col1.SetText(td.AmpPower)
			row2Col2.SetText(td.AirTemp)
			row2Col3.SetText(td.SinkTemp)

			t, _ := strconv.ParseFloat(td.SinkTemp, 64)
			if t > app.tempThreshold && colorCode != red {
				colorCode = red
				bgColor := canvas.NewRectangle(color.NRGBA{R: 255, G: 0, B: 0, A: 150})
				newRow1 := container.New(layout.NewMaxLayout(), bgColor, row1)
				newRow2 := container.New(layout.NewMaxLayout(), bgColor, row2)
				grid := container.New(layout.NewVBoxLayout(), newRow1, newRow2, button)
				w.SetContent(grid)
				w.Show()
			}
			if t < app.tempThreshold && colorCode == red {
				colorCode = notRed
				grid := container.New(layout.NewVBoxLayout(), row1, row2)
				w.SetContent(grid)
				w.Show()
			}
			if app.remoteOn {
				button.SetText(turnOff)
			} else {
				button.SetText(turnOn)
			}
		}
	}()
	w.Show()
	a.Run()
	fmt.Println("Exited")
}

func rowData(s []string, colorCode string) *fyne.Container {
	c := container.New(layout.NewGridLayout(3),
		widget.NewLabel(s[0]),
		widget.NewLabel(s[1]),
		widget.NewLabel(s[2]))

	if colorCode == "red" {
		bgColor := canvas.NewRectangle(color.NRGBA{R: 255, G: 0, B: 0, A: 150})
		return container.New(layout.NewMaxLayout(), bgColor, c)
	}
	return c
}

func getGrid(row1, row3 []string, colorCode string) *fyne.Container {
	return container.New(layout.NewVBoxLayout(),
		rowData(row1, colorCode),
		rowData(row3, colorCode))
}
