package gopiston

import (
	"net/http"
)

/*
Client struct to allow the usage of Piston API endpoints.
*/
type Client struct {
	HttpClient *http.Client
	BaseURL    string
	ApiKey     string
}

/*
Slice Struct that holds all the supported runtimes by the Piston API.
*/
type Runtimes []struct {
	Language string   `json:"language"`
	Version  string   `json:"version"`
	Aliases  []string `json:"aliases"`
	Runtime  string   `json:"runtime,omitempty"`
}

/*
Struct for storing the name of the file and content.
*/
type Code struct {
	Name    string `json:"name,omitempty"`
	Content string `json:"content"`
}

/*
Request Body that is sent over to the Piston API for the execute endpoint.
*/
type RequestBody struct {
	Language           string   `json:"language"`
	Version            string   `json:"version"`
	Files              []Code   `json:"files"`
	Stdin              string   `json:"stdin,omitempty"`
	Args               []string `json:"args,omitempty"`
	CompileTimeout     int      `json:"compile_timeout,omitempty"`
	RunTimeout         int      `json:"run_timeout,omitempty"`
	CompileMemoryLimit int      `json:"compile_memory_limit,omitempty"`
	RunMemoryLimit     int      `json:"run_memory_limit,omitempty"`
}

/*
Response Received from the Piston API.
*/
type PistonExecution struct {
	Language string `json:"language"`
	Version  string `json:"version"`
	Run      struct {
		Stdout string `json:"stdout,omitempty"`
		Stderr string `json:"stderr,omitempty"`
		Output string `json:"output,omitempty"`
		Code   int    `json:"code,omitempty"`
		Signal string `json:"signal,omitempty"`
	} `json:"run"`
}
