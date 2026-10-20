package keywordmatch

import (
	"strings"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/keywordmatch/models"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
)

// execute runs the action configured on the matched rule.
func (m *Manager) execute(ctx conversationCtx, match models.Match) error {
	content := strings.TrimSpace(match.ReplyContent)

	switch match.Rule.Action {
	case models.ActionAgentAndVisitor:
		// Send the reply into the conversation with `silent = false` so it is
		// broadcast to the agents and delivered to the visitor's widget.
		if content == "" {
			return nil
		}
		metaMap := map[string]any{
			"type":    cmodels.MessageMetaTypeBotKeywordMatch,
			"rule_id": match.Rule.ID,
		}
		if _, err := m.conv.SendAutoReply(ctx.inboxID, ctx.contactID, ctx.conversationUUID, content, metaMap, false); err != nil {
			return err
		}
		return nil

	default:
		// Agent-only alert: a system prompt on the agent UI. Nothing is written
		// to the conversation, so it never shows up in the transcript or in the
		// unread counters.
		m.notifyAgents(ctx, match)
		return nil
	}
}

// notifyAgents pushes the keyword match alert to the agents allowed to read the
// conversation. When no agent is authorized (or online) the alert goes to every
// online agent so the hint is never lost.
func (m *Manager) notifyAgents(ctx conversationCtx, match models.Match) {
	userIDs := m.conv.AuthorizedConnectedAgentIDs(ctx.assignedUserID, ctx.assignedTeamID)
	data := map[string]any{
		"conversation_uuid": ctx.conversationUUID,
		"rule_id":           match.Rule.ID,
		"rule_name":         match.Rule.Name,
		"matched_text":      match.MatchedText,
		"content":           strings.TrimSpace(match.ReplyContent),
		"created_at":        time.Now().Format(time.RFC3339),
	}
	m.conv.BroadcastToAgents(userIDs, wsmodels.MessageTypeKeywordMatchAlert, data)
}
