package main

import (
	//	"flag"

	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"syscall"

	"gopkg.in/yaml.v2"
)

//The design of this program is along the lines of Alex Edward's
//Let's Go except since it is a single user local program, it
//ignore the rules for a shared over the internet application

//for injecting data into handlers
type application struct {
	powerFactor   float64
	tempFactor    float64
	airFactor     float64
	sinkFactor    float64
	errorLog      *log.Logger
	infoLog       *log.Logger
	templateCache map[string]*template.Template
}

type configType struct {
	AbsZero           float64 `yaml:"absZero"`
	CalTemp           float64 `yaml:"calTemp"`
	CalVoltage        float64 `yaml:"calVoltage"`
	AirFactor         float64 `yaml:"airFactor"`
	SinkFactor        float64 `yaml:"sinkFactor"`
	PlusFive          float64 `yaml:"plusFive"`
	MaxAtoD           float64 `yaml:"maxAtoD"`
	MaxPower          float64 `yaml:"maxPower"`
	MaxPowerIndicator float64 `yaml:"maxPowerIndictor`
}

func main() {
	var err error

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime|log.LUTC)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.LUTC|log.Llongfile)

	//note, this requires the run command be issues from the project base
	templateCache, err := newTemplateCache("./ui/html/")
	if err != nil {
		errorLog.Fatal(err)
	}

	app := &application{
		powerFactor:   (plusFive / maxAtoD) * (maxPower / maxPowerIndicator),     //for converting A/D reads into power
		tempFactor:    (plusFive / maxAtoD) * ((calTemp + absZero) / calVoltage), //for converting A/D reads to deg K
		airFactor:     airFactor,
		sinkFactor:    sinkFactor,
		errorLog:      errorLog,
		infoLog:       infoLog,
		templateCache: templateCache,
	}
	configFlag := true
	config := &configType{}
	configPath := os.Getenv("EME_Base")
	configData, err := os.ReadFile(fmt.Sprintf("%s/adjust.yaml", configPath))
	if err != nil {
		if errors.Is(err, syscall.ENOENT) { //if no yaml file, keep the previous configuration numbers
			configFlag = false
		} else {
			errorLog.Fatal(err)
		}
	}
	if configFlag {
		err = yaml.Unmarshal(configData, config)
		if err != nil {
			errorLog.Fatal(err)
		}
		app.powerFactor = (config.PlusFive / config.MaxAtoD) * (config.MaxPower / config.MaxPowerIndicator)
		app.tempFactor = (config.PlusFive / config.MaxAtoD) * ((config.CalTemp + config.AbsZero) / config.CalVoltage)
		app.airFactor = config.AirFactor
		app.tempFactor = config.SinkFactor
	}

	mux := app.routes()
	srv := &http.Server{
		Addr:     ":4000",
		ErrorLog: errorLog,
		Handler:  mux,
	}
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))
	infoLog.Printf("starting server on :4000")
	err = srv.ListenAndServe()
	errorLog.Fatal(err)
}

func (app *application) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", app.home)
	mux.HandleFunc("/home", app.home)
	mux.HandleFunc("/monitor", app.monitor)
	mux.HandleFunc("/update-monitor", app.updateMonitor)
	mux.HandleFunc("/ampOn", app.ampOn)
	mux.HandleFunc("/ampOff", app.ampOff)
	mux.HandleFunc("/adjustments", app.adjustments)
	return mux
}
