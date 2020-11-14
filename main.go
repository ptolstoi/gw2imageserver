package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ptolstoi/neversorrow"
)

var (
	_version   string = "UNSET"
	_buildTime string = "UNSET"

	_showStacktrace string = ""
)

func main() {
	fmt.Printf("\n\n\n\n\n\nStarting GW2ImageServer\n=======================\n")

	listenOn := "localhost:7089"

	if len(os.Args) > 1 {
		listenOn = os.Args[1]
	}

	config := neversorrow.Config{
		Address: neversorrow.EnvOr("ADDRESS", listenOn),

		Version:        _version,
		BuildTime:      _buildTime,
		ShowStacktrace: _showStacktrace == "",
	}

	app, err := newApp(config)
	if err != nil {
		log.Fatalf("couldn't create neversorrow: %v", err)
	}

	// _, _ = http.Get("http://localhost:7089/v1/image/66955.png?noCache")

	if _, err := app.RunUntilSignal(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
