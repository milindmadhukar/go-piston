package gopiston

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

// OfficialAPIBaseURL is the base URL of the officially hosted Piston API.
// Access to it requires an API key granted at the maintainer's discretion;
// see the README for details. Most users should self-host Piston instead
// and pass their own instance's base URL to NewClient.
const OfficialAPIBaseURL = "https://emkc.org/api/v2/piston/"

// ClientOption configures a Client created via NewClient.
type ClientOption func(*Client)

// WithAPIKey sets the API key sent with every request, required when using
// the official Piston API.
func WithAPIKey(apiKey string) ClientOption {
	return func(client *Client) {
		client.ApiKey = apiKey
	}
}

// WithHTTPClient overrides the *http.Client used to make requests.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *Client) {
		client.HttpClient = httpClient
	}
}

/*
NewClient creates a Client for the Piston instance at baseURL, typically a
self-hosted instance. To use the official API instead, pass
piston.OfficialAPIBaseURL and provide an API key with WithAPIKey.
*/
func NewClient(baseURL string, opts ...ClientOption) *Client {
	client := &Client{
		HttpClient: http.DefaultClient,
		BaseURL:    baseURL,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

/*
This endpoint will return the supported languages along with the current version and aliases.
To execute code for a particular language using the Execute() function, either the name or one of the aliases must be provided, along with the version. Multiple versions of the same language may be present at the same time, and may be selected when running a job.
*/
func (client *Client) GetRuntimes(ctx context.Context) (*Runtimes, error) {
	resp, err := client.handleRequest(ctx, http.MethodGet, client.BaseURL+"runtimes", nil)
	if err != nil {
		return nil, err
	}

	var runtimes *Runtimes
	err = json.NewDecoder(resp.Body).Decode(&runtimes)
	if err != nil {
		return nil, err
	}

	return runtimes, nil
}

/*
Returns a slice of all the supported languages by the Piston API.
*/
func (client *Client) GetLanguages(ctx context.Context) *[]string {
	var languages []string

	runtimes, _ := client.GetRuntimes(ctx)
	for _, runtime := range *runtimes {
		languages = append(languages, runtime.Language)
	}
	return &languages
}

/*
This endpoint requests execution of some arbitrary code.

language (required) The language to use for execution, must be a string and must be installed.

version (required) The version of the language to use for execution, must be a string containing a SemVer selector for the version or the specific version number to use. If an empty string is passed, the latest version is used.

files (required) A Slice of files containing code or other data that should be used for execution. The first file in this array is considered the main file. The name of the files is optional.
Files can be added using path with the "Files()" method. To provide Code directly, provide a slice of "Code" struct.

Stdin (optional) The text to pass as stdin to the program. Must be a string or left out. Defaults to blank string.

Args (optional) The arguments to pass to the program. Must be an array or left out. Defaults to [].

CompileTimeout (optional) The maximum time allowed for the compile stage to finish before bailing out in milliseconds. Must be a "time.Duration" object. Defaults to 10 seconds.

RunTimeout (optional) The maximum time allowed for the run stage to finish before bailing out in milliseconds. Must be a "time.Duration" object. Defaults to 3 seconds.

CompileMemoryLimit (optional) The maximum amount of memory the compile stage is allowed to use in bytes. Must be a number or left out. Defaults to -1 (no limit)

RunMemoryLimit (optional) The maximum amount of memory the run stage is allowed to use in bytes. Must be a number or left out. Defaults to -1 (no limit)
*/
func (client *Client) Execute(ctx context.Context, language string, version string, code []Code, params ...Param) (*PistonExecution, error) {
	// Initializing the request body.
	reqBody := RequestBody{}

	// Setting language.
	reqBody.Language = language

	// Checking if no version is passed, if true, find the latest version.
	if version == "" {
		langVer, err := client.GetLatestVersion(ctx, language)
		if err != nil {
			return nil, err
		}
		version = langVer

	}

	reqBody.Version = version
	reqBody.Files = code

	reqBody = *processParams(&reqBody, params...)

	// Getting a json bytes.
	bytesBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	body := bytes.NewReader(bytesBody)

	if err != nil {
		return nil, err
	}

	// Sending the POST request to the Piston API.
	resp, err := client.handleRequest(ctx, http.MethodPost, client.BaseURL+"execute", body)
	if err != nil {
		return nil, err
	}

	execution := &PistonExecution{}

	err = json.NewDecoder(resp.Body).Decode(execution)
	if err != nil {
		return nil, err
	}

	return execution, nil
}
