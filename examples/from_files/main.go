package main

import (
	"log"
	"net/http"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.New("", http.DefaultClient)
	paths := []string{"main.py", "test.py"}
	files, err := piston.Files(paths)
	if err != nil {
		log.Fatal(err)
	}
	output, err := client.Execute("python", "",
		files, nil)

	log.Println(output.GetOutput())
}
