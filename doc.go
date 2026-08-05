// Package gopiston is a client for the Piston code execution engine
// (https://github.com/engineer-man/piston). It covers the runtimes, execute
// and packages endpoints of the v2 API.
//
// # Choosing an instance
//
// Create a client with [NewClient], passing the base URL of the instance to
// use. The official API at [OfficialAPIBaseURL] has been whitelist-only since
// February 2026 and requires a key supplied with [WithAPIKey], so most callers
// should run their own instance:
//
//	client := gopiston.NewClient("http://localhost:2000/api/v2/")
//
// Package management ([Client.GetPackages], [Client.InstallPackage],
// [Client.UninstallPackage]) operates on an instance's own runtime store and
// is available only when self-hosting. Calling it on a client targeting the
// official API fails with [ErrUnsupportedByOfficialAPI] rather than making a
// request.
//
// # Running code
//
// [Client.Execute] runs a job. An empty version selects the highest installed
// version of the language:
//
//	execution, err := client.Execute(ctx, "python", "",
//		[]gopiston.File{{Content: "print('hello')"}},
//		gopiston.Stdin("input"),
//	)
//	if err != nil {
//		return err
//	}
//	fmt.Println(execution.GetOutput())
//
// Every method takes a [context.Context], which bounds the underlying HTTP
// request. Note that this is separate from the limits Piston applies to the
// code itself, which are set with [RunTimeout] and the other [Param] options.
//
// # Interactive execution
//
// [Client.Connect] opens a WebSocket [Session] instead, which streams output
// as the process produces it and accepts input while it is still running:
//
//	session, err := client.Connect(ctx, "python", "", files)
//	if err != nil {
//		return err
//	}
//	defer session.Close()
//
//	for {
//		event, err := session.Next(ctx)
//		if errors.Is(err, io.EOF) {
//			break // the job finished
//		}
//		if err != nil {
//			return err
//		}
//		if event.Type == gopiston.EventStdout {
//			fmt.Print(event.Data)
//		}
//	}
//
// Like package management, this is unavailable on the official API.
//
// # Errors
//
// A non-2xx response is reported as an [APIError] carrying the status code and
// the instance's own message. Failures can be classified with errors.Is
// against [ErrUnauthorized], [ErrAPIKeyRequired], [ErrRateLimited],
// [ErrBadRequest], [ErrNotFound] and [ErrServer]:
//
//	if errors.Is(err, gopiston.ErrAPIKeyRequired) {
//		// this instance needs a key; pass gopiston.WithAPIKey
//	}
package gopiston
