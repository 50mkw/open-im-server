package kafka

import (
	"context"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"
)

// MessageProducer abstracts single and multi-cluster Kafka producers.
type MessageProducer interface {
	SendMessage(ctx context.Context, key string, msg proto.Message) (int32, int64, error)
}

// ConsumerGroupHandler abstracts single and multi-cluster Kafka consumer groups.
type ConsumerGroupHandler interface {
	RegisterHandleAndConsumer(ctx context.Context, handler sarama.ConsumerGroupHandler)
	GetContextFromMsg(cMsg *sarama.ConsumerMessage) context.Context
	Close() error
}
