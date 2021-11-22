package gopiston

import (
	"bytes"
	"encoding/json"
	"net/http"
)

/*
Creates a default client object and returns it for access to the methods.
*/
func CreateDefaultClient() *Client {
	return &Client{
		httpClient: http.DefaultClient,
		baseUrl:    "https://emkc.org/api/v2/piston/",
		apiKey:     "",
	}
}

/*
Creates a Client object which allows the use of custom url and api key.
*/
func New(apiKey string, httpClient *http.Client, baseUrl string) *Client {
	return &Client{
		httpClient: httpClient,
		baseUrl:    baseUrl,
		apiKey:     apiKey,
	}
}

/*
This endpoint will return the supported languages along with the current version and aliases.
To execute code for a particular language using the Execute() function, either the name or one of the aliases must be provided, along with the version. Multiple versions of the same language may be present at the same time, and may be selected when running a job.
*/
func (client *Client) GetRuntimes() (*Runtimes, error) {
	resp, err := client.handleRequest("GET", client.baseUrl+"runtimes", nil)
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
func (client *Client) GetLanguages() *[]string {
	var languages []string

	runtimes, _ := client.GetRuntimes()
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
func (client *Client) Execute(language string, version string, code []Code, params ...Param) (*PistonResponse, error) {
	// Initializing the request body.
	reqBody := RequestBody{}

	// Setting language.
	reqBody.Language = language

	// Checking if no version is passed, if true, find the latest version.
	if version == "" {

		langVer, err := client.GetLatestVersion(language)
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
	resp, err := client.handleRequest("POST", client.baseUrl+"execute", body)
	if err != nil {
		return nil, err
	}

	output := &PistonResponse{}

	err = json.NewDecoder(resp.Body).Decode(output)

	if err != nil {
		return nil, err
	}

	return output, nil

}
