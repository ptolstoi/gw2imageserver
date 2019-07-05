package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

var (
	contentType = "content-type"
)

func (app *app) initHTTP() {
	app.httpRouter = httprouter.New()
	app.httpRouter.GET("/v1/image/:file", app.serveFile)
}

func (app *app) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Printf("%v %v", req.Method, req.URL)

	app.httpRouter.ServeHTTP(w, req)
}

func (app *app) serveFile(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	noCache := len(r.URL.Query()["noCache"]) != 0
	log.Printf("nocache: %v", noCache)

	parts := strings.SplitN(ps.ByName("file"), ".", 2)
	extension := "png"
	if len(parts) > 1 {
		extension = parts[1]
	}
	fileToServe := parts[0]

	file, err := app.getFileFromCache(fileToServe, extension)

	headers := w.Header()
	headers.Add("content-type", "application/json")

	if err == nil && (file == nil || noCache) {
		file, err = app.noFileInCache(fileToServe, extension)
	}

	if err != nil {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(struct {
			Error string `json:"error"`
		}{
			Error: fmt.Sprintf("error during lookup of file %v: %v", fileToServe, err),
		})
		return
	} else if file == nil {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(struct {
			Error string `json:"error"`
		}{
			Error: "file not found",
		})
		return
	}

	log.Printf("file found: %v %v %v", file.file, file.fileType, file.lastModified)

	if file.fileType == "png" {
		headers.Set(contentType, "image/png")
	} else {
		headers.Set(contentType, "text/plain")
	}

	w.Write(file.content)
}
