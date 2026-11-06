package kafka

import (
	"context"
	"errors"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafkago.Reader
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	return &Consumer{
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:     brokers,
			Topic:       topic,
			GroupID:     groupID,
			StartOffset: kafkago.FirstOffset,
		}),
	}
}

func (c *Consumer) Run(ctx context.Context, handle func(ctx context.Context, value []byte) error) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		if err := handle(ctx, msg.Value); err != nil {
			return fmt.Errorf("handle message: %w", err)
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("commit message: %w", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
