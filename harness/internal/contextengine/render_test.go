package contextengine

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

func TestCompactSystemGoldenHash(t *testing.T) {
	specs := []tools.Spec{
		{Name: "read", Description: "Read file lines"},
		{Name: "patch", Description: "Apply patch"},
	}
	r, err := RenderVariant(VariantCompact, specs)
	if err != nil {
		t.Fatal(err)
	}
	block := SystemBlock(r)
	golden := filepath.Join("..", "..", "testdata", "prompts", "compact_system.golden")
	if os.Getenv("UPDATE_PROMPT_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(block), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("missing golden %s (run UPDATE_PROMPT_GOLDEN=1): %v", golden, err)
	}
	if block != string(want) {
		sum := sha256.Sum256([]byte(block))
		t.Fatalf("compact system changed hash=%s", hex.EncodeToString(sum[:8]))
	}
}

func TestStandardSystemGoldenHash(t *testing.T) {
	specs := []tools.Spec{
		{Name: "read", Description: "Read file lines"},
		{Name: "patch", Description: "Apply patch"},
	}
	r, err := RenderVariant(VariantStandard, specs)
	if err != nil {
		t.Fatal(err)
	}
	block := SystemBlock(r)
	golden := filepath.Join("..", "..", "testdata", "prompts", "standard_system.golden")
	if os.Getenv("UPDATE_PROMPT_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(block), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("missing golden %s (run UPDATE_PROMPT_GOLDEN=1): %v", golden, err)
	}
	if block != string(want) {
		sum := sha256.Sum256([]byte(block))
		t.Fatalf("standard system changed hash=%s", hex.EncodeToString(sum[:8]))
	}
}

func TestRulesAllowTextReply(t *testing.T) {
	for _, variant := range []string{VariantCompact, VariantStandard} {
		t.Run(variant, func(t *testing.T) {
			r, err := RenderVariant(variant, nil)
			if err != nil {
				t.Fatalf("RenderVariant(%q) error = %v", variant, err)
			}
			for _, banned := range []string{"nothing else", "not chat"} {
				if strings.Contains(r.SystemRules, banned) {
					t.Errorf("RenderVariant(%q) rules contain %q", variant, banned)
				}
			}
		})
	}
}

func TestStandardRendersWithoutExample(t *testing.T) {
	r, err := RenderVariant(VariantStandard, []tools.Spec{{Name: "read", Description: "read"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Example != "" {
		t.Fatal("standard should not include example")
	}
	if r.SystemRules == "" || r.ToolCatalog == "" {
		t.Fatal("empty render")
	}
}
