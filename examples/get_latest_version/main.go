package main

import (
	"log"
	"net/http"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.GetDefaultClient(http.DefaultClient)
	lang := "python"

	latestVersion, err := client.GetLatestVersion(lang)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("The latest version of", lang, " supported by Piston API is", latestVersion)
}
