package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Printf("\n\n\n\n\n\nStarting GW2ImageServer\n=======================\n")

	app := newApp()
	defer app.close()

	app.start()

	_, _ = http.Get("http://localhost:7089/v1/image/66955.png?noCache")

	stopChannel := make(chan os.Signal, 1)
	signal.Notify(stopChannel, os.Interrupt)

	<-stopChannel

	app.stop()
}
