package replay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
)

// Step is one recorded model turn.
type Step struct {
	Hash     string            `json:"hash"`
	Texts    []string          `json:"texts,omitempty"`
	Response provider.Response `json:"response"`
}

// Provider serves recorded responses in order.
type Provider struct {
	steps []Step
	idx   int
}

// New returns a replay provider.
func New(steps []Step) *Provider {
	return &Provider{steps: steps}
}

// Load reads a replay file.
func Load(path string) (*Provider, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var steps []Step
	if err := json.Unmarshal(b, &steps); err != nil {
		return nil, err
	}
	return New(steps), nil
}

// HashMessages hashes assembled chat messages.
func HashMessages(msgs []provider.Message) string {
	h := sha256.New()
	for _, m := range msgs {
		h.Write([]byte(m.Role))
		h.Write([]byte{0})
		h.Write([]byte(m.Content))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func messageTexts(msgs []provider.Message) string {
	var b strings.Builder
	for _, m := range msgs {
		b.WriteString(m.Role)
		b.WriteString(": ")
		b.WriteString(m.Content)
		b.WriteByte('\n')
	}
	return b.String()
}

// Complete implements provider.Provider.
func (p *Provider) Complete(ctx context.Context, req provider.Request) (provider.Response, error) {
	if err := ctx.Err(); err != nil {
		return provider.Response{}, err
	}
	if p.idx >= len(p.steps) {
		return provider.Response{}, fmt.Errorf("replay: sequence exhausted at %d", p.idx)
	}
	step := p.steps[p.idx]
	p.idx++
	got := HashMessages(req.Messages)
	if step.Hash != "" && step.Hash != got {
		return provider.Response{}, fmt.Errorf("replay hash mismatch\nwant:\n%s\ngot:\n%s", strings.Join(step.Texts, "\n"), messageTexts(req.Messages))
	}
	return step.Response, nil
}

// Stream implements provider.Provider.
func (p *Provider) Stream(ctx context.Context, req provider.Request) (provider.Stream, error) {
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	return &oneShot{resp: resp}, nil
}

type oneShot struct {
	resp provider.Response
	done bool
}

func (o *oneShot) Next() (provider.Response, error) {
	if o.done {
		return provider.Response{}, fmt.Errorf("EOF")
	}
	o.done = true
	return o.resp, nil
}

func (o *oneShot) Close() error { return nil }

// WriteFile stores replay steps.
func WriteFile(path string, steps []Step) error {
	b, err := json.MarshalIndent(steps, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
