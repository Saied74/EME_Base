package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"

	"path/filepath"
	"runtime/debug"
)

const (
	remoteAddr        = "192.168.4.72" //TODO: need to change it to a server name
	absZero           = 273.15         //per Lord Kelvin
	calTemp           = 25.0           //per TI datasheet
	calVoltage        = 2.982          //per TI datasheet
	airFactor         = 1.002          //correction factor for air temperature sensor relative to thermocouple
	sinkFactor        = 1.007          //correction factor for the heatsink temperature sensor relative to thermocouple
	plusFive          = 4.94           //Arduino Nano measured reference voltage.
	maxAtoD           = 1023.0         //10 bits all ones.
	maxPower          = 250.0          //assumed
	maxPowerIndicator = 5.0            //assumed
	//ampThreshold := 66.0; //votage value for 66 degrees C temperature
	//TODO: need to build an alarm for high temperature
)

// <+++++++++++++++++++++++ Template Processing +++++++++++++++++++++++++++>

func newTemplateCache(dir string) (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := filepath.Glob(filepath.Join(dir, "*.page.html"))
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)
		ts, err := template.ParseFiles(page)
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseGlob(filepath.Join(dir, "*.layout.html"))
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseGlob(filepath.Join(dir, "*.partial.html"))
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}
	return cache, nil
}

func (app *application) render(w http.ResponseWriter, r *http.Request,
	name string, td *templateData) {
	ts, ok := app.templateCache[name]
	if !ok {
		app.serverError(w, fmt.Errorf("The template %s does not exist",
			name))
		return
	}
	buf := new(bytes.Buffer)
	err := ts.Execute(buf, td)
	if err != nil {
		app.serverError(w, err)
		return
	}
	buf.WriteTo(w)
}

//<++++++++++++++++   centralized error handling   +++++++++++++++++++>

func (app *application) serverError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.errorLog.Output(2, trace) //to not get the helper file...
	http.Error(w, http.StatusText(http.StatusInternalServerError),
		http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

//<++++++++++++++++++++++   Query head end   +++++++++++++++++++++++++++>

func (app *application) getRemote(q string) (*templateData, error) {
	td := &templateData{}

	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	url := fmt.Sprintf("http://%s/?q=%s", remoteAddr, q)
	response, err := client.Get(url)
	if err != nil {
		if e, ok := err.(net.Error); ok && e.Timeout() {
			app.errorLog.Printf("%v", err)
			return &templateData{
				NoConnection: "true",
				YesData:      "false",
				Msg:          "Connection to the head end is lost",
			}, nil
		}
		if errors.Is(err, syscall.EACCES) {
			app.errorLog.Printf("%v", err)
			return &templateData{
				YesData:      "false",
				NoConnection: "true",
				Msg:          "Connection access denied",
			}, nil
		}
		if errors.Is(err, syscall.ECONNREFUSED) {
			app.errorLog.Printf("%v", err)
			return &templateData{
				YesData:      "false",
				NoConnection: "true",
				Msg:          "Connection to the head end refused",
			}, nil
		}
		if errors.Is(err, syscall.ECONNRESET) {
			app.errorLog.Printf("%v", err)
			return &templateData{
				YesData:      "false",
				NoConnection: "true",
				Msg:          "Connection reset by the head end",
			}, nil
		}
		if errors.Is(err, syscall.EHOSTDOWN) {
			app.errorLog.Printf("%v", err)
			return &templateData{
				YesData:      "false",
				NoConnection: "true",
				Msg:          "Host is down",
			}, nil
		}
		return td, err
	}
	defer response.Body.Close()
	buf := make([]byte, 1024)
	buf, err = io.ReadAll(response.Body)
	if err != nil {
		return td, err
	}
	var contentsStr = string(buf)
	retPairs := strings.Split(contentsStr, ";")
	if q == "r" {
		if len(retPairs) == 1 {
			return td, fmt.Errorf("%s", retPairs[0])
		}
		for _, pairs := range retPairs {
			pair := strings.Split(pairs, "=")
			if len(pair) != 2 {
				return td, fmt.Errorf("pair failed text %v", pairs)
			}
			switch pair[0] {
			case "airTemp":
				td.AirTemp = pair[1]
			case "sinkTemp":
				td.SinkTemp = pair[1]
			case "ampPower":
				td.AmpPower = pair[1]
			case "doorStatus":
				td.DoorStatus = pair[1]
			case "ampStatus":
				td.AmpStatus = pair[1]
			}
		}
		td.YesData = "true"
		td.NoConnection = "false"
		td.Msg = ""
	}
	return td, nil
}

func (app *application) processSensors(td *templateData) (*templateData, error) {

	ampPower, err := strconv.ParseFloat(td.AmpPower, 64)
	if err != nil {
		td.Msg = "Amp power from the far end was not a number"
		return td, nil
	}
	ampPower *= app.powerFactor
	td.AmpPower = fmt.Sprintf("%0.2f", ampPower)

	sinkTemp, err := strconv.ParseFloat(td.SinkTemp, 64)
	if err != nil {
		td.Msg = "Heatsink temperature from the far end was not a number"
		return td, nil
	}
	sinkTemp = sinkTemp*app.tempFactor*app.sinkFactor - absZero
	td.SinkTemp = fmt.Sprintf("%0.2f", sinkTemp)

	airTemp, err := strconv.ParseFloat(td.AirTemp, 64)
	if err != nil {
		td.Msg = "Air temperature from the far end was not a number"
		return td, nil
	}
	airTemp = airTemp*app.tempFactor*app.airFactor - absZero
	td.AirTemp = fmt.Sprintf("%0.2f", airTemp)

	return td, nil
}
