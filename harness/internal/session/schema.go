package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MajorSchema reads schema_version from the first session.start line.
func MajorSchema(tracePath string) (int, error) {
	f, err := os.Open(tracePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var ev struct {
			Type          string `json:"type"`
			SchemaVersion string `json:"schema_version"`
		}
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Type != "session.start" || ev.SchemaVersion == "" {
			continue
		}
		major, err := strconv.Atoi(strings.Split(ev.SchemaVersion, ".")[0])
		if err != nil {
			return 0, fmt.Errorf("schema_version %q", ev.SchemaVersion)
		}
		return major, nil
	}
	return 1, sc.Err()
}
