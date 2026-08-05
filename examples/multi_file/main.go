package main

import (
	"context"
	"fmt"
	"log"

	piston "github.com/milindmadhukar/go-piston/v2"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")

	// Unlike from_files, which reads source from disk, this builds
	// multiple in-memory files directly. The first file is the entrypoint;
	// the rest are made available alongside it (e.g. as importable modules).
	output, err := client.Execute(context.Background(), "python", "",
		[]piston.Code{
			{Name: "main.py", Content: "from helper import shout\nprint(shout('hi'))"},
			{Name: "helper.py", Content: "def shout(s):\n    return s.upper()"},
		})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(output.GetOutput())
}
