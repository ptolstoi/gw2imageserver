package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/ptolstoi/neversorrow"
)

type app struct {
	neversorrow.App

	db *sql.DB

	httpClient *http.Client
}

func newApp(config neversorrow.Config) (*app, error) {
	neversorrowApp, err := neversorrow.New(config)
	if err != nil {
		return nil, err
	}

	app := app{
		App: neversorrowApp,

		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}

	if err := app.initDB(); err != nil {
		return nil, err
	}
	app.initHTTP()

	app.OnClose(app.close)

	return &app, nil
}

func (app *app) close(neversorrow.App) {
	app.closeDB()
}
