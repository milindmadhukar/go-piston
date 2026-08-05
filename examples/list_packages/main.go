package main

import (
	"context"
	"fmt"
	"log"

	piston "github.com/milindmadhukar/go-piston/v2"
)

func main() {
	// Package management is only available on a self-hosted instance.
	client := piston.NewClient("http://localhost:2000/api/v2/")

	packages, err := client.GetPackages(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for _, pkg := range packages {
		fmt.Printf("%s %s (installed: %v)\n", pkg.Language, pkg.LanguageVersion, pkg.Installed)
	}
}
