package kafka

import (
	"context"
	"sync"

	"github.com/IBM/sarama"
	"github.com/openimsdk/tools/errs"
	extKafka "github.com/openimsdk/tools/mq/kafka"
)

// MultiConsumerGroup subscribes to the same topic across multiple Kafka clusters.
// One consumer group is created per cluster; all fan-in to the same handler.
// Adding or removing clusters only requires updating the config — no code changes needed.
type MultiConsumerGroup struct {
	groups []*extKafka.MConsumerGroup
}

// NewMultiConsumerGroup creates one MConsumerGroup per cluster config.
func NewMultiConsumerGroup(configs []*extKafka.Config, groupID string, topics []string, autoCommitEnable bool) (*MultiConsumerGroup, error) {
	if len(configs) == 0 {
		return nil, errs.New("NewMultiConsumerGroup: at least one cluster config is required")
	}
	groups := make([]*extKafka.MConsumerGroup, 0, len(configs))
	for i, cfg := range configs {
		g, err := extKafka.NewMConsumerGroup(cfg, groupID, topics, autoCommitEnable)
		if err != nil {
			return nil, errs.WrapMsg(err, "NewMConsumerGroup failed", "clusterIndex", i, "addr", cfg.Addr)
		}
		groups = append(groups, g)
	}
	return &MultiConsumerGroup{groups: groups}, nil
}

// RegisterHandleAndConsumer starts one goroutine per cluster and blocks until all exit.
// The same handler receives messages from all clusters.
func (m *MultiConsumerGroup) RegisterHandleAndConsumer(ctx context.Context, handler sarama.ConsumerGroupHandler) {
	var wg sync.WaitGroup
	for _, g := range m.groups {
		wg.Add(1)
		go func(group *extKafka.MConsumerGroup) {
			defer wg.Done()
			group.RegisterHandleAndConsumer(ctx, handler)
		}(g)
	}
	wg.Wait()
}

// GetContextFromMsg delegates to the first group (context extraction is cluster-agnostic).
func (m *MultiConsumerGroup) GetContextFromMsg(cMsg *sarama.ConsumerMessage) context.Context {
	return m.groups[0].GetContextFromMsg(cMsg)
}

// Close closes all cluster consumer groups, returning the last error if any.
func (m *MultiConsumerGroup) Close() error {
	var lastErr error
	for _, g := range m.groups {
		if err := g.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
