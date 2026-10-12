package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

type JetStream struct {
	connection *nats.Conn
	context    nats.JetStreamContext
	stream     string
	subject    string
	consumer   string
}

func Open(url, stream, subject, consumer string, logger *slog.Logger) (*JetStream, error) {
	connection, err := nats.Connect(
		url,
		nats.Name("imageflow-worker"),
		nats.Timeout(5*time.Second),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			logger.Warn("nats disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(connection *nats.Conn) {
			logger.Info("nats reconnected", "url", connection.ConnectedUrl())
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to nats: %w", err)
	}
	jetStream, err := connection.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		connection.Close()
		return nil, fmt.Errorf("open jetstream context: %w", err)
	}
	client := &JetStream{connection: connection, context: jetStream, stream: stream, subject: subject, consumer: consumer}
	if err := client.ensureStream(); err != nil {
		connection.Close()
		return nil, err
	}
	return client, nil
}

func (client *JetStream) ensureStream() error {
	if _, err := client.context.StreamInfo(client.stream); err == nil {
		return nil
	} else if err != nats.ErrStreamNotFound {
		return fmt.Errorf("inspect stream: %w", err)
	}
	_, err := client.context.AddStream(&nats.StreamConfig{
		Name:       client.stream,
		Subjects:   []string{client.subject},
		Retention:  nats.WorkQueuePolicy,
		Storage:    nats.FileStorage,
		MaxAge:     7 * 24 * time.Hour,
		Duplicates: 2 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("create stream: %w", err)
	}
	return nil
}

func (client *JetStream) Publish(ctx context.Context, eventID string, payload []byte) error {
	message := nats.NewMsg(client.subject)
	message.Data = payload
	message.Header.Set(nats.MsgIdHdr, eventID)
	if _, err := client.context.PublishMsg(message, nats.Context(ctx)); err != nil {
		return fmt.Errorf("publish jetstream message: %w", err)
	}
	return nil
}

func (client *JetStream) Subscribe() (*nats.Subscription, error) {
	subscription, err := client.context.PullSubscribe(
		client.subject,
		client.consumer,
		nats.BindStream(client.stream),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.AckWait(30*time.Second),
		nats.MaxDeliver(10),
	)
	if err != nil {
		return nil, fmt.Errorf("create durable consumer: %w", err)
	}
	return subscription, nil
}

func (client *JetStream) Close() error {
	if err := client.connection.Drain(); err != nil {
		client.connection.Close()
		return fmt.Errorf("drain nats connection: %w", err)
	}
	return nil
}
