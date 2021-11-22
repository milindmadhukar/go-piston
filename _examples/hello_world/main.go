package main

import (
	"log"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.CreateDefaultClient()

	output, err := client.Execute("python", "",
		[]piston.Code{
			{Content: "print('Hello World')"},
		})

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Output is: ", output.GetOutput(), "Language used is: ", output.Language)
}
