package main

import (
	"log"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.CreateDefaultClient()
	files, err := piston.Files("main.py", "test.py")
	if err != nil {
		log.Fatal(err)
	}
	output, err := client.Execute("python", "",
		files)

	log.Println(output.GetOutput())
}
