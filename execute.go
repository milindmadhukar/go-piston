package gopiston

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// latestVersionSelector is the SemVer selector Piston resolves to the highest
// installed version of a language. Sending it lets the server pick the version
// and saves a round trip to /runtimes.
const latestVersionSelector = "*"

// Execute runs code on the Piston instance and returns the result.
//
// language must be the name or an alias of a runtime the instance has
// installed. version is a SemVer selector — a specific version such as
// "3.10.0", a range such as "3.x", or an empty string to use the highest
// installed version.
//
// code is the set of files making up the job; the first is the entry point.
// Build it directly as a []File, or read files from disk with Files.
//
// Optional behavior is supplied through params: Stdin, Args, RunTimeout,
// CompileTimeout, RunCpuTime, CompileCpuTime, RunMemoryLimit and
// CompileMemoryLimit.
func (client *Client) Execute(ctx context.Context, language string, version string, code []File, params ...Param) (*Execution, error) {
	if version == "" {
		version = latestVersionSelector
	}

	reqBody := RequestBody{
		Language: language,
		Version:  version,
		Files:    code,
	}
	processParams(&reqBody, params...)

	bytesBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("piston: encode request: %w", err)
	}

	respBody, err := client.doRequest(ctx, http.MethodPost, client.endpoint("execute"), bytesBody)
	if err != nil {
		return nil, err
	}

	execution := &Execution{}
	if err := decodeJSON(respBody, execution); err != nil {
		return nil, err
	}

	return execution, nil
}

// GetOutput returns the run stage's stdout and stderr, interleaved in the
// order the process produced them.
func (resp *Execution) GetOutput() string {
	return resp.Run.Output
}

// processParams applies the optional parameters to the request body in place.
func processParams(body *RequestBody, params ...Param) {
	p := Params{requestBody: body}
	for _, param := range params {
		param(&p)
	}
}
