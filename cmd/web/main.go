package main

import (
	//	"flag"

	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

const pingCount = 50

//The design of this program is along the lines of Alex Edward's
//Let's Go except since it is a single user local program, it
//ignore the rules for a shared over the internet application

// for injecting data into handlers
type application struct {
	powerFactor float64 //multiply by A/D output to get the power
	tempFactor  float64
	//multiply by A/D output and airFactor then subtract absolute zero to get air temparture
	//multiply by A/D output and sinkFactor then subtract absolute zero to get sink temparture
	airFactor     float64
	sinkFactor    float64
	errorLog      *log.Logger
	infoLog       *log.Logger
	debugOption   bool
	templateCache map[string]*template.Template
	tempThreshold float64
	remoteOn      bool
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
	MaxPowerIndicator float64 `yaml:"maxPowerIndicator"`
	TempThreshold     float64 `yaml:"tempThreshold"`
}

func main() {
	var err error

	optionDebug := flag.Bool("d", false, "true turns on debug option")
	flag.Parse()

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
		debugOption:   *optionDebug,
		templateCache: templateCache,
		tempThreshold: tempThreshold, //small window turns read above this threshold
		remoteOn:      false,
	}
	err = app.adjust()
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < pingCount; i++ {
		_, err = app.getRemote("p")
		if err == nil {
			fmt.Println("Count", i)
			break
		}
	}
	fmt.Println(err)

	mux := app.routes()
	srv := &http.Server{
		Addr:     ":4000",
		ErrorLog: errorLog,
		Handler:  mux,
	}
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))
	infoLog.Printf("starting server on :4000")
	go srv.ListenAndServe()
	// err = srv.ListenAndServe()
	// errorLog.Fatal(err)
	app.little()
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
	mux.HandleFunc("/readjust", app.reAdjust)
	return mux
}
