package agent

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"ibkr-stock-analysis/internal/domain"
)

func TestMarkdownKnowledgeProviderParsesSectionsAndSkipsEmptyFiles(t *testing.T) {
	provider := NewMarkdownKnowledgeProvider(fstest.MapFS{
		"突破 4.md": &fstest.MapFile{Data: []byte("")},
		"趋势 1.md": &fstest.MapFile{Data: []byte("### 核心观点\n牛市趋势关注更高的低点。\n\n### 止损\n止损放在主要低点下方。\n")},
	})

	snippets, err := provider.Snippets(domain.AgentInput{
		Derived: domain.DerivedFeatures{LastCloseRelativeToRange: "middle"},
	})
	if err != nil {
		t.Fatalf("Snippets returned error: %v", err)
	}

	if len(snippets) != 2 {
		t.Fatalf("len(snippets) = %d, want 2", len(snippets))
	}
	for _, snippet := range snippets {
		if strings.Contains(snippet.Source, "突破 4.md") {
			t.Fatalf("empty file was included: %+v", snippet)
		}
	}
	assertKnowledgeContains(t, snippets, "核心观点", "更高的低点")
	assertKnowledgeContains(t, snippets, "止损", "主要低点")
}

func TestMarkdownKnowledgeProviderSelectsTopReversalContextForUpperRange(t *testing.T) {
	provider := NewMarkdownKnowledgeProvider(fstest.MapFS{
		"顶部.md": &fstest.MapFile{Data: []byte("### 顶部 MTR\n顶部反转关注更低高点、阻力和高潮反转。")},
		"底部.md": &fstest.MapFile{Data: []byte("### 底部 MTR\n底部反转关注更高低点和支撑。")},
	})

	snippets, err := provider.Snippets(domain.AgentInput{
		Derived: domain.DerivedFeatures{
			LastCloseRelativeToRange: "upper",
			VolumeContext:            "elevated",
		},
	})
	if err != nil {
		t.Fatalf("Snippets returned error: %v", err)
	}

	if len(snippets) == 0 {
		t.Fatal("expected snippets")
	}
	if !strings.Contains(snippets[0].Text, "顶部反转") {
		t.Fatalf("top snippet = %+v, want top reversal context first", snippets[0])
	}
}

func TestMarkdownKnowledgeProviderSelectsBottomReversalContextForLowerRange(t *testing.T) {
	provider := NewMarkdownKnowledgeProvider(fstest.MapFS{
		"顶部.md": &fstest.MapFile{Data: []byte("### 顶部 MTR\n顶部反转关注更低高点、阻力和高潮反转。")},
		"底部.md": &fstest.MapFile{Data: []byte("### 底部 MTR\n底部反转关注更高低点和支撑。")},
	})

	snippets, err := provider.Snippets(domain.AgentInput{
		Derived: domain.DerivedFeatures{
			LastCloseRelativeToRange: "lower",
			VolumeContext:            "normal",
		},
	})
	if err != nil {
		t.Fatalf("Snippets returned error: %v", err)
	}

	if len(snippets) == 0 {
		t.Fatal("expected snippets")
	}
	if !strings.Contains(snippets[0].Text, "底部反转") {
		t.Fatalf("top snippet = %+v, want bottom reversal context first", snippets[0])
	}
}

func TestMarkdownKnowledgeProviderEnforcesLimitsDeterministically(t *testing.T) {
	files := fstest.MapFS{}
	for _, name := range []string{"b.md", "a.md", "c.md"} {
		files[name] = &fstest.MapFile{Data: []byte("### 趋势 突破 风险\n" + strings.Repeat("趋势突破风险", 100))}
	}
	provider := NewMarkdownKnowledgeProvider(files)
	provider.MaxSnippets = 2
	provider.MaxSnippetChars = 20
	provider.MaxTotalChars = 40

	first, err := provider.Snippets(domain.AgentInput{})
	if err != nil {
		t.Fatalf("Snippets returned error: %v", err)
	}
	second, err := provider.Snippets(domain.AgentInput{})
	if err != nil {
		t.Fatalf("Snippets returned error: %v", err)
	}

	if len(first) != 2 {
		t.Fatalf("len(first) = %d, want 2", len(first))
	}
	if first[0] != second[0] || first[1] != second[1] {
		t.Fatalf("snippets are not deterministic:\nfirst=%+v\nsecond=%+v", first, second)
	}
	total := 0
	for _, snippet := range first {
		if len([]rune(snippet.Text)) > 20 {
			t.Fatalf("snippet text length = %d, want <= 20", len([]rune(snippet.Text)))
		}
		total += len([]rune(snippet.Text))
	}
	if total > 40 {
		t.Fatalf("total chars = %d, want <= 40", total)
	}
	if first[0].Source != "priceaction/a.md" {
		t.Fatalf("first source = %q, want stable lexical tiebreak", first[0].Source)
	}
}

func TestDefaultKnowledgeProviderLoadsOverridePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "custom.md"), []byte("### 自定义策略\n只使用外部知识库。"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PRICEACTION_KB_PATH", dir)

	provider := NewDefaultKnowledgeProvider()
	snippets, err := provider.Snippets(domain.AgentInput{})
	if err != nil {
		t.Fatalf("Snippets returned error: %v", err)
	}

	assertKnowledgeContains(t, snippets, "自定义策略", "外部知识库")
}

func TestDefaultKnowledgeProviderUsesEmbeddedPriceActionFiles(t *testing.T) {
	t.Setenv("PRICEACTION_KB_PATH", "")

	provider := NewDefaultKnowledgeProvider()
	snippets, err := provider.Snippets(domain.AgentInput{})
	if err != nil {
		t.Fatalf("Snippets returned error: %v", err)
	}

	if len(snippets) == 0 {
		t.Fatal("expected embedded snippets")
	}
	if _, err := fs.Stat(providerFS(t, provider), "突破 4.md"); err != nil {
		t.Fatalf("embedded fs missing expected knowledge file: %v", err)
	}
}

func assertKnowledgeContains(t *testing.T, snippets []KnowledgeSnippet, title string, text string) {
	t.Helper()
	for _, snippet := range snippets {
		if strings.Contains(snippet.Title, title) && strings.Contains(snippet.Text, text) {
			return
		}
	}
	t.Fatalf("snippets do not contain title %q with text %q: %+v", title, text, snippets)
}

func providerFS(t *testing.T, provider KnowledgeProvider) fs.FS {
	t.Helper()
	markdown, ok := provider.(*MarkdownKnowledgeProvider)
	if !ok {
		t.Fatalf("provider type = %T, want *MarkdownKnowledgeProvider", provider)
	}
	return markdown.fsys
}
