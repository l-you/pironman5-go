package imagecmd

import (
	"fmt"
	"io"

	"github.com/l-you/pironman5-go/internal/imageconv"
)

func Run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}
	switch args[0] {
	case "-h", "--help", "help":
		printUsage(stdout)
		return nil
	case "convert":
		return runConvert(args[1:], stdout)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runConvert(args []string, stdout io.Writer) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: pironman5-image convert <input.{jpg|png}> <output.pbm>")
	}
	if err := imageconv.ConvertToPBM(args[0], args[1]); err != nil {
		return err
	}
	_, err := fmt.Fprintf(stdout, "wrote %s\n", args[1])
	return err
}

func printUsage(stdout io.Writer) {
	fmt.Fprint(stdout, `pironman5-image

Commands:
  convert <input.{jpg|png}> <output.pbm>
`)
}
