package main

import (
	"context"
	"fmt"
	"log"
	"time"

	piston "github.com/milindmadhukar/go-piston/v2"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")

	// This program sleeps for 3 seconds, but RunTimeout only allows 2, so it
	// will be killed before it can print "after sleep".
	output, err := client.Execute(context.Background(), "python", "",
		[]piston.Code{
			{Content: "import time\nprint('before sleep')\ntime.sleep(3)\nprint('after sleep')"},
		},
		piston.RunTimeout(2*time.Second),
		piston.CompileTimeout(10*time.Second),
		piston.RunCpuTime(2*time.Second),
		piston.CompileCpuTime(10*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("stdout:", output.Run.Stdout)
	fmt.Println("signal:", output.Run.Signal)
}
