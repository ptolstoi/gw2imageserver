package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ptolstoi/neversorrow"
	"github.com/ptolstoi/neversorrow/errors"
)

var (
	contentType = "content-type"
)

func (app *app) initHTTP() {
	app.AddRoute("GET", "/v1/image/:file", app.serveImage)
}

func (app *app) serveImage(ctx neversorrow.Context) {
	noCache := len(ctx.Request().URL.Query()["noCache"]) != 0

	parts := strings.SplitN(ctx.Params()["file"], ".", 2)
	extension := "png"
	if len(parts) > 1 {
		extension = parts[1]
	}
	fileToServe := parts[0]

	file, err := app.getFileFromCache(fileToServe, extension)

	if err == nil && (file == nil || noCache) {
		file, err = app.noImageFileInCache(fileToServe, extension)
	}

	if err != nil {
		errorFromCache := fmt.Sprintf("error during lookup of file %v: %v", fileToServe, err)

		ctx.Error(errors.NewWithCode(errorFromCache, http.StatusInternalServerError))

		return
	} else if file == nil {

		ctx.Error(errors.NewWithCode("file not found", http.StatusNotFound))
		return
	}

	log.Printf("[serveFile] file found: %v %v %v", file.file, file.fileType, file.lastModified)

	resp := ctx.ResponseWriter()

	if file.fileType == "png" {
		resp.Header().Set(contentType, "image/png")
	} else {
		resp.Header().Set(contentType, "text/plain")
	}

	resp.Write(file.content)
}
