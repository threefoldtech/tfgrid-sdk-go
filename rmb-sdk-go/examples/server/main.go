package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
)

func app() error {
	router, err := rmb.NewRouter(rmb.DefaultAddress)

	if err != nil {
		return err
	}

	app := router.Subroute("calculator")

	// this function then becomes calculator.add
	app.WithHandler("add", func(ctx context.Context, payload []byte) (interface{}, error) {
		// payload is the entire request data
		// get full request if needed
		msg := rmb.GetRequest(ctx)
		if msg.Schema != rmb.DefaultSchema {
			return nil, fmt.Errorf("expecting schema to be %s", rmb.DefaultSchema)
		}

		var numbers []float64

		if err := json.Unmarshal(payload, &numbers); err != nil {
			return nil, fmt.Errorf("failed to load request payload was expecting list of float: %w", err)
		}

		var result float64
		for _, v := range numbers {
			result += v
		}

		return result, nil
	})

	// this function then becomes calculator.sub
	app.WithHandler("sub", func(ctx context.Context, payload []byte) (interface{}, error) {
		// implement sub here
		return nil, nil
	})

	return router.Run(context.Background())
}

func main() {
	if err := app(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
