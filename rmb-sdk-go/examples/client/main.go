package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
)

func app() error {

	client, err := rmb.Default()

	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const dst = 7 // <- replace this with the twin id of where the service is running
	// it's okay to run both the server and the client behind the same rmb-peer
	var output float64
	if err := client.Call(ctx, dst, nil, "calculator.add", []float64{10, 20}, &output); err != nil {
		return err
	}

	fmt.Printf("output: %f\n", output)

	return nil
}

func main() {
	if err := app(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
