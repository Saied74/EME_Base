package main

import (
	"encoding/json"
	"net/http"
)

type templateData struct {
	YesData      string `json:"yesData"`
	NoConnection string `json:"noConnection"`
	Msg          string `json:"msg"`
	AmpStatus    string `json:"ampStatus"`
	AmpPower     string `json:"ampPower"`
	AirTemp      string `json:"airTemp"`
	SinkTemp     string `json:"sinkTemp"`
	DoorStatus   string `json:"doorStatus"`
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "home.page.html", nil)
}

func (app *application) monitor(w http.ResponseWriter, r *http.Request) {
	td, err := app.getRemote("r") //r for report (status)
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.render(w, r, "monitor.page.html", td)
}

func (app *application) updateMonitor(w http.ResponseWriter, r *http.Request) {
	td, err := app.getRemote("r") //r for report (status
	if err != nil {
		app.serverError(w, err)
		return
	}
	if err == nil && td.Msg == "" {
		td, err = app.processSensors(td)
		if err != nil {
			app.serverError(w, err)
		}
	}
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
	td, err := app.getRemote("r") // r for report
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
	td, err := app.getRemote("r") // r for report
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.render(w, r, "monitor.page.html", td)

}

func (app *application) adjustments(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "adjustments.page.html", nil)
}
