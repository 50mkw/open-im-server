package kafka

import (
	"context"
	"hash/fnv"

	"github.com/openimsdk/open-im-server/v3/pkg/common/storage/kafka"
	"github.com/openimsdk/tools/errs"
	"google.golang.org/protobuf/proto"
)

// MultiProducer routes messages to multiple Kafka clusters.
// Cluster is selected by: hash(key) % len(clusters)
// This guarantees the same conversationID always lands on the same cluster,
// preserving ordering guarantees.
type MultiProducer struct {
	producers []*kafka.Producer
}

// NewMultiProducer creates one producer per cluster config.
// If only one config is provided it behaves identically to a plain Producer.
func NewMultiProducer(configs []*kafka.Config, topic string) (*MultiProducer, error) {
	if len(configs) == 0 {
		return nil, errs.New("NewMultiProducer: at least one cluster config is required")
	}
	producers := make([]*kafka.Producer, 0, len(configs))
	for i, cfg := range configs {
		saramaCfg, err := kafka.BuildProducerConfig(*cfg)
		if err != nil {
			return nil, errs.WrapMsg(err, "BuildProducerConfig failed", "clusterIndex", i, "addr", cfg.Addr)
		}
		p, err := kafka.NewKafkaProducer(saramaCfg, cfg.Addr, topic)
		if err != nil {
			return nil, errs.WrapMsg(err, "NewKafkaProducer failed", "clusterIndex", i, "addr", cfg.Addr)
		}
		producers = append(producers, p)
	}
	return &MultiProducer{producers: producers}, nil
}

// SendMessage selects the target cluster via hash(key) % numClusters, then sends.
func (m *MultiProducer) SendMessage(ctx context.Context, key string, msg proto.Message) (int32, int64, error) {
	idx := clusterIndex(key, len(m.producers))
	return m.producers[idx].SendMessage(ctx, key, msg)
}

// clusterIndex returns a stable, non-negative index for the given key and cluster count.
func clusterIndex(key string, n int) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % n
}
