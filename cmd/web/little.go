package main

import (
	"fmt"
	"image/color"
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
	red    = "red"
	notRed = "white"
)

func (app *application) little() {
	a := ap.New()
	w := a.NewWindow("Temperature and Power")

	row1Col1 := widget.NewLabel("Amp")
	row1Col2 := widget.NewLabel("Heatsink")
	row1Col3 := widget.NewLabel("Air")
	row1 := container.New(layout.NewGridLayout(3), row1Col1, row1Col2, row1Col3)

	td, err := app.updateSensors()
	if err != nil {
		trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
		app.errorLog.Output(2, trace)
		// app.serverError(w, err)
		return
	}

	row2Col1 := widget.NewLabel(td.AmpPower)
	row2Col2 := widget.NewLabel(td.SinkTemp)
	row2Col3 := widget.NewLabel(td.AirTemp)
	row2 := container.New(layout.NewGridLayout(3), row2Col1, row2Col2, row2Col3)

	grid := container.New(layout.NewVBoxLayout(), row1, row2)
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
			row2Col2.SetText(td.SinkTemp)
			row2Col3.SetText(td.AirTemp)

			t, _ := strconv.ParseFloat(td.SinkTemp, 64)
			if t > app.tempThreshold && colorCode != red {
				colorCode = red
				bgColor := canvas.NewRectangle(color.NRGBA{R: 255, G: 0, B: 0, A: 150})
				newRow1 := container.New(layout.NewMaxLayout(), bgColor, row1)
				newRow2 := container.New(layout.NewMaxLayout(), bgColor, row2)
				grid := container.New(layout.NewVBoxLayout(), newRow1, newRow2)
				w.SetContent(grid)
				w.Show()
			}
			if t < app.tempThreshold && colorCode == red {
				colorCode = notRed
				grid := container.New(layout.NewVBoxLayout(), row1, row2)
				w.SetContent(grid)
				w.Show()
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
