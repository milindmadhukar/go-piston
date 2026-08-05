package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	// The official Piston API is whitelist-only and requires an API key; see
	// the top-level README for how to obtain one. Read it from the
	// environment rather than hardcoding it.
	client := piston.NewClient(
		piston.OfficialAPIBaseURL,
		piston.WithAPIKey(os.Getenv("PISTON_API_KEY")),
	)

	output, err := client.Execute(context.Background(), "python", "",
		[]piston.Code{
			{Content: "print('Hello from the official API')"},
		})
	if err != nil {
		// This is the failure most readers of this example will hit.
		if errors.Is(err, piston.ErrAPIKeyRequired) {
			log.Fatal("no API key configured; set PISTON_API_KEY, or self-host Piston instead")
		}
		log.Fatal(err)
	}

	fmt.Println(output.GetOutput())
}
