package main

import (
	"fmt"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.CreateDefaultClient()
	output, err := client.Execute("python", "", // Passing language. Since no version is specified, it uses the latest supported version.
		[]piston.Code{
			{Content: "inp = input()\nprint(inp[::-1])"},
		}, // Passing Code.
		piston.Stdin("hello world"), // Passing input as "hello world".
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(output.GetOutput())
}
