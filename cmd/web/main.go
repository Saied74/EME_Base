package main

import (
	//	"flag"

	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const pingCount = 50

//The design of this program is along the lines of Alex Edward's
//Let's Go except since it is a single user local program, it
//ignore the rules for a shared over the internet application

// for injecting data into handlers
type application struct {
	voltageFactor float64 //multiply remote sensor read to get voltage value
	powerFactor   float64 //no longer used
	tempFactor    float64 //multiply the voltage to get temperature
	airFactor     float64 //set to 1.0, useless
	sinkFactor    float64 //set to 1.0, useless
	errorLog      *log.Logger
	infoLog       *log.Logger
	debugOption   bool
	templateCache map[string]*template.Template
	airThreshold  float64
	sinkThreshold float64
	remoteOn      bool
	remoteAddress string
	dish          int //1 = 3m dish, 2 = 15' dish
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
	AirThreshold      float64 `yaml:"airThreshold"`
	SinkThreshold     float64 `yaml:"sinkThreshold"`
	RemoteAddress1    string  `yaml:"remoteAddress1"`
	RemoteAddress2    string  `yaml:"remoteAddress2"`
}

func main() {
	var err error

	optionDebug := flag.Bool("d", false, "true turns on debug option")
	dishChoice := flag.Int("dish", 1, "1 for 3m 2 for 15' dish")
	timeOut := flag.Int("timeout", 500, "https timeout in milliseconds")
	flag.Parse()

	if !(*dishChoice == 1 || *dishChoice == 2) {
		fmt.Printf("%d\t%T\n", *dishChoice, *dishChoice)
		log.Fatal("Dish choices are 1 or 2")
	}

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime|log.LUTC)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.LUTC|log.Llongfile)

	//note, this requires the run command be issues from the project base
	emePath := os.Getenv("EMEPATH")
	cachePath := filepath.Join(emePath, "ui/html/")

	templateCache, err := newTemplateCache(cachePath)
	if err != nil {
		errorLog.Fatal(err)
	}
	timeout = *timeOut

	fmt.Printf("Timeout: %d, Dish: %d\n", timeout, *dishChoice)

	app := &application{
		voltageFactor: (plusFive / maxAtoD),                                  //for converting remote-sensor-read to voltage
		powerFactor:   (plusFive / maxAtoD) * (maxPower / maxPowerIndicator), //for converting A/D reads into power
		tempFactor:    (plusFive / maxAtoD) * 100.0,                          //for converting voltage reads to deg C
		airFactor:     airFactor,
		sinkFactor:    sinkFactor,
		errorLog:      errorLog,
		infoLog:       infoLog,
		debugOption:   *optionDebug,
		templateCache: templateCache,
		airThreshold:  airTempThreshold, //small window turns read above this threshold
		sinkThreshold: sinkTempThreshold,
		remoteOn:      false,
		dish:          *dishChoice,
		//remoteAddress: remoteAddress,
	}
	err = app.adjust()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("read and processed adjust Dish: %s\n", app.remoteAddress)
	/*
		for i := 0; i < pingCount; i++ {
			_, err = app.getRemote("p")
			if err == nil {
				fmt.Println("Count", i)
				break
			}
		}
		fmt.Println(err)
	*/
	mux := app.routes()
	srv := &http.Server{
		Addr:     ":4000",
		ErrorLog: errorLog,
		Handler:  mux,
	}
	staticPath := filepath.Join(emePath, "ui/static/")
	fileServer := http.FileServer(http.Dir(staticPath))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))
	infoLog.Printf("starting server on :4000")
	go srv.ListenAndServe()

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
