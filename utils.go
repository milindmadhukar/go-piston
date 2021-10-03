package gopiston

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

/*
Returns the output of the given code.
*/
func (resp *PistonResponse) GetOutput() string {
	return resp.Run.Output
}

/*
Utility method to pass file paths instead of actual code in the string.
Providing a slice of paths will send all the files.
*/
func Files(paths []string) ([]Code, error) {
	var files []Code

	for _, path := range paths {

		fileobj, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		defer fileobj.Close()

		file := Code{
			Name: fileobj.Name(),
		}

		content, err := ioutil.ReadAll(fileobj)

		if err != nil {
			return nil, err
		}

		file.Content = string(content)

		if err != nil {
			return nil, err
		}

		files = append(files, file)
	}

	return files, nil
}

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
func (client *Client) GetLatestVersion(language string) (string, error) {

	runtimes, err := client.GetRuntimes()

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
	case http.StatusNotFound:
		err = errors.New("Invalid language or version")
	case http.StatusBadRequest:
		err = errors.New("Invalid Request. " + respBody)
	default:
		err = errors.New("Unexpected Error. " + respBody)
	}

	return err
}

func (client *Client) handleRequest(method string, url string, body *bytes.Reader) (*http.Response, error) {
	if body == nil {
		body = &bytes.Reader{}
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if apiKey := client.apiKey; apiKey != "" {
		req.Header.Add("Authorization", apiKey)
	}

	resp, err := client.httpClient.Do(req)

	if err != nil {
		return nil, err
	}

	respBody, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	resp.Body.Close()

	resp.Body = ioutil.NopCloser(bytes.NewBuffer(respBody))

	err = handleStatusCode(resp.StatusCode, string(respBody))

	if err != nil {
		return nil, err
	}

	return resp, nil

}
