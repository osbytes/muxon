package main

import (
	"bytes"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	versionCmd.SetOut(buf)
	versionCmd.Run(versionCmd, nil)
	if buf.String() != version+"\n" {
		t.Fatalf("got %q want %q", buf.String(), version+"\n")
	}
}
