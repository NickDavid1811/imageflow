package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

type JetStream struct {
	connection        *nats.Conn
	context           nats.JetStreamContext
	stream            string
	subject           string
	consumer          string
	deadLetterStream  string
	deadLetterSubject string
}

func Open(url, stream, subject, consumer, deadLetterStream, deadLetterSubject string, logger *slog.Logger) (*JetStream, error) {
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
	client := &JetStream{
		connection: connection, context: jetStream, stream: stream, subject: subject, consumer: consumer,
		deadLetterStream: deadLetterStream, deadLetterSubject: deadLetterSubject,
	}
	if err := client.ensureStream(stream, subject, nats.WorkQueuePolicy, 7*24*time.Hour); err != nil {
		connection.Close()
		return nil, err
	}
	if err := client.ensureStream(deadLetterStream, deadLetterSubject, nats.LimitsPolicy, 30*24*time.Hour); err != nil {
		connection.Close()
		return nil, err
	}
	return client, nil
}

func (client *JetStream) ensureStream(name, subject string, retention nats.RetentionPolicy, maxAge time.Duration) error {
	if info, err := client.context.StreamInfo(name); err == nil {
		if info.Config.Retention != retention || info.Config.Storage != nats.FileStorage || len(info.Config.Subjects) != 1 || info.Config.Subjects[0] != subject {
			return fmt.Errorf("stream %s has incompatible configuration", name)
		}
		return nil
	} else if err != nats.ErrStreamNotFound {
		return fmt.Errorf("inspect stream: %w", err)
	}
	_, err := client.context.AddStream(&nats.StreamConfig{
		Name:       name,
		Subjects:   []string{subject},
		Retention:  retention,
		Storage:    nats.FileStorage,
		MaxAge:     maxAge,
		Duplicates: 2 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("create stream: %w", err)
	}
	return nil
}

type deadLetterEnvelope struct {
	EventID    string    `json:"event_id"`
	Source     string    `json:"source"`
	Payload    []byte    `json:"payload"`
	Error      string    `json:"error"`
	Deliveries uint64    `json:"deliveries"`
	FailedAt   time.Time `json:"failed_at"`
}

func (client *JetStream) PublishDeadLetter(ctx context.Context, eventID, source string, payload []byte, cause string, deliveries uint64) error {
	envelope, err := json.Marshal(deadLetterEnvelope{
		EventID: eventID, Source: source, Payload: payload, Error: cause, Deliveries: deliveries, FailedAt: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("encode dead-letter event: %w", err)
	}
	message := nats.NewMsg(client.deadLetterSubject)
	message.Data = envelope
	message.Header.Set(nats.MsgIdHdr, "dlq_"+eventID)
	if _, err := client.context.PublishMsg(message, nats.Context(ctx)); err != nil {
		return fmt.Errorf("publish dead-letter event: %w", err)
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
	if err := client.ensureUnlimitedDeliveries(); err != nil {
		return nil, err
	}
	subscription, err := client.context.PullSubscribe(
		client.subject,
		client.consumer,
		nats.BindStream(client.stream),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.AckWait(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("create durable consumer: %w", err)
	}
	return subscription, nil
}

func (client *JetStream) ensureUnlimitedDeliveries() error {
	info, err := client.context.ConsumerInfo(client.stream, client.consumer)
	if err == nats.ErrConsumerNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect durable consumer: %w", err)
	}
	if info.Config.MaxDeliver == -1 {
		return nil
	}
	config := info.Config
	config.MaxDeliver = -1
	if _, err := client.context.UpdateConsumer(client.stream, &config); err != nil {
		return fmt.Errorf("update durable consumer delivery policy: %w", err)
	}
	return nil
}

func (client *JetStream) Close() error {
	if err := client.connection.Drain(); err != nil {
		client.connection.Close()
		return fmt.Errorf("drain nats connection: %w", err)
	}
	return nil
}
