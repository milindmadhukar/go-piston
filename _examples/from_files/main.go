package main

import (
	"context"
	"log"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")
	ctx := context.Background()

	files, err := piston.Files("main.py", "test.py")
	if err != nil {
		log.Fatal(err)
	}
	response, err := client.Execute(ctx, "python", "", files)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(response.GetOutput())
}
