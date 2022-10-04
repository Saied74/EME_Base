package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type templateData struct {
	YesData       string `json:"yesData"`
	NoConnection  string `json:"noConnection"`
	Msg           string `json:"msg"`
	AmpStatus     string `json:"ampStatus"`
	AmpPower      string `json:"ampPower"`
	AirTemp       string `json:"airTemp"`
	SinkTemp      string `json:"sinkTemp"`
	DoorStatus    string `json:"doorStatus"`
	tempThreshold string `json:"tempThreshold"`
	Peep          bool
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	td := &templateData{}
	app.render(w, r, "home.page.html", td)
}

func (app *application) monitor(w http.ResponseWriter, r *http.Request) {
	td, err := app.updateSensors()
	if err != nil {
		app.serverError(w, err)
		return
	}
	td.Peep = true
	app.render(w, r, "monitor.page.html", td)
}

func (app *application) updateMonitor(w http.ResponseWriter, r *http.Request) {

	td, err := app.updateSensors()
	if err != nil {
		app.serverError(w, err)
		return
	}
	td.tempThreshold = fmt.Sprintf("0.3%f", app.tempThreshold)
	b, err := json.Marshal(td)
	if err != nil {
		app.serverError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)

}

func (app *application) ampOn(w http.ResponseWriter, r *http.Request) {
	_, err := app.getRemote("t") //t for turn on
	if err != nil {
		app.serverError(w, err)
		return
	}
	td, err := app.updateSensors()
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.render(w, r, "monitor.page.html", td)
}

func (app *application) ampOff(w http.ResponseWriter, r *http.Request) {
	_, err := app.getRemote("f") //f for turn off
	if err != nil {
		app.serverError(w, err)
		return
	}
	td, err := app.updateSensors()
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.render(w, r, "monitor.page.html", td)

}

func (app *application) adjustments(w http.ResponseWriter, r *http.Request) {
	td := &templateData{}
	app.render(w, r, "adjustments.page.html", td)
}

func (app *application) reAdjust(w http.ResponseWriter, r *http.Request) {
	app.adjust()
	td, err := app.updateSensors()
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, r, "monitor.page.html", td)
}
