package main

import (
	"os"
	"os/signal"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	app := newApp()
	defer app.close()

	app.start()

	stopChannel := make(chan os.Signal, 1)
	signal.Notify(stopChannel, os.Interrupt)

	<-stopChannel

	app.stop()
}
