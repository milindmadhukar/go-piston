package main

import (
	"log"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.CreateDefaultClient()
	languages := client.GetLanguages()

	log.Println("Supported Languages by Piston are: ", *languages)
}
