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

	log.Println("Runtimes supported by the Piston API are: ", *runtimes)
}
