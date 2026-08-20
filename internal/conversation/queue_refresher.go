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
