# Go-Piston!

This is a Go wrapper for working with the [Piston](https://github.com/engineer-man/piston) API.

It supports both the endpoints, namely `runtimes` and `execute`, mentioned [here](https://github.com/engineer-man/piston#public-api).


## 💻 Installation
To install the library simply open a terminal and type:
```
go get github.com/milindmadhukar/go-piston
```

## ️️🛠️ Tools Used

This project was written purely in `Golang` for `Golang`.</br>
The module helps with the usage of the [Piston API](https://github.com/engineer-man/piston#public-api).

## 🏁 Basic Setup:

```go
package main

import (
	"fmt"
	"net/http"

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.New("", http.DefaultClient)
	output, err := client.Execute("python", "", // Passing language. Since no version is specified, it uses th elates supported version.
		&[]piston.Code{{Content: "inp = input()\nprint(inp[::-1])"}}, // Passing Code.
		piston.OptionalParams{Stdin: "hello world"})                  // Passing input as "hello world".
	if err != nil {
		panic(err)
	}
	fmt.Println(output.GetOutput())
}
```

### Output
```
dlrow olleh
```


## 🧿 Extras

If you face any difficulties contact me [here.](https://milindm.me/contact/)

Thats it, have fun ✚✖
