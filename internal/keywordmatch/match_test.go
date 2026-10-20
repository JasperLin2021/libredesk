package keywordmatch

import (
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/keywordmatch/models"
	"github.com/zerodha/logf"
)

// newTestManager builds a manager with the given rules preloaded into the cache
// so matching can be exercised without a database.
func newTestManager(rules []models.KeywordMatchRule) *Manager {
	lo := logf.New(logf.Opts{})
	return &Manager{
		lo:       &lo,
		rules:    rules,
		cachedAt: time.Now(),
	}
}

func rule(id int, source, matchType, pattern, action, reply string, caseSensitive, searchFirst bool) models.KeywordMatchRule {
	r := models.KeywordMatchRule{
		ID:            id,
		Name:          "rule",
		Source:        source,
		MatchType:     matchType,
		Pattern:       pattern,
		Action:        action,
		ReplyContent:  reply,
		Enabled:       true,
		CaseSensitive: caseSensitive,
		SearchFirst:   searchFirst,
	}
	return r
}

func TestMatchKeyword(t *testing.T) {
	m := newTestManager([]models.KeywordMatchRule{
		rule(1, models.SourceVisitor, models.MatchTypeKeyword, "退款,退货", models.ActionAgentNotify, "命中 ##", false, false),
	})

	matches := m.Match(models.SourceVisitor, "我要申请退款")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].MatchedText != "退款" {
		t.Errorf("expected matched keyword 退款, got %q", matches[0].MatchedText)
	}
	if matches[0].ReplyContent != "命中 退款" {
		t.Errorf("expected ## to be replaced, got %q", matches[0].ReplyContent)
	}

	// Case insensitive by default.
	if got := m.Match(models.SourceVisitor, "REFUND 退款"); len(got) != 1 {
		t.Errorf("expected case-insensitive match, got %d", len(got))
	}
	// Wrong source is ignored.
	if got := m.Match(models.SourceAgent, "我要退款"); len(got) != 0 {
		t.Errorf("expected no match for the agent source, got %d", len(got))
	}
	// No keyword present.
	if got := m.Match(models.SourceVisitor, "你好"); len(got) != 0 {
		t.Errorf("expected no match, got %d", len(got))
	}
}

func TestMatchKeywordCaseSensitive(t *testing.T) {
	m := newTestManager([]models.KeywordMatchRule{
		rule(1, models.SourceVisitor, models.MatchTypeKeyword, "Refund", models.ActionAgentNotify, "##", true, false),
	})
	if got := m.Match(models.SourceVisitor, "refund"); len(got) != 0 {
		t.Errorf("expected no match when case sensitive, got %d", len(got))
	}
	if got := m.Match(models.SourceVisitor, "Refund"); len(got) != 1 {
		t.Errorf("expected a match, got %d", len(got))
	}
}

func TestMatchRegex(t *testing.T) {
	m := newTestManager([]models.KeywordMatchRule{
		rule(1, models.SourceVisitor, models.MatchTypeRegex, `(退款|退货)`, models.ActionAgentAndVisitor, "已收到：##", false, false),
	})
	matches := m.Match(models.SourceVisitor, "我想退货可以吗")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].MatchedText != "退货" {
		t.Errorf("expected matched text 退货, got %q", matches[0].MatchedText)
	}
	if matches[0].ReplyContent != "已收到：退货" {
		t.Errorf("unexpected reply content %q", matches[0].ReplyContent)
	}
}

func TestMatchSearchFirst(t *testing.T) {
	rules := []models.KeywordMatchRule{
		rule(1, models.SourceVisitor, models.MatchTypeKeyword, "退款", models.ActionAgentNotify, "first", false, true),
		rule(2, models.SourceVisitor, models.MatchTypeKeyword, "退款", models.ActionAgentNotify, "second", false, false),
	}
	m := newTestManager(rules)
	matches := m.Match(models.SourceVisitor, "我要退款")
	if len(matches) != 1 {
		t.Fatalf("expected search first to stop after 1 match, got %d", len(matches))
	}
	if matches[0].Rule.ID != 1 {
		t.Errorf("expected the first rule, got %d", matches[0].Rule.ID)
	}

	// Without search_first every matching rule is returned.
	m = newTestManager([]models.KeywordMatchRule{
		rule(1, models.SourceVisitor, models.MatchTypeKeyword, "退款", models.ActionAgentNotify, "first", false, false),
		rule(2, models.SourceVisitor, models.MatchTypeKeyword, "退款", models.ActionAgentNotify, "second", false, false),
	})
	if got := m.Match(models.SourceVisitor, "我要退款"); len(got) != 2 {
		t.Errorf("expected 2 matches, got %d", len(got))
	}
}

func TestMatchEmptyContent(t *testing.T) {
	m := newTestManager([]models.KeywordMatchRule{
		rule(1, models.SourceVisitor, models.MatchTypeKeyword, "", models.ActionAgentNotify, "", false, false),
	})
	if got := m.Match(models.SourceVisitor, "   "); len(got) != 0 {
		t.Errorf("expected no match for empty content, got %d", len(got))
	}
}
