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
	//red     = "red"
	notRed            = "white"
	turnOn            = "Turn Remote On"
	turnOff           = "Turn Remote Off"
	airTempThreshold  = 30.0
	ampTempThreshold  = 50.0
	refPowerThreshold = 50.0
	swrMidThreshold   = 1.7
	swrHighThreshold  = 2.5
)

var (
	red    = color.NRGBA{R: 255, G: 0, B: 0, A: 125}
	green  = color.NRGBA{R: 0, G: 255, B: 0, A: 150}
	yellow = color.NRGBA{R: 255, G: 255, B: 0, A: 150}
	baige  = color.NRGBA{R: 207, G: 185, B: 151, A: 150}
)

func (app *application) little() {

	buttonText := turnOn
	a := ap.New()
	w := a.NewWindow("Remote Statistics")
	dvra, err := fyne.LoadResourceFromPath("ui/static/img/dvra.jpeg")
	if err != nil {
		log.Fatal(err)
	}
	w.SetIcon(dvra)
	
	row1Col1 := makeItem("Fwd", baige, true) //widget.NewLabel("Fwd")
	row1Col2 := makeItem("Ref", baige, true) //widget.NewLabel("Rev")
	row1Col3 := makeItem("SWR", baige, true)

	row3Col1 := makeItem("Air", baige, true)  //widget.NewLabel("Air")
	row3Col2 := makeItem("Amp", baige, true)  //widget.NewLabel("Heatsink")
	row3Col3 := makeItem("Door", baige, true) //widget.NewLabel("Fan 1")

	row5Col1 := makeItem("Fan1", baige, true) //widget.NewLabel("Fan 2")
	row5Col2 := makeItem("Fan2", baige, true) // widget.NewLabel("Door")
	row5Col3 := makeItem("PTT", baige, true)  //widget.NewLabel("PTT")

	row1 := container.New(layout.NewGridLayout(3), row1Col1, row1Col2, row1Col3)
	row3 := container.New(layout.NewGridLayout(3), row3Col1, row3Col2, row3Col3)
	row5 := container.New(layout.NewGridLayout(3), row5Col1, row5Col2, row5Col3)

	td, err := app.updateSensors()
	if err != nil {
		trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
		app.errorLog.Output(2, trace)
		// app.serverError(w, err)
		return
	}

	row2Col1 := makeItem(td.AmpPower, color.White, false) //widget.NewLabel(td.AmpPower)
	row2Col2 := makeItem(td.RefPower, color.White, false) //widget.NewLabel(td.RefPower)
	row2Col3 := makeItem(td.SWR, color.White, false)

	row4Col1 := makeItem(td.AirTemp, color.White, false)    //widget.NewLabel(td.AirTemp)
	row4Col2 := makeItem(td.SinkTemp, color.White, false)   //widget.NewLabel(td.SinkTemp)
	row4Col3 := makeItem(td.DoorStatus, color.White, false) //widget.NewLabel(td.Fan1)

	row6Col1 := makeItem(td.Fan1, color.White, false)      //widget.NewLabel(td.Fan2)
	row6Col2 := makeItem(td.Fan2, color.White, false)      //widget.NewLabel(td.DoorStatus)
	row6Col3 := makeItem(td.PttStatus, color.White, false) //widget.NewLabel(td.PttStatus)

	button := widget.NewButton(buttonText, func() {
		if app.remoteOn {
			app.getRemote("f")
			app.remoteOn = false
		} else {
			app.getRemote("t")
			app.remoteOn = true
		}
	})
	row2 := container.New(layout.NewGridLayout(3), row2Col1, row2Col2, row2Col3)
	row4 := container.New(layout.NewGridLayout(3), row4Col1, row4Col2, row4Col3)
	row6 := container.New(layout.NewGridLayout(3), row6Col1, row6Col2, row6Col3)

	grid := container.New(layout.NewVBoxLayout(), row1, row2, row3, row4, row5, row6, button)
	w.SetContent(grid)
	w.Resize(fyne.NewSize(180, 80))

	go func() {
		//		colorCode := notRed
		for range time.Tick(time.Second) {
			td, err := app.updateSensors()
			if err != nil {
				trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
				app.errorLog.Output(2, trace)
			}

			dc := td.setColors()
			row2Col1 = makeItem(td.AmpPower, dc.forColor, false) //widget.NewLabel(td.AmpPower)
			row2Col2 = makeItem(td.RefPower, dc.refColor, false) //widget.NewLabel(td.RefPower)
			row2Col3 = makeItem(td.SWR, dc.swrColor, false)

			row4Col1 = makeItem(td.AirTemp, dc.airColor, false)     //widget.NewLabel(td.AirTemp)
			row4Col2 = makeItem(td.SinkTemp, dc.ampColor, false)    //widget.NewLabel(td.SinkTemp)
			row4Col3 = makeItem(td.DoorStatus, dc.doorColor, false) //widget.NewLabel(td.Fan1)

			row6Col1 = makeItem(td.Fan1, dc.fan1Color, false)     //widget.NewLabel(td.Fan2)
			row6Col2 = makeItem(td.Fan2, dc.fan2Color, false)     //widget.NewLabel(td.DoorStatus)
			row6Col3 = makeItem(td.PttStatus, dc.pttColor, false) //widget.NewLabel(td.PttStatus)

			row2 = container.New(layout.NewGridLayout(3), row2Col1, row2Col2, row2Col3)
			row4 = container.New(layout.NewGridLayout(3), row4Col1, row4Col2, row4Col3)
			row6 = container.New(layout.NewGridLayout(3), row6Col1, row6Col2, row6Col3)

			grid = container.New(layout.NewVBoxLayout(), row1, row2, row3, row4, row5, row6, button)
			w.SetContent(grid)
			//w.Show()
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

func makeItem(txt string, clr color.Color, bld bool) *fyne.Container {
	bgColor := canvas.NewRectangle(clr)
	can := canvas.NewText(txt, color.Black)
	if bld {
		can.TextStyle.Bold = true
	}
	con := container.New(layout.NewCenterLayout(), can)
	return container.New(layout.NewMaxLayout(), bgColor, con)
}

type displayColors struct {
	forColor  color.Color
	refColor  color.Color
	swrColor  color.Color
	airColor  color.Color
	ampColor  color.Color
	doorColor color.Color
	fan1Color color.Color
	fan2Color color.Color
	pttColor  color.Color
}

func (td *templateData) setColors() *displayColors {

	dc := &displayColors{}

	dc.forColor = color.White
	dc.refColor = color.White
	dc.swrColor = color.White
	dc.airColor = color.White
	dc.ampColor = color.White
	dc.doorColor = color.White
	dc.fan1Color = color.White
	dc.fan2Color = color.White
	dc.pttColor = color.White

	airT, _ := strconv.ParseFloat(td.AirTemp, 64)
	if airT > airTempThreshold {
		dc.airColor = red
	}
	ampT, _ := strconv.ParseFloat(td.SinkTemp, 64)
	if ampT > ampTempThreshold {
		dc.ampColor = red
	}
	swr, _ := strconv.ParseFloat(td.SWR, 64)
	if swr > swrMidThreshold {
		dc.swrColor = yellow
	}
	if swr > swrHighThreshold {
		dc.swrColor = red
	}
	return dc
}
