package conversation

import (
	"context"
	"time"
)

// RunQueueInfoRefresher periodically recomputes the queue count for every
// conversation waiting for a human agent and pushes it to their widget clients.
// The interval is configured via conversation.queue_info_refresh_interval.
func (c *Manager) RunQueueInfoRefresher(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.RefreshWaitingQueueInfo(); err != nil {
				c.lo.Error("error running queue info refresher", "error", err)
			}
		}
	}
}

// RunNoReplyTimeoutRefresher periodically scans for open conversations that are
// assigned to an agent and have not received a visitor reply within the
// configured no-reply timeout. For each such conversation it sends the
// configured "no reply timeout" message on behalf of the assigned agent and
// then closes the conversation. The timeout is configured via
// conversation.no_reply_timeout and the scan interval via
// conversation.no_reply_timeout_scan_interval.
func (c *Manager) RunNoReplyTimeoutRefresher(ctx context.Context, timeout, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkNoReplyTimeouts(timeout)
		}
	}
}

// checkNoReplyTimeouts finds all stale assigned conversations and routes them
// through the quick reply handler (send timeout message + close conversation).
func (c *Manager) checkNoReplyTimeouts(timeout time.Duration) {
	if c.quickReply == nil {
		return
	}
	var stale []staleAgentConversation
	// Pass the timeout as an interval string (e.g. "5m0s") rather than a
	// time.Duration, which lib/pq would encode as an integer number of
	// nanoseconds and PostgreSQL would then interpret as seconds, blowing past
	// the valid timestamp range.
	if err := c.q.GetStaleAgentConversations.Select(&stale, timeout.String()); err != nil {
		c.lo.Error("error fetching stale agent conversations", "error", err)
		return
	}
	for _, s := range stale {
		conversation, err := c.GetConversation(0, s.UUID, "")
		if err != nil {
			c.lo.Error("error fetching conversation for no reply timeout", "conversation_uuid", s.UUID, "error", err)
			continue
		}
		if err := c.quickReply.HandleNoReplyTimeout(conversation); err != nil {
			c.lo.Error("error handling no reply timeout", "conversation_uuid", s.UUID, "error", err)
		}
	}
}
