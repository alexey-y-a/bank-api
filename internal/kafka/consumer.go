package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
	"github.com/sirupsen/logrus"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	consumerGroupTransactions = "bank-api-analytics"
	batchSize                 = 100
	flushInterval             = 2 * time.Second
	shutdownFlushTimeout      = 5 * time.Second
)

type bufferedEvent struct {
	event  *domain.Event
	record *kgo.Record
}

type Consumer struct {
	client    *kgo.Client
	analytics repository.AnalyticsRepository
	log       *logrus.Logger
	buffer    []bufferedEvent
}

func NewConsumer(brokers []string, analytics repository.AnalyticsRepository, log *logrus.Logger) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(consumerGroupTransactions),
		kgo.ConsumeTopics(TopicTransactions),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.AutoCommitMarks(),
	)
	if err != nil {
		return nil, fmt.Errorf("create kafka consumer: %w", err)
	}

	return &Consumer{
		client:    client,
		analytics: analytics,
		log:       log,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) {
	c.log.WithFields(logrus.Fields{
		"topic": TopicTransactions,
		"group": consumerGroupTransactions,
	}).Info("kafka consumer started")

	var lastFlush time.Time

	for {
		pollCtx, cancelPoll := context.WithTimeout(ctx, flushInterval)
		fetches := c.client.PollFetches(pollCtx)
		cancelPoll()

		fetches.EachError(func(topic string, partition int32, err error) {
			if errors.Is(err, context.DeadlineExceeded) {
				return
			}

			c.log.WithError(err).WithFields(logrus.Fields{
				"topic":     topic,
				"partition": partition,
			}).Error("kafka fetch error")
		})

		for _, record := range fetches.Records() {
			c.processRecordSafely(record)
		}

		if ctx.Err() != nil {
			c.flushOnShutdown()
			c.log.Info("kafka consumer stopped")

			return
		}

		if len(c.buffer) >= batchSize || time.Since(lastFlush) > flushInterval {
			c.flush(ctx)

			if len(c.buffer) == 0 {
				lastFlush = time.Now()
			}
		}
	}
}

func (c *Consumer) processRecordSafely(record *kgo.Record) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		c.log.WithFields(logrus.Fields{
			"panic":     r,
			"topic":     record.Topic,
			"partition": record.Partition,
			"offset":    record.Offset,
		}).Error("panic while processing kafka record")

		c.client.MarkCommitRecords(record)
	}()

	c.processRecord(record)
}

func (c *Consumer) processRecord(record *kgo.Record) {
	var event domain.Event

	err := json.Unmarshal(record.Value, &event)
	if err != nil {
		c.log.WithError(err).WithFields(logrus.Fields{
			"topic":     record.Topic,
			"partition": record.Partition,
			"offset":    record.Offset,
		}).Error("skip invalid kafka event")

		c.client.MarkCommitRecords(record)

		return
	}

	c.buffer = append(c.buffer, bufferedEvent{
		event:  &event,
		record: record,
	})

	c.log.WithFields(logrus.Fields{
		"event_id": event.ID,
		"type":     event.Type,
		"amount":   event.Amount,
	}).Debug("kafka event buffered")
}

func (c *Consumer) flush(ctx context.Context) {
	if len(c.buffer) == 0 {
		return
	}

	events := make([]*domain.Event, 0, len(c.buffer))
	records := make([]*kgo.Record, 0, len(c.buffer))

	for _, item := range c.buffer {
		events = append(events, item.event)
		records = append(records, item.record)
	}

	err := c.analytics.SaveBatch(ctx, events)
	if err != nil {
		c.log.WithError(err).WithField("batch_size", len(c.buffer)).Error("flush events to clickhouse")

		return
	}

	c.client.MarkCommitRecords(records...)
	c.buffer = c.buffer[:0]

	c.log.WithFields(logrus.Fields{
		"batch_size": len(records),
	}).Info("kafka events stored to clickhouse")
}

func (c *Consumer) flushOnShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownFlushTimeout)

	defer cancel()

	c.flush(ctx)
}

func (c *Consumer) Close() {
	c.client.Close()
}
