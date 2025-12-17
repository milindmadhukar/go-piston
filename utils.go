package gopiston

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
)

func processParams(body *RequestBody, params ...Param) *RequestBody {
	p := Params{
		requestBody: body,
	}
	for _, param := range params {
		param(&p)
	}

	return body
}

/*
Returns the output of the given code.
*/
func (resp *PistonExecution) GetOutput() string {
	return resp.Run.Output
}

/*
Utility method to pass file paths instead of actual code in the string.
Providing a slice of paths will send all the files.
*/
func Files(paths ...string) ([]Code, error) {
	var files []Code

	for _, path := range paths {
		fileobj, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		file := Code{
			Name: fileobj.Name(),
		}

		content, err := io.ReadAll(fileobj)
		fileobj.Close()
		if err != nil {
			return nil, err
		}

		file.Content = string(content)

		files = append(files, file)
	}

	return files, nil
}

// Returns a boolean value checking if a string is found in the slice or not.
func isPresent(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

/*
Returns the latest version of the language supported by the Piston API.
*/
func (client *Client) GetLatestVersion(ctx context.Context, language string) (string, error) {
	runtimes, err := client.GetRuntimes(ctx)
	if err != nil {
		return "", err
	}

	for _, runtime := range *runtimes {
		if language == runtime.Language || isPresent(runtime.Aliases, language) {
			return runtime.Version, nil
		}
	}

	return "", errors.New("Could not find a version for the language " + language)
}

// Handles the various status codes from the Piston API.
func handleStatusCode(code int, respBody string) error {
	var err error

	if code < 300 && code >= 200 {
		return nil
	}

	switch code {
	case http.StatusTooManyRequests:
		err = errors.New("You have been ratelimited.Try again later")
	case http.StatusInternalServerError:
		err = errors.New("Server failed to respond. Try again later")
	case http.StatusBadRequest:
		err = errors.New("Invalid Request. The language or version may be incorrect.")
	case http.StatusNotFound:
		err = errors.New("Not found." + respBody)
	default:
		err = errors.New("Unexpected Error. " + respBody)
	}

	return err
}

// Handles sending the request to the Piston API and returning a response.
func (client *Client) handleRequest(ctx context.Context, method string, url string, body *bytes.Reader) (*http.Response, error) {
	if body == nil {
		body = &bytes.Reader{}
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	if apiKey := client.ApiKey; apiKey != "" {
		req.Header.Add("Authorization", apiKey)
	}

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	resp.Body.Close()

	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	err = handleStatusCode(resp.StatusCode, string(respBody))
	if err != nil {
		return nil, err
	}

	return resp, nil
}
