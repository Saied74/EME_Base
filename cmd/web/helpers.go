package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"image/color"
	//"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"path/filepath"
	"runtime/debug"

	"gopkg.in/yaml.v2"
	//		"fyne.io/fyne/v2/data/binding"
)

const (
	remoteAddr        = "192.168.6.97" //"192.168.4.72" //TODO: need to change it to a server name
	absZero           = 273.15         //per Lord Kelvin
	calTemp           = 25.0           //per TI datasheet
	calVoltage        = 2.982          //per TI datasheet
	airFactor         = 1.002          //correction factor for air temperature sensor relative to thermocouple
	sinkFactor        = 1.007          //correction factor for the heatsink temperature sensor relative to thermocouple
	plusFive          = 4.94           //Arduino Nano measured reference voltage.
	maxAtoD           = 1023.0         //10 bits all ones.
	maxPower          = 1000.0         //assumed
	maxPowerIndicator = 5.0            //assumed
	tempThreshold     = 50.0           //threshold at which user will be warned
	retries           = 10
	//ampThreshold := 66.0; //votage value for 66 degrees C temperature
	//TODO: need to build an alarm for high temperature
)

// <+++++++++++++++++++++++ Template Processing +++++++++++++++++++++++++++>

// This is straight out of Alex Edward's Let's Go book
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

// This is straight out of Alex Edward's Let's Go book
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

// This is straight out of Alex Edward's Let's Go book
func (app *application) serverError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.errorLog.Output(2, trace) //to not get the helper file...
	http.Error(w, http.StatusText(http.StatusInternalServerError),
		http.StatusInternalServerError)
}

// This is straight out of Alex Edward's Let's Go book
func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

// This is straight out of Alex Edward's Let's Go book
func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

//<++++++++++++++++++++++   Query head end   +++++++++++++++++++++++++++>

// Makes an HTTP call to the far end with the parameter given.  In most cases,
// it does not return an error (see below) but populates the message field.
// return data is raw A/D convertor data
func (app *application) getRemote(q string) (*templateData, error) {
	td := &templateData{}

	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	// TODO: replace remote address with server name
	//better yet, make the server name or IP address a command line flag
	url := fmt.Sprintf("http://%s/?q=%s", remoteAddr, q)
	response, err := client.Get(url)
	for i := 0; i < retries && err != nil; i++ {
		response, err = client.Get(url)
	}
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			app.errorLog.Printf("%v", err)
			return &templateData{
				NoConnection: "true",
				YesData:      "false",
				Msg:          "Context deadline exceeded (1)",
			}, nil
		}
		if e, ok := err.(net.Error); ok && e.Timeout() { //timeout error
			app.errorLog.Printf("%v", err)
			return &templateData{
				NoConnection: "true",
				YesData:      "false",
				Msg:          "Connection to the head end is lost",
			}, nil
		}
		if errors.Is(err, syscall.EACCES) { //Access denied
			app.errorLog.Printf("%v", err)
			return &templateData{
				YesData:      "false",
				NoConnection: "true",
				Msg:          "Connection access denied",
			}, nil
		}
		if errors.Is(err, syscall.ECONNREFUSED) { //connection refused
			app.errorLog.Printf("%v", err)
			return &templateData{
				YesData:      "false",
				NoConnection: "true",
				Msg:          "Connection to the head end refused",
			}, nil
		}
		if errors.Is(err, syscall.ECONNRESET) { //connecton reset
			app.errorLog.Printf("%v", err)
			return &templateData{
				YesData:      "false",
				NoConnection: "true",
				Msg:          "Connection reset by the head end",
			}, nil
		}
		if errors.Is(err, syscall.EHOSTDOWN) { //host down
			//app.errorLog.Printf("%v", err)
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
	retPairs := strings.Split(contentsStr, ";") //all key=value pairs are ; seperated
	if q == "r" {                               //process this way if the request was a quary
		if len(retPairs) == 1 {
			return td, fmt.Errorf("%s", retPairs[0])
		}
		for _, pairs := range retPairs {
			pair := strings.Split(pairs, "=") //split the key=value pairs
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
			case "refPower":
				td.RefPower = pair[1]
			case "fan1Curr":
				td.Fan1 = pair[1]
			case "fan2Curr":
				td.Fan2 = pair[1]
			case "doorStatus":
				td.DoorStatus = pair[1]
			case "pttStatus":
				td.PttStatus = pair[1]
			}
		}
		td.YesData = "true"
		td.NoConnection = "false"
		td.Msg = ""
	}
	return td, nil
}

// take the raw A/D data from the app structure and make it usable.
// See the Adjustments page for details.
func (app *application) processSensors(td *templateData) (*templateData, error) {

	ampPower, err := strconv.ParseFloat(td.AmpPower, 64)
	if err != nil {
		td.Msg = "Amp power from the far end was not a number"
		return td, nil
	}
	ampPower *= app.powerFactor
	td.AmpPower = fmt.Sprintf("%0.2f", ampPower)

	refPower, err := strconv.ParseFloat(td.RefPower, 64)
	if err != nil {
		td.Msg = "Reflected power from the far end was not a number"
		return td, nil
	}
	refPower *= app.powerFactor
	td.RefPower = fmt.Sprintf("%0.2f", refPower)
	//if refPower < 2 {
	//	refPower = 2
	//}
	g := refPower / ampPower
	swr := (1 + g) / (1 - g)
	td.SWR = fmt.Sprintf("%0.2f", swr)

	sinkTemp, err := strconv.ParseFloat(td.SinkTemp, 64)
	if err != nil {
		td.Msg = "Heatsink temperature from the far end was not a number"
		return td, nil
	}
	sinkTemp = sinkTemp*app.tempFactor*app.sinkFactor - absZero
	if sinkTemp > sinkTempThreshold {
		sinkExternalColor = color.White
	} else {
		sinkExternalColor = color.White
		}

	td.SinkTemp = fmt.Sprintf("%0.2f", sinkTemp)

	airTemp, err := strconv.ParseFloat(td.AirTemp, 64)
	if err != nil {
		td.Msg = "Air temperature from the far end was not a number"
		return td, nil
	}
	airTemp = airTemp*app.tempFactor*app.airFactor - absZero
		if airTemp > airTempThreshold {
		airExternalColor = red
	} else {
		airExternalColor = color.White
			}

	td.AirTemp = fmt.Sprintf("%0.2f", airTemp)

	return td, nil
}

// read adjust.yaml file and change the Adjustment parmeters accordingly.
func (app *application) adjust() error {
	config := &configType{}

	emePath := os.Getenv("EMEPATH")
	configPath := filepath.Join(emePath, "adjust.yaml")

	configData, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, syscall.ENOENT) { //if no yaml file, keep the previous configuration numbers
			fmt.Println("No adjust.yaml file found", err)
			return nil
		}
		return err
	}
	err = yaml.Unmarshal(configData, config)
	if err != nil {
		return err
	}

	app.powerFactor = (config.PlusFive / config.MaxAtoD) * (config.MaxPower / config.MaxPowerIndicator)
	app.tempFactor = (config.PlusFive / config.MaxAtoD) * ((config.CalTemp + config.AbsZero) / config.CalVoltage)
	app.airFactor = config.AirFactor
	app.sinkFactor = config.SinkFactor
	app.tempThreshold = config.TempThreshold
	if app.debugOption {
		fmt.Printf("<-----------Adjustment values---------------->\n")
		fmt.Printf("Absolute Zero: %0.2f\n", config.AbsZero)
		fmt.Printf("Cal Temp: %0.2f\n", config.CalTemp)
		fmt.Printf("Cal Voltage: %0.2f\n", config.CalVoltage)
		fmt.Printf("Air Factor: %0.2f\n", config.AirFactor)
		fmt.Printf("Sink Factor: %0.2f\n", config.SinkFactor)
		fmt.Printf("Plus Five: %0.2f\n", config.PlusFive)
		fmt.Printf("Max A/D: %0.2f\n", config.MaxAtoD)
		fmt.Printf("Max Power: %0.2f\n", config.MaxPower)
		fmt.Printf("Max Power Indicator: %0.2f\n", config.MaxPowerIndicator)
		fmt.Printf("Power Factor: %0.3f\n", app.powerFactor)
		fmt.Printf("Temp Factor: %0.3f\n", app.tempFactor)
		fmt.Printf("Air Factor: %0.3f\n", app.airFactor)
		fmt.Printf("Sink Factor: %0.3f\n", app.sinkFactor)
		fmt.Printf("Temperature Threshold: %0.3f\n", app.tempThreshold)
	}
	return nil
}

// Wrapper for getRemote and processSensors since they are often called together.
func (app *application) updateSensors() (*templateData, error) {
	td, err := app.getRemote("r") //r for report (status
	if err != nil {
		return &templateData{}, err
	}
	if err == nil && td.Msg == "" {
		td, err = app.processSensors(td)
		if err != nil {
			return &templateData{}, err
		}
	}
	return td, nil
}

func (app *application) updateBoundSensors() error {

	td, err := app.getRemote("r")
	if err != nil {
		return err
	}

	if err == nil && td.Msg == "" {
		td, err = app.processSensors(td)
		if err != nil {
			return err
		}
		td.PttStatus = strings.TrimSuffix(td.PttStatus, "\n")
		ampStatus.Set(td.AmpStatus)
		ampPower.Set(td.AmpPower)
		refPower.Set(td.RefPower)
		swr.Set(td.SWR)
		airTemp.Set(td.AirTemp)
		sinkTemp.Set(td.SinkTemp)
		fan1.Set(td.Fan1)
		fan2.Set(td.Fan2)
		doorStatus.Set(td.DoorStatus)
		pttStatus.Set(td.PttStatus)
		airBoundExternalColor.Reload()
		sinkBoundExternalColor.Reload()

	}
	return nil
}
