package service

import (
	"context"
	"log/slog"
	"sync"

	"github.com/segmentio/kafka-go"

	"gw-analytics/internal/config"
	"gw-analytics/internal/consumer"
	"gw-analytics/internal/domain"
	"gw-analytics/internal/storage"
	"gw-analytics/pkg/retry"
)

type Processor struct {
	reader  *kafka.Reader
	storage storage.Storage
	dlq     *consumer.DLQWriter
	log     *slog.Logger
	cfg     *config.Config
	retry   retry.Config

	commitMu sync.Mutex
	toCommit []kafka.Message
}

func NewProcessor(
	r *kafka.Reader,
	st storage.Storage,
	dlq *consumer.DLQWriter,
	cfg *config.Config,
	log *slog.Logger,
) *Processor {
	return &Processor{
		reader:  r,
		storage: st,
		dlq:     dlq,
		log:     log,
		cfg:     cfg,
		retry: retry.Config{
			MaxAttempts: cfg.RetryMaxAttempts,
			Initial:     cfg.RetryInitial,
			Max:         cfg.RetryMax,
		},
	}
}

type batchItem struct {
	msg kafka.Message
	row domain.ClickHouseRow
}

func (p *Processor) Run(ctx context.Context) error {
	incoming := make(chan batchItem, p.cfg.QueueSize)

	commitCtx, cancelCommit := context.WithCancel(context.Background())
	commitDone := make(chan struct{})
	go func() {
		p.commitLoop(commitCtx)
		close(commitDone)
	}()

	readerErrCh := make(chan error, 1)
	go func() {
		readerErrCh <- p.readLoop(ctx, incoming)
		close(incoming)
	}()

	batchErr := p.batchLoop(ctx, incoming)
	readerErr := <-readerErrCh

	cancelCommit()
	<-commitDone
	p.flushCommits(context.Background())

	if batchErr != nil {
		return batchErr
	}
	return readerErr
}
