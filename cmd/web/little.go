package main

import (
	"fmt"
	"image/color"
	"log"
	"runtime/debug"
	//	"strconv"
	"time"

	"fyne.io/fyne/v2"
	ap "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const (
	notRed            = "white"
	turnOn            = "Turn Remote On"
	turnOff           = "Turn Remote Off"
	airTempThreshold  = 30.0
	sinkTempThreshold  = 50.0
	refPowerThreshold = 50.0
	swrMidThreshold   = 1.7
	swrHighThreshold  = 2.5
)

var ampStatus = binding.NewString()
var ampPower = binding.NewString()
var refPower = binding.NewString()
var swr = binding.NewString()
var airTemp = binding.NewString()
var sinkTemp = binding.NewString()
var fan1 = binding.NewString()
var fan2 = binding.NewString()
var doorStatus = binding.NewString()
var pttStatus = binding.NewString()
var whiteExternalColor color.Color = color.White
var airExternalColor color.Color = color.White
var sinkExternalColor color.Color = color.White
var whiteBoundExternalColor = binding.BindUntyped(&whiteExternalColor)
var airBoundExternalColor = binding.BindUntyped(&airExternalColor)
var sinkBoundExternalColor = binding.BindUntyped(&sinkExternalColor)
	


var (
	red    = color.NRGBA{R: 255, G: 0, B: 0, A: 125}
	green  = color.NRGBA{R: 0, G: 255, B: 0, A: 150}
	yellow = color.NRGBA{R: 255, G: 255, B: 0, A: 150}
	baige  = color.NRGBA{R: 207, G: 185, B: 151, A: 150}
	clr = color.White
)


type BoundColorRectWidget struct {
	widget.BaseWidget
	rect        *canvas.Rectangle
	boundColor  binding.Untyped      // Now binds the untyped binding directly
	dataListener binding.DataListener
}

func NewBoundColorRectWidget(boundColor binding.Untyped) *BoundColorRectWidget {
	r := &BoundColorRectWidget{
		rect:       canvas.NewRectangle(color.Black), // Initial color
		boundColor: boundColor,
	}
	r.ExtendBaseWidget(r)

	// Set up the listener internally to avoid manual calls outside
	r.dataListener = binding.NewDataListener(func() {
		val, err := r.boundColor.Get()
		if err == nil {
			if col, ok := val.(color.Color); ok { // <--- CRUCIAL: Type assertion here
				r.rect.FillColor = col
				r.rect.Refresh()
			}
		}
	})
	r.boundColor.AddListener(r.dataListener)

	// Set initial color from the binding
	if val, err := r.boundColor.Get(); err == nil {
		if col, ok := val.(color.Color); ok {
			r.rect.FillColor = col
		}
	}

	return r
}

func (r *BoundColorRectWidget) CreateRenderer() fyne.WidgetRenderer {
	return &boundColorRectWidgetRenderer{
		widget:  r,
		objects: []fyne.CanvasObject{r.rect},
	}
}

type boundColorRectWidgetRenderer struct {
	widget  *BoundColorRectWidget
	objects []fyne.CanvasObject
}

func (r *boundColorRectWidgetRenderer) MinSize() fyne.Size {
	return fyne.NewSize(5,5)
}

func (r *boundColorRectWidgetRenderer) Layout(size fyne.Size) {
	r.widget.rect.Resize(size)
}

func (r *boundColorRectWidgetRenderer) Refresh() {
	r.widget.rect.Refresh()
}

func (r *boundColorRectWidgetRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *boundColorRectWidgetRenderer) Destroy() {
	r.widget.boundColor.RemoveListener(r.widget.dataListener)
}

func (app *application) little() {
	airBoundExternalColor.Reload()
	sinkBoundExternalColor.Reload()

	go func() {
		time.Sleep(1 * time.Second)
		for {
			err := app.updateBoundSensors()
			if err != nil {
				trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
				app.errorLog.Output(2, trace)
			}
			time.Sleep(1 * time.Second)
		}
	}()

	onText := turnOn
	calText := "Recalibrate"
	a := ap.New()
	w := a.NewWindow("Remote Statistics")
	dvra, err := fyne.LoadResourceFromPath("ui/static/img/dvra.jpeg")
	if err != nil {
		log.Fatal(err)
	}
	w.SetIcon(dvra)

	row1Col1 := makeItem("Fwd", baige, true)
	row1Col2 := makeItem("Ref", baige, true)
	row1Col3 := makeItem("SWR", baige, true)

	row3Col1 := makeItem("Air", baige, true)
	row3Col2 := makeItem("Amp", baige, true)
	row3Col3 := makeItem("Door", baige, true)

	row5Col1 := makeItem("Fan1", baige, true)
	row5Col2 := makeItem("Fan2", baige, true)
	row5Col3 := makeItem("PTT", baige, true)

	row1 := container.New(layout.NewGridLayout(3), row1Col1, row1Col2, row1Col3)
	row3 := container.New(layout.NewGridLayout(3), row3Col1, row3Col2, row3Col3)
	row5 := container.New(layout.NewGridLayout(3), row5Col1, row5Col2, row5Col3)
	
	row2Col1 := makeBoundItem(ampPower, whiteBoundExternalColor)
	row2Col2 := makeBoundItem(refPower, whiteBoundExternalColor)
	row2Col3 := makeBoundItem(swr, whiteBoundExternalColor)
	row4Col1 := makeBoundItem(airTemp, airBoundExternalColor)
	row4Col2 := makeBoundItem(sinkTemp, sinkBoundExternalColor)
	row4Col3 := makeBoundItem(doorStatus, whiteBoundExternalColor)

	row6Col1 := makeBoundItem(fan1, whiteBoundExternalColor)
	row6Col2 := makeBoundItem(fan2, whiteBoundExternalColor)
	row6Col3 := makeBoundItem(pttStatus, whiteBoundExternalColor)

	button := widget.NewButton(onText, func() {
		if app.remoteOn {
			app.getRemote("f")
			app.remoteOn = false
		} else {
			app.getRemote("t")
			app.remoteOn = true
		}
	})

	calButton := widget.NewButton(calText, func() {
		app.adjust()
	})

	row2 := container.New(layout.NewGridLayout(3), row2Col1, row2Col2, row2Col3)
	row4 := container.New(layout.NewGridLayout(3), row4Col1, row4Col2, row4Col3)
	row6 := container.New(layout.NewGridLayout(3), row6Col1, row6Col2, row6Col3)

	grid := container.New(layout.NewVBoxLayout(), row1, row2, row3, row4, row5, row6, button, calButton)
	w.SetContent(grid)
	w.Resize(fyne.NewSize(180, 80))

	w.Show()
	a.Run()
	fmt.Println("Exited")
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

func makeBoundItem(b binding.String, bndExtClr binding.ExternalUntyped) *fyne.Container {
	//return container.New(layout.NewCenterLayout(), widget.NewLabelWithData(b))
	
	bgColor := NewBoundColorRectWidget(bndExtClr) //canvas.NewRectangle(clr)
	//bndExtClr.Reload()

	can := widget.NewLabelWithData(b)//canvas.NewText(bStr, color.Black)
	
	con := container.New(layout.NewCenterLayout(), can)
	return container.New(layout.NewMaxLayout(), bgColor, con)
}
