package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	piston "github.com/milindmadhukar/go-piston/v2"
)

func main() {
	// Points at a self-hosted Piston instance, with a custom *http.Client
	// (e.g. to set a request timeout).
	client := piston.NewClient(
		"http://localhost:2000/api/v2/",
		piston.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
	)

	output, err := client.Execute(context.Background(), "python", "",
		[]piston.Code{
			{Content: "print('Hello from a self-hosted instance')"},
		})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(output.GetOutput())
}
