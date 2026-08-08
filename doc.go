// Package gopiston is a client for the Piston code execution engine
// (https://github.com/engineer-man/piston). It covers the runtimes, execute,
// packages and operations endpoints of the v2 API, along with the interactive
// WebSocket endpoint.
//
// Piston instances differ in what they serve: the operations endpoints are an
// addition that older instances do not have. Nothing here assumes them —
// [Client.SupportsOperations] reports whether they are available, and every
// other method works against any v2 instance.
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
// # Installing packages in the background
//
// [Client.InstallPackage] is synchronous: it holds the request open until the
// runtime is installed, which for a package compiled from source can be an
// hour. [Client.InstallPackageAsync] starts the same work and returns an
// [Operation] to follow instead, with [Client.GetOperation],
// [Client.GetOperationLog], or [Client.ConnectOperation] for a live stream.
//
// These endpoints are an addition to the v2 API, so probe once and keep the
// answer:
//
//	supported, err := client.SupportsOperations(ctx)
//
// An instance without them answers [ErrOperationsUnsupported]; fall back to
// [Client.InstallPackage], which every instance has.
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
// Closing a session kills the running stage on instances that support it,
// which makes [Session.Close] the portable way to stop a job early;
// [Session.SendSignal] is delivered only by newer instances.
//
// # Errors
//
// A non-2xx response is reported as an [APIError] carrying the status code and
// the instance's own message. Failures can be classified with errors.Is
// against [ErrUnauthorized], [ErrAPIKeyRequired], [ErrRateLimited],
// [ErrBadRequest], [ErrNotFound], [ErrConflict] and [ErrServer]:
//
//	if errors.Is(err, gopiston.ErrAPIKeyRequired) {
//		// this instance needs a key; pass gopiston.WithAPIKey
//	}
//
// Not every failure carries a JSON body. [Client.Execute] answers an internal
// error with a 500 and no body at all, in which case [APIError.Message] is
// empty and Error reports the status text.
//
// # Reading results
//
// Piston sends null for a [Stage] field that does not apply, which Go decodes
// as the zero value. In particular a process killed by a signal reports
// Code as 0, which is indistinguishable from a clean exit — check
// [Stage.Signal] first, and treat a non-empty one as the outcome:
//
//	if execution.Run.Signal != "" {
//		// killed; Code is meaningless here
//	}
package gopiston
