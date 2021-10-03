package main

import (
	"log"
	"net/http"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.New("", http.DefaultClient)

	output, err := client.Execute("python", "",
		[]piston.Code{
			{Content: "print('Hello World')"},
		}, nil)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Output is: ", output.GetOutput(), "Language used is: ", output.Language)
}
