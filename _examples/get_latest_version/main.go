package main

import (
	"context"
	"log"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")
	lang := "python"

	latestVersion, err := client.GetLatestVersion(context.Background(), lang)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("The latest version of", lang, " supported by Piston API is", latestVersion)
}
