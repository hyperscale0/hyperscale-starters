// The whole starter: read three environment variables, make one call, print
// what came back. Run it with `go run .`. Standard library only.
package main

import (
	"fmt"
	"os"
)

// preview is enough to see the shape without scrolling; the rest is a count.
const preview = 10

func main() {
	config, err := ReadConfig(os.Getenv)
	if err != nil {
		fail(err)
	}

	product, err := FetchProductDescriptor(config)
	if err != nil {
		fail(err)
	}

	fmt.Printf("%s (%s)\n", product.Title, config.Environment)
	fmt.Printf("%d operations\n", len(product.Operations))
	for index, operation := range product.Operations {
		if index == preview {
			fmt.Printf("  and %d more\n", len(product.Operations)-preview)
			break
		}
		fmt.Printf("  %-6s %s  %s\n", operation.Method, operation.Path, operation.OperationID)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
