package main

import (
	"context"
	"log"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")

	runtimes, err := client.GetRuntimes(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for _, runtime := range runtimes {
		log.Printf("%s %s (aliases: %v)", runtime.Language, runtime.Version, runtime.Aliases)
	}
}
