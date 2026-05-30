package agent

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"ibkr-stock-analysis/internal/domain"
	"ibkr-stock-analysis/priceaction"
)

const (
	defaultMaxKnowledgeSnippets    = 8
	defaultMaxKnowledgeSnippetChar = 1200
	defaultMaxKnowledgeTotalChar   = 7000
)

type KnowledgeSnippet struct {
	Source string `json:"source"`
	Title  string `json:"title"`
	Text   string `json:"text"`
}

type KnowledgeProvider interface {
	Snippets(input domain.AgentInput) ([]KnowledgeSnippet, error)
}

type MarkdownKnowledgeProvider struct {
	fsys            fs.FS
	MaxSnippets     int
	MaxSnippetChars int
	MaxTotalChars   int
}

func NewDefaultKnowledgeProvider() KnowledgeProvider {
	if override := strings.TrimSpace(os.Getenv("PRICEACTION_KB_PATH")); override != "" {
		return NewMarkdownKnowledgeProvider(os.DirFS(override))
	}
	return NewMarkdownKnowledgeProvider(priceaction.FS())
}

func NewMarkdownKnowledgeProvider(fsys fs.FS) *MarkdownKnowledgeProvider {
	return &MarkdownKnowledgeProvider{
		fsys:            fsys,
		MaxSnippets:     defaultMaxKnowledgeSnippets,
		MaxSnippetChars: defaultMaxKnowledgeSnippetChar,
		MaxTotalChars:   defaultMaxKnowledgeTotalChar,
	}
}

func (p *MarkdownKnowledgeProvider) Snippets(input domain.AgentInput) ([]KnowledgeSnippet, error) {
	if p == nil || p.fsys == nil {
		return nil, fmt.Errorf("PriceAction knowledge base unavailable: no filesystem configured")
	}
	entries, err := fs.ReadDir(p.fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("PriceAction knowledge base unavailable: %w", err)
	}

	var snippets []KnowledgeSnippet
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(path.Ext(entry.Name())) != ".md" {
			continue
		}
		data, err := fs.ReadFile(p.fsys, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("PriceAction knowledge base unavailable: read %s: %w", entry.Name(), err)
		}
		snippets = append(snippets, parseMarkdownKnowledgeSections(entry.Name(), string(data))...)
	}

	sort.SliceStable(snippets, func(i, j int) bool {
		left := scoreKnowledgeSnippet(snippets[i], input)
		right := scoreKnowledgeSnippet(snippets[j], input)
		if left != right {
			return left > right
		}
		if snippets[i].Source != snippets[j].Source {
			return snippets[i].Source < snippets[j].Source
		}
		return snippets[i].Title < snippets[j].Title
	})

	return limitKnowledgeSnippets(snippets, p.maxSnippets(), p.maxSnippetChars(), p.maxTotalChars()), nil
}

func knowledgeSnippetsJSON(snippets []KnowledgeSnippet) (string, error) {
	if len(snippets) == 0 {
		return "[]", nil
	}
	payload, err := json.MarshalIndent(snippets, "", "  ")
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func parseMarkdownKnowledgeSections(filename string, text string) []KnowledgeSnippet {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	source := "priceaction/" + filename
	defaultTitle := strings.TrimSuffix(filename, path.Ext(filename))
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	title := defaultTitle
	var body []string
	var snippets []KnowledgeSnippet

	flush := func() {
		trimmed := strings.TrimSpace(strings.Join(body, "\n"))
		if trimmed == "" {
			return
		}
		snippets = append(snippets, KnowledgeSnippet{
			Source: source,
			Title:  strings.TrimSpace(title),
			Text:   trimmed,
		})
	}

	for _, line := range lines {
		if heading, ok := markdownHeading(line); ok {
			flush()
			title = heading
			body = body[:0]
			continue
		}
		body = append(body, line)
	}
	flush()
	return snippets
}

func markdownHeading(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	hashes := 0
	for hashes < len(trimmed) && trimmed[hashes] == '#' {
		hashes++
	}
	if hashes == 0 || hashes > 6 || hashes >= len(trimmed) || trimmed[hashes] != ' ' {
		return "", false
	}
	title := strings.TrimSpace(trimmed[hashes+1:])
	title = strings.TrimRight(title, "#")
	title = strings.TrimSpace(title)
	return title, title != ""
}

func scoreKnowledgeSnippet(snippet KnowledgeSnippet, input domain.AgentInput) int {
	text := strings.ToLower(snippet.Source + "\n" + snippet.Title + "\n" + snippet.Text)
	score := countTerms(text, []string{
		"价格行为", "市场周期", "趋势", "突破", "交易区间", "盘整", "止损", "目标", "风险", "盈亏比",
	})

	rangeContext := strings.ToLower(input.Derived.LastCloseRelativeToRange)
	if strings.Contains(rangeContext, "upper") || strings.Contains(rangeContext, "high") || strings.Contains(rangeContext, "top") || strings.Contains(rangeContext, "上") {
		score += 2 * countTerms(text, []string{"顶部", "更低高点", "mtr", "趋势后期", "高潮反转", "阻力"})
	}
	if strings.Contains(rangeContext, "lower") || strings.Contains(rangeContext, "low") || strings.Contains(rangeContext, "bottom") || strings.Contains(rangeContext, "下") {
		score += 2 * countTerms(text, []string{"底部", "更高低点", "mtr", "趋势后期", "高潮反转", "支撑"})
	}

	volumeContext := strings.ToLower(input.Derived.VolumeContext)
	if strings.Contains(volumeContext, "high") || strings.Contains(volumeContext, "elevated") || strings.Contains(volumeContext, "strong") || strings.Contains(volumeContext, "高") || strings.Contains(volumeContext, "放量") {
		score += 2 * countTerms(text, []string{"高潮", "突破", "跟进"})
	}
	return score
}

func countTerms(text string, terms []string) int {
	score := 0
	for _, term := range terms {
		score += strings.Count(text, strings.ToLower(term))
	}
	return score
}

func limitKnowledgeSnippets(snippets []KnowledgeSnippet, maxSnippets int, maxSnippetChars int, maxTotalChars int) []KnowledgeSnippet {
	if maxSnippets <= 0 || maxSnippetChars <= 0 || maxTotalChars <= 0 {
		return nil
	}

	limited := make([]KnowledgeSnippet, 0, min(maxSnippets, len(snippets)))
	total := 0
	for _, snippet := range snippets {
		if len(limited) >= maxSnippets || total >= maxTotalChars {
			break
		}
		text := truncateRunes(strings.TrimSpace(snippet.Text), maxSnippetChars)
		remaining := maxTotalChars - total
		text = truncateRunes(text, remaining)
		if strings.TrimSpace(text) == "" {
			continue
		}
		snippet.Text = text
		limited = append(limited, snippet)
		total += len([]rune(text))
	}
	return limited
}

func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if limit <= 0 {
		return ""
	}
	if len(runes) <= limit {
		return text
	}
	return strings.TrimSpace(string(runes[:limit]))
}

func (p *MarkdownKnowledgeProvider) maxSnippets() int {
	if p.MaxSnippets > 0 {
		return p.MaxSnippets
	}
	return defaultMaxKnowledgeSnippets
}

func (p *MarkdownKnowledgeProvider) maxSnippetChars() int {
	if p.MaxSnippetChars > 0 {
		return p.MaxSnippetChars
	}
	return defaultMaxKnowledgeSnippetChar
}

func (p *MarkdownKnowledgeProvider) maxTotalChars() int {
	if p.MaxTotalChars > 0 {
		return p.MaxTotalChars
	}
	return defaultMaxKnowledgeTotalChar
}
