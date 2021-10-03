package main

import (
	"log"
	"net/http"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.New("", http.DefaultClient)
	languages := client.GetLanguages()

	log.Println("Supported Languages by Piston are: ", *languages)
}
