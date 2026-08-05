package main

import (
	"context"
	"fmt"
	"log"

	piston "github.com/milindmadhukar/go-piston/v2"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")

	// This program tries to allocate 200MB, but RunMemoryLimit only allows
	// 16MB, so it is expected to fail.
	output, err := client.Execute(context.Background(), "python", "",
		[]piston.Code{
			{Content: "x = bytearray(200 * 1024 * 1024)\nprint('allocated')"},
		},
		piston.RunMemoryLimit(16*1024*1024),
		piston.CompileMemoryLimit(16*1024*1024),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("exit code:", output.Run.Code)
	fmt.Println("stderr:", output.Run.Stderr)
}
