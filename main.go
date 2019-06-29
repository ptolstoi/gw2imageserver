package main

import (
	"net/http"
	"os"
	"os/signal"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	app := newApp()
	defer app.close()

	app.start()

	_, _ = http.Get("http://localhost:7089/v1/image/67000.png")

	stopChannel := make(chan os.Signal, 1)
	signal.Notify(stopChannel, os.Interrupt)

	<-stopChannel

	app.stop()
}
