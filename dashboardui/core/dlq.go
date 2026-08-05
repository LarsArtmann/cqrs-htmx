package core

import (
	"context"
)

// DLQProjectionLink holds a projection name and its dead-letter count for
// rendering on the DLQ index page.
type DLQProjectionLink struct {
	Name  string
	Count int
}

// DLQProjectionLinks returns one entry per registered projection with its
// dead-letter count. When a DeadLetterStore is configured, the count is
// exact (via List). Otherwise the projection's error counter is used as
// a fallback.
func DLQProjectionLinks(ctx context.Context, cfg Config) []DLQProjectionLink {
	if cfg.ProjectionHost == nil {
		return nil
	}

	var links []DLQProjectionLink

	for _, ws := range cfg.ProjectionHost.Status() {
		count := 0

		if cfg.DeadLetterStore != nil {
			entries, err := cfg.DeadLetterStore.List(ctx, ws.Name)
			if err == nil {
				count = len(entries)
			}
		} else {
			count = int(ws.Errors)
		}

		links = append(links, DLQProjectionLink{Name: ws.Name, Count: count})
	}

	return links
}
