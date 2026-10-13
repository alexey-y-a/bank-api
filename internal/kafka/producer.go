package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/twmb/franz-go/pkg/kgo"
)

const TopicTransactions = "transactions"

type Producer struct {
	client *kgo.Client
}

func NewProducer(brokers []string) (*Producer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
	)
	if err != nil {
		return nil, fmt.Errorf("create kafka producer: %w", err)
	}

	return &Producer{client: client}, nil
}

func (p *Producer) Ping(ctx context.Context) error {
	err := p.client.Ping(ctx)
	if err != nil {
		return fmt.Errorf("ping kafka brokers: %w", err)
	}

	return nil
}

func (p *Producer) Publish(ctx context.Context, event *domain.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event to json: %w", err)
	}

	key := strconv.FormatInt(event.FromAccountID, 10)
	if event.FromAccountID == 0 {
		key = strconv.FormatInt(event.ToAccountID, 10)
	}

	record := &kgo.Record{
		Topic: TopicTransactions,
		Key:   []byte(key),
		Value: data,
	}

	result := p.client.ProduceSync(ctx, record)
	err = result.FirstErr()
	if err != nil {
		return fmt.Errorf("publish event to kafka topic %q: %w", TopicTransactions, err)
	}

	return nil
}

func (p *Producer) Close() {
	p.client.Close()
}
