package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")

	execution, err := client.Execute(context.Background(), "python", "",
		[]piston.Code{
			{Content: "print('hello')"},
		})
	if err != nil {
		handle(err)
		os.Exit(1)
	}

	fmt.Println(execution.GetOutput())
}

func handle(err error) {
	// Sentinel errors classify the failure, so callers can react to it
	// without parsing the message.
	switch {
	case errors.Is(err, piston.ErrAPIKeyRequired):
		log.Println("this instance needs an API key; pass piston.WithAPIKey(...)")
	case errors.Is(err, piston.ErrUnauthorized):
		log.Println("the configured API key was rejected")
	case errors.Is(err, piston.ErrRateLimited):
		log.Println("rate limited; back off and retry")
	case errors.Is(err, piston.ErrBadRequest):
		log.Println("the instance rejected the request; check the language and version")
	case errors.Is(err, piston.ErrLanguageNotFound):
		log.Println("that language is not installed on this instance")
	case errors.Is(err, piston.ErrServer):
		log.Println("the instance failed to handle the request; try again later")
	default:
		log.Println("unexpected failure:", err)
	}

	// APIError carries the status code, the instance's own message, and the
	// raw body for anything the sentinels do not cover.
	var apiErr *piston.APIError
	if errors.As(err, &apiErr) {
		log.Printf("status=%d message=%q", apiErr.StatusCode, apiErr.Message)
	}
}
