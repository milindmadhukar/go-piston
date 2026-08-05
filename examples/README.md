# Examples

Each directory below is a standalone, runnable `go run main.go` program demonstrating one feature of `go-piston`. Unless noted otherwise, they point at a self-hosted Piston instance running on `http://localhost:2000`.

| Example | Description |
| --- | --- |
| [hello_world](hello_world) | Minimal example: execute an inline snippet and print its output |
| [with_input](with_input) | Pass stdin to the executed program via `piston.Stdin` |
| [with_args](with_args) | Pass command line arguments via `piston.Args` |
| [with_timeouts](with_timeouts) | Compile/run timeouts and CPU-time limits |
| [with_memory_limits](with_memory_limits) | Compile/run memory limits |
| [from_files](from_files) | Read code from files on disk with `piston.Files` |
| [multi_file](multi_file) | Execute multiple in-memory files together |
| [get_runtimes](get_runtimes) | List every supported language, version, and alias |
| [get_languages](get_languages) | List just the supported language names |
| [get_latest_version](get_latest_version) | Resolve the latest installed version of a language |
| [list_packages](list_packages) | List available and installed packages |
| [self_hosted_client](self_hosted_client) | Configure a client for a self-hosted instance, including a custom `*http.Client` |
| [official_api_client](official_api_client) | Configure a client for the official, key-gated Piston API |
| [interactive](interactive) | Stream output and write to stdin of a live process over WebSocket |
| [error_handling](error_handling) | Classify failures with `errors.Is` and inspect `*piston.APIError` |
