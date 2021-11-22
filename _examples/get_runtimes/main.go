package main

import (
	"log"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.CreateDefaultClient()

	runtimes, err := client.GetRuntimes()

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Runtimes supported by the Piston API are: ", *runtimes)
}
