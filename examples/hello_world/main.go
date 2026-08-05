package main

import (
	"context"
	"log"

	piston "github.com/milindmadhukar/go-piston/v2"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")

	output, err := client.Execute(context.Background(), "python", "",
		[]piston.File{
			{Content: "print('Hello World')"},
		})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Output is: ", output.GetOutput(), "Language used is: ", output.Language)
}
