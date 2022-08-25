package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"

	"path/filepath"
	"runtime/debug"
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
	// var localAddr = "192.168.0.30"
	// localAddress, _ := net.ResolveTCPAddr("tcp", localAddr)
	//
	// // Create a transport like http.DefaultTransport, but with a specified localAddr
	// transport := &http.Transport{
	// 	Proxy: http.ProxyFromEnvironment,
	// 	Dial: (&net.Dialer{
	// 		Timeout:   325 * time.Millisecond,
	// 		KeepAlive: 325 * time.Millisecond,
	// 		LocalAddr: localAddress,
	// 	}).Dial,
	// }
	//
	// client := &http.Client{
	// 	Transport: transport,
	// }
	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	//	var response *http.Response
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
