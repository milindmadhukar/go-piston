package main

import (
	"context"
	"log"

	piston "github.com/milindmadhukar/go-piston/v2"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")

	languages, err := client.GetLanguages(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Supported Languages by Piston are: ", languages)
}
