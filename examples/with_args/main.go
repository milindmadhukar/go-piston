package main

import (
	"context"
	"fmt"
	"log"

	piston "github.com/milindmadhukar/go-piston/v2"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")

	output, err := client.Execute(context.Background(), "python", "",
		[]piston.File{
			{Content: "import sys\nprint(sys.argv[1:])"},
		},
		piston.Args([]string{"foo", "bar"}), // Passing command line arguments.
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(output.GetOutput())
}
