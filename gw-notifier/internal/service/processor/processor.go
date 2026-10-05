package processor

import (
	"context"
	"log/slog"
	"sync"

	"github.com/segmentio/kafka-go"

	"gw-notifier/internal/config"
	"gw-notifier/internal/consumer"
	"gw-notifier/internal/domain"
	"gw-notifier/internal/storage"
	"gw-notifier/pkg/retry"
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
	ev  domain.TransferEvent
}

// Run запускает три горутины: reader, batcher, committer
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
