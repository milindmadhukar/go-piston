package main

import (
	"context"
	"fmt"
	"log"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")
	output, err := client.Execute(context.Background(), "python", "", // Passing language. Since no version is specified, it uses the latest supported version.
		[]piston.Code{
			{Content: "inp = input()\nprint(inp[::-1])"},
		}, // Passing Code.
		piston.Stdin("hello world"), // Passing input as "hello world".
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(output.GetOutput())
}
