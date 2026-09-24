// The whole starter: read three environment variables, make one call, print
// what came back. Run it with `go run .`. Standard library only.
package main

import (
	"fmt"
	"os"
)

func main() {
	config, err := ReadConfig(os.Getenv)
	if err != nil {
		fail(err)
	}

	page, err := ListOperations(config)
	if err != nil {
		fail(err)
	}

	fmt.Printf("Operations (%s)\n", config.Environment)
	if len(page.Operations) == 0 {
		fmt.Println("  none yet; actions you run with this key show up here")
	}
	for _, operation := range page.Operations {
		fmt.Printf("  %-9s %s  %s\n", operation.Status, operation.Name, operation.OperationID)
	}
	if page.HasMore {
		fmt.Println("  and more on the next page")
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
