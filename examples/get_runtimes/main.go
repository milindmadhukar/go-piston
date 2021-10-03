package main

import (
	"log"
	"net/http"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.New("", http.DefaultClient)

	runtimes, err := client.GetRuntimes()

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Runtimes supported by the Piston API are: ", *runtimes)
}
