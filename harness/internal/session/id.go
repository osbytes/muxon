package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewID returns a session id (YYYYMMDDTHHMMSSZ-xxxxxx).
func NewID() string {
	now := time.Now().UTC().Format("20060102T%H%M%SZ")
	var b [3]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s-%s", now, hex.EncodeToString(b[:]))
}
