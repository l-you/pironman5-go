package main

import (
	"fmt"
	"os"

	"github.com/l-you/pironman5-go/internal/imagecmd"
)

func main() {
	if err := imagecmd.Run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
