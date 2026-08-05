package main

import (
	"context"
	"fmt"
	"log"
	"os"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	// The official Piston API requires an API key; see the top-level
	// README for how to obtain one. Read it from an environment variable
	// rather than hardcoding it.
	client := piston.NewClient(
		piston.OfficialAPIBaseURL,
		piston.WithAPIKey(os.Getenv("PISTON_API_KEY")),
	)

	output, err := client.Execute(context.Background(), "python", "",
		[]piston.Code{
			{Content: "print('Hello from the official API')"},
		})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(output.GetOutput())
}
