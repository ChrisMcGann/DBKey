// DBKey - Spectral library conversion tool
package main

import (
	"fmt"
	"os"

	"github.com/ChrisMcGann/DBKey/cmd/dbkey/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
