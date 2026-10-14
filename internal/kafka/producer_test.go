package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
)

func newTestProducer(t *testing.T) (*Producer, *kfake.Cluster) {
	t.Helper()

	cluster := kfake.MustCluster(
		kfake.NumBrokers(1),
		kfake.SeedTopics(3, TopicTransactions),
	)

	t.Cleanup(cluster.Close)

	producer, err := NewProducer(cluster.ListenAddrs())
	require.NoError(t, err)

	t.Cleanup(producer.Close)

	return producer, cluster
}

func cunsumeOne(t *testing.T, cluster *kfake.Cluster) *kgo.Record {
	t.Helper()

	consumer, err := kgo.NewClient(
		kgo.SeedBrokers(cluster.ListenAddrs()...),
		kgo.ConsumeTopics(TopicTransactions),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)

	require.NoError(t, err)
	t.Cleanup(consumer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fetches := consumer.PollFetches(ctx)

	require.NoError(t, fetches.Err())
	require.Len(t, fetches.Records(), 1)

	return fetches.Records()[0]
}

func TestProducerPublish(t *testing.T) {
	tests := []struct {
		name      string
		eventType domain.EventType
		fromID    int64
		toID      int64
		wantKey   string
	}{
		{
			name:      "transfer uses from account as key",
			eventType: domain.EventTypeTransfer,
			fromID:    1,
			toID:      2,
			wantKey:   "1",
		},
		{
			name:      "deposit uses to account as key",
			eventType: domain.EventTypeDeposit,
			fromID:    0,
			toID:      5,
			wantKey:   "5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			producer, cluster := newTestProducer(t)

			event, err := domain.NewEvent(tt.eventType, tt.fromID, tt.toID, 2500)
			require.NoError(t, err)

			err = producer.Publish(context.Background(), event)
			require.NoError(t, err)

			record := cunsumeOne(t, cluster)

			require.Equal(t, tt.wantKey, string(record.Key))

			var got domain.Event
			err = json.Unmarshal(record.Value, &got)
			require.NoError(t, err)

			require.Equal(t, event.ID, got.ID)
			require.Equal(t, tt.eventType, got.Type)
			require.Equal(t, tt.fromID, got.FromAccountID)
			require.Equal(t, tt.toID, got.ToAccountID)
			require.Equal(t, event.Amount, got.Amount)

			require.WithinDuration(t, event.CreatedAt, got.CreatedAt, time.Second)
		})
	}
}

func TestProducerPing(t *testing.T) {
	producer, _ := newTestProducer(t)

	err := producer.Ping(context.Background())
	require.NoError(t, err)
}
