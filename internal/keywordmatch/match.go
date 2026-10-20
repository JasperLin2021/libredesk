package keywordmatch

import (
	"strings"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/keywordmatch/models"
	"github.com/volatiletech/null/v9"
)

// conversationCtx carries everything needed to execute a matched rule.
type conversationCtx struct {
	inboxID         int
	contactID       int
	conversationUUID string
	assignedUserID  null.Int
	assignedTeamID  null.Int
}

// HandleVisitorMessage runs the visitor-source rules against an incoming
// visitor message and executes the actions of every matching rule.
func (m *Manager) HandleVisitorMessage(conversation cmodels.Conversation, content string) {
	m.handle(models.SourceVisitor, content, conversationCtx{
		inboxID:          conversation.InboxID,
		contactID:        conversation.ContactID,
		conversationUUID: conversation.UUID,
		assignedUserID:   conversation.AssignedUserID,
		assignedTeamID:   conversation.AssignedTeamID,
	})
}

// HandleAgentMessage runs the agent-source rules against an outgoing agent
// message and executes the actions of every matching rule.
func (m *Manager) HandleAgentMessage(inboxID, contactID int, conversationUUID, content string, assignedUserID, assignedTeamID null.Int) {
	m.handle(models.SourceAgent, content, conversationCtx{
		inboxID:          inboxID,
		contactID:        contactID,
		conversationUUID: conversationUUID,
		assignedUserID:   assignedUserID,
		assignedTeamID:   assignedTeamID,
	})
}

// handle matches the content against the rules of the given source and executes
// each match. A failure is logged and never propagated: automatic replies must
// not block the message that triggered them.
func (m *Manager) handle(source, content string, ctx conversationCtx) {
	if m == nil || m.conv == nil {
		return
	}
	matches := m.Match(source, content)
	for _, match := range matches {
		if err := m.execute(ctx, match); err != nil {
			m.lo.Error("error executing keyword match rule",
				"rule_id", match.Rule.ID, "conversation_uuid", ctx.conversationUUID, "error", err)
		}
	}
}

// Match returns the rules of the given source that match the content, in rule
// order. When a matching rule has `search_first` set, matching stops there and
// only that match is returned.
func (m *Manager) Match(source, content string) []models.Match {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	rules := m.enabledRules()
	matches := make([]models.Match, 0, 1)
	for _, rule := range rules {
		if rule.Source != source {
			continue
		}
		matched, ok := m.matchRule(rule, content)
		if !ok {
			continue
		}
		matches = append(matches, models.Match{
			Rule:         rule,
			MatchedText:  matched,
			ReplyContent: strings.ReplaceAll(rule.ReplyContent, models.MatchPlaceholder, matched),
		})
		// "搜索第一项": stop at the first match and ignore the remaining rules.
		if rule.SearchFirst {
			break
		}
	}
	return matches
}

// matchRule tests a single rule against the content and returns the text that
// matched along with whether it matched at all.
func (m *Manager) matchRule(rule models.KeywordMatchRule, content string) (string, bool) {
	if rule.MatchType == models.MatchTypeRegex {
		re, err := compileRegex(rule.Pattern, rule.CaseSensitive)
		if err != nil {
			m.lo.Error("error compiling keyword match regex", "rule_id", rule.ID, "error", err)
			return "", false
		}
		matched := re.FindString(content)
		if matched == "" {
			return "", false
		}
		return matched, true
	}

	// Keyword match: the pattern holds one or more keywords separated by
	// newlines, commas or Chinese commas. Any keyword contained in the message
	// triggers the rule.
	haystack := content
	for _, keyword := range splitKeywords(rule.Pattern) {
		needle := keyword
		if !rule.CaseSensitive {
			haystack = strings.ToLower(content)
			needle = strings.ToLower(keyword)
		}
		if strings.Contains(haystack, needle) {
			return keyword, true
		}
	}
	return "", false
}

// splitKeywords splits a keyword pattern into individual keywords.
func splitKeywords(pattern string) []string {
	splitter := func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == '，' || r == ';' || r == '；'
	}
	keywords := make([]string, 0)
	for _, part := range strings.FieldsFunc(pattern, splitter) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		keywords = append(keywords, part)
	}
	return keywords
}
