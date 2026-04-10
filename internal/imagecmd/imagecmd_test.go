package imagecmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	var out bytes.Buffer
	if err := Run(nil, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "convert") {
		t.Fatalf("help = %q", out.String())
	}
}

func TestRunConvertUsage(t *testing.T) {
	var out bytes.Buffer
	err := Run([]string{"convert", "only-input.jpg"}, &out)
	if err == nil {
		t.Fatal("expected usage error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Fatalf("error = %v", err)
	}
}
