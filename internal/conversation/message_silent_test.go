package conversation

import (
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
)

func TestConversationMetaFlag(t *testing.T) {
	cases := []struct {
		name string
		meta json.RawMessage
		key  string
		want bool
	}{
		{"nil meta", nil, models.ConversationMetaBotHumanRequested, false},
		{"empty meta", json.RawMessage(``), models.ConversationMetaBotHumanRequested, false},
		{"empty object", json.RawMessage(`{}`), models.ConversationMetaBotHumanRequested, false},
		{"flag true", json.RawMessage(`{"bot_human_requested":true}`), models.ConversationMetaBotHumanRequested, true},
		{"flag false", json.RawMessage(`{"bot_human_requested":false}`), models.ConversationMetaBotHumanRequested, false},
		{
			"flag set to a non-boolean is ignored",
			json.RawMessage(`{"bot_human_requested":"yes"}`),
			models.ConversationMetaBotHumanRequested,
			false,
		},
		{
			"survives other keys being present",
			json.RawMessage(`{"queue_info":{"count":3},"bot_human_requested":true}`),
			models.ConversationMetaBotHumanRequested,
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := conversationMetaFlag(c.meta, c.key)
			if err != nil {
				t.Fatalf("conversationMetaFlag: unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("conversationMetaFlag() = %v, want %v", got, c.want)
			}
		})
	}
}

// TestConversationMetaFlagMalformed covers the case where meta is valid JSON
// but not an object: the caller must not blow up, it should simply treat the
// flag as absent.
func TestConversationMetaFlagMalformed(t *testing.T) {
	if _, err := conversationMetaFlag(json.RawMessage(`["not","an","object"]`), models.ConversationMetaBotHumanRequested); err == nil {
		t.Errorf("expected an error for malformed meta, got nil")
	}
}
