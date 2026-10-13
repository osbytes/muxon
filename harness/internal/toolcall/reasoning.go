package toolcall

import "strings"

const thinkOpen = "<think>"
const thinkClose = "</think>"

// StripReasoning removes thinking tags from model text before parsing.
func StripReasoning(text string) string {
	for {
		i := strings.Index(text, thinkOpen)
		if i < 0 {
			if strings.Contains(text, thinkOpen) {
				return strings.TrimSpace(strings.Split(text, thinkOpen)[0])
			}
			return strings.TrimSpace(text)
		}
		before := text[:i]
		rest := text[i+len(thinkOpen):]
		j := strings.Index(rest, thinkClose)
		if j < 0 {
			return strings.TrimSpace(before)
		}
		text = before + rest[j+len(thinkClose):]
	}
}
