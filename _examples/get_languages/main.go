package main

import (
	"context"
	"log"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")
	languages := client.GetLanguages(context.Background())

	log.Println("Supported Languages by Piston are: ", *languages)
}
