package gw2imageserver

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

type App interface {
	RunUntilSignal() error
}

func NewApp(config neversorrow.Config) (App, error) {
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

func (app *app) RunUntilSignal() error {
	_, err := app.App.RunUntilSignal()
	return err
}

func (app *app) close(neversorrow.App) {
	app.closeDB()
}
