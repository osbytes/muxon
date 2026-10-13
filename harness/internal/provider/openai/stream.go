package openai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

type sseStream struct {
	body   io.ReadCloser
	reader *bufio.Reader
	done   bool
	acc    provider.Response
}

func newSSEStream(body io.ReadCloser) *sseStream {
	return &sseStream{body: body, reader: bufio.NewReader(body)}
}

func (s *sseStream) Next() (provider.Response, error) {
	if s.done {
		return provider.Response{}, fmt.Errorf("stream closed")
	}
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				s.done = true
				return s.acc, nil
			}
			return provider.Response{}, err
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			s.done = true
			return s.acc, nil
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string         `json:"content"`
					ToolCalls []toolCallWire `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return provider.Response{}, err
		}
		if len(chunk.Choices) > 0 {
			d := chunk.Choices[0].Delta
			s.acc.Text += d.Content
			for _, tc := range d.ToolCalls {
				s.acc.ToolCalls = append(s.acc.ToolCalls, provider.ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: json.RawMessage(tc.Function.Arguments),
				})
			}
			if chunk.Choices[0].FinishReason != "" {
				s.acc.FinishReason = chunk.Choices[0].FinishReason
			}
		}
		if chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 {
			s.acc.Usage = provider.Usage{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
			}
		}
		if s.acc.FinishReason != "" {
			s.done = true
			return s.acc, nil
		}
	}
}

func (s *sseStream) Close() error {
	s.done = true
	return s.body.Close()
}
