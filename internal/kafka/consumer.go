package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/sirupsen/logrus"
	"github.com/twmb/franz-go/pkg/kgo"
)

const consumerGroupTransactions = "bank-api-analytics"

type Consumer struct {
	client *kgo.Client
	log    *logrus.Logger
}

func NewConsumer(brokers []string, log *logrus.Logger) (*Consumer, error) {
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
		client: client,
		log:    log,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) {
	c.log.WithFields(logrus.Fields{
		"topic": TopicTransactions,
		"group": consumerGroupTransactions,
	}).Info("kafka consumer started")

	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			c.log.Info("kafka consumer stopped")

			return
		}

		fetches.EachError(func(topic string, partition int32, err error) {
			c.log.WithError(err).WithFields(logrus.Fields{
				"topic":     topic,
				"partition": partition,
			}).Error("kafka fetch error")
		})

		for _, record := range fetches.Records() {
			c.processRecordSafely(ctx, record)
		}
	}
}

func (c *Consumer) processRecordSafely(ctx context.Context, record *kgo.Record) {
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

	c.processRecord(ctx, record)
}

func (c *Consumer) processRecord(ctx context.Context, record *kgo.Record) {
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

	c.log.WithFields(logrus.Fields{
		"event_id":        event.ID,
		"type":            event.Type,
		"from_account_id": event.FromAccountID,
		"to_account_id":   event.ToAccountID,
		"amount":          event.Amount,
		"topic":           record.Topic,
		"partition":       record.Partition,
		"offset":          record.Offset,
	}).Info("kafka event received")

	c.client.MarkCommitRecords(record)
}

func (c *Consumer) Close() {
	c.client.Close()
}
