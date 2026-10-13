package observability

import (
	"bufio"
	"os"

	"github.com/PLACEHOLDER/thee/harness/internal/security"
)

// ExportRedacted copies a trace, redacting secret-like strings.
func ExportRedacted(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	sc := bufio.NewScanner(in)
	for sc.Scan() {
		line := security.RedactSecrets(sc.Text())
		if _, err := out.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return sc.Err()
}
