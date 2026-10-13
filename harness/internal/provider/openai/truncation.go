package openai

// TruncatedByServer reports whether usage suggests server-side prompt truncation.
func TruncatedByServer(estimatedPromptTokens, reportedPromptTokens int) bool {
	if estimatedPromptTokens <= 0 || reportedPromptTokens <= 0 {
		return false
	}
	return reportedPromptTokens < estimatedPromptTokens/2
}
