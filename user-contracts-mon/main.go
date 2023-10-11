package main

import (
	"log"

	app "github.com/threefoldtech/tfgrid-sdk-go/user-contracts-mon/app"
)

func main() {
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
