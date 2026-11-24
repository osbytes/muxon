package scripted

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

// Provider returns queued responses in order.
type Provider struct {
	mu    sync.Mutex
	queue []provider.Response
	idx   int
}

// New returns a provider that drains responses in order.
func New(responses []provider.Response) *Provider {
	return &Provider{queue: responses}
}

// FromEnv loads responses from THEE_SCRIPTED_PROVIDER_FILE (JSON array).
func FromEnv() (*Provider, error) {
	path := os.Getenv("THEE_SCRIPTED_PROVIDER_FILE")
	if path == "" {
		return nil, fmt.Errorf("THEE_SCRIPTED_PROVIDER_FILE not set")
	}
	data, err := os.ReadFile(path) //nolint:gosec // G703: test-only provider; path is set by the developer running the harness
	if err != nil {
		return nil, err
	}
	var responses []provider.Response
	if err := json.Unmarshal(data, &responses); err != nil {
		return nil, err
	}
	return New(responses), nil
}

func (p *Provider) next() (provider.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.idx >= len(p.queue) {
		return provider.Response{}, fmt.Errorf("scripted provider: queue exhausted at %d", p.idx)
	}
	r := p.queue[p.idx]
	p.idx++
	return r, nil
}

// Complete implements provider.Provider.
func (p *Provider) Complete(ctx context.Context, req provider.Request) (provider.Response, error) {
	if err := ctx.Err(); err != nil {
		return provider.Response{}, err
	}
	return p.next()
}

// Stream implements provider.Provider by returning the full response as one chunk.
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
		return provider.Response{}, fmt.Errorf("stream closed")
	}
	o.done = true
	return o.resp, nil
}

func (o *oneShot) Close() error {
	o.done = true
	return nil
}
