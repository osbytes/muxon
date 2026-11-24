// Command muxon is the muxon harness CLI entrypoint.
package main

import (
	"os"
)

func main() {
	if err := execute(); err != nil {
		if ec, ok := err.(exitError); ok {
			os.Exit(ec.code)
		}
		os.Exit(1)
	}
}
