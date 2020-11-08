package main

import (
	"fmt"
	"github.com/ptolstoi/gw2imageserver/gw2dat"
	"log"
	"os"
	"os/signal"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	reader, err := gw2dat.NewGW2DatReader("s:/guild wars 2/gw2.dat")
	if err != nil {
		log.Fatalf("Error when creating a reader: %v", err)
	}
	defer reader.Close()

	log.Printf("Header: %#v", reader.Header())
	log.Printf("MFTHeader: %#v", reader.MFTHeader())

	if reader != nil {
		log.Fatal("Done!")
	}

	fmt.Printf("\n\n\n\n\n\nStarting GW2ImageServer\n=======================\n")

	app := newApp()
	defer app.close()

	app.start()

	// _, _ = http.Get("http://localhost:7089/v1/image/66955.png?noCache")

	stopChannel := make(chan os.Signal, 1)
	signal.Notify(stopChannel, os.Interrupt)

	<-stopChannel

	app.stop()
}
