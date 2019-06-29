package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

type app struct {
	db         *sql.DB
	listenOn   string
	server     *http.Server
	httpRouter *httprouter.Router
	httpClient *http.Client
}

func newApp() *app {
	listenOn := "localhost:7089"

	if len(os.Args) > 1 {
		listenOn = os.Args[1]
	}

	app := app{
		listenOn: listenOn,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}

	app.server = &http.Server{
		Handler: &app,
	}

	app.initDB()
	app.initHTTP()

	return &app
}

func (app *app) start() {
	networkSocketType := "unix"
	if strings.Contains(app.listenOn, ":") {
		networkSocketType = "tcp"
	}

	listener, err := net.Listen(networkSocketType, app.listenOn)
	if err != nil {
		panic(err)
	}

	go func() {
		log.Printf("listening on %v\n", listener.Addr())
		app.server.Serve(listener)
	}()

}

func (app *app) stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	if err := app.server.Shutdown(ctx); err != nil {
		log.Printf("Error when shutting down: %v", err)
	}

	cancel()
}

func (app *app) close() {
	app.closeDB()
}
