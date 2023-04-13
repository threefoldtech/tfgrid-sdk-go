package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello "+os.Getenv("USERNAME"))
	})

	fmt.Println("Server is listening on 3000")
	err := http.ListenAndServe(":3000", nil)

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Print("server closed\n")
	} else if err != nil {
		fmt.Print("error starting server: \n", err)
		os.Exit(1)
	}
}
