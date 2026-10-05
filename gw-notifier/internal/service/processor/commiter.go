package processor

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

func (p *Processor) addToCommit(msg kafka.Message) {
	p.commitMu.Lock()
	p.toCommit = append(p.toCommit, msg)
	p.commitMu.Unlock()
}

func (p *Processor) commitLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.flushCommits(ctx)
		}
	}
}

func (p *Processor) flushCommits(ctx context.Context) {
	p.commitMu.Lock()
	batch := p.toCommit
	p.toCommit = nil
	p.commitMu.Unlock()

	if len(batch) == 0 {
		return
	}

	commitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := p.reader.CommitMessages(commitCtx, batch...); err != nil {
		p.log.Error("commit failed; returning to pending", "error", err, "count", len(batch))
		p.commitMu.Lock()
		p.toCommit = append(batch, p.toCommit...)
		p.commitMu.Unlock()
		return
	}
	p.log.Debug("committed offsets", "count", len(batch))
}
