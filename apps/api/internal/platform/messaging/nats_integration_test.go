//go:build integration

package messaging

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestJetStreamPublishesAndConsumes(t *testing.T) {
	natsURL := os.Getenv("IMAGEFLOW_TEST_NATS_URL")
	if natsURL == "" {
		t.Skip("IMAGEFLOW_TEST_NATS_URL is not configured")
	}

	suffix := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	stream := "IMAGEFLOW_TEST_" + suffix
	subject := "imageflow.test." + suffix
	consumer := "imageflow-test-worker-" + suffix
	deadLetterStream := stream + "_DLQ"
	deadLetterSubject := subject + ".failed"
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	client, err := Open(natsURL, stream, subject, consumer, deadLetterStream, deadLetterSubject, logger)
	if err != nil {
		t.Fatalf("open JetStream: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	t.Cleanup(func() {
		connection, connectErr := nats.Connect(natsURL)
		if connectErr != nil {
			return
		}
		defer connection.Close()
		jetStream, contextErr := connection.JetStream()
		if contextErr == nil {
			_ = jetStream.DeleteStream(stream)
			_ = jetStream.DeleteStream(deadLetterStream)
		}
	})

	_, err = client.context.AddConsumer(stream, &nats.ConsumerConfig{
		Durable:       consumer,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    10,
		FilterSubject: subject,
	})
	if err != nil {
		t.Fatalf("create legacy consumer: %v", err)
	}
	subscription, err := client.Subscribe()
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	consumerInfo, err := client.context.ConsumerInfo(stream, consumer)
	if err != nil {
		t.Fatalf("inspect consumer: %v", err)
	}
	if consumerInfo.Config.MaxDeliver != -1 {
		t.Fatalf("consumer MaxDeliver = %d, want -1", consumerInfo.Config.MaxDeliver)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	payload := []byte(`{"job_id":"job_integration"}`)
	if err := client.Publish(ctx, "event-integration", payload); err != nil {
		t.Fatalf("publish: %v", err)
	}
	messages, err := subscription.Fetch(1, nats.MaxWait(5*time.Second))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(messages) != 1 || string(messages[0].Data) != string(payload) {
		t.Fatalf("messages = %#v, want payload %q", messages, payload)
	}
	if err := messages[0].AckSync(); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}

	if err := client.PublishDeadLetter(ctx, "event-integration", subject, payload, "processing failed", 10); err != nil {
		t.Fatalf("publish dead letter: %v", err)
	}
	info, err := client.context.StreamInfo(deadLetterStream)
	if err != nil {
		t.Fatalf("inspect dead-letter stream: %v", err)
	}
	stored, err := client.context.GetMsg(deadLetterStream, info.State.LastSeq)
	if err != nil {
		t.Fatalf("read dead-letter message: %v", err)
	}
	var envelope deadLetterEnvelope
	if err := json.Unmarshal(stored.Data, &envelope); err != nil {
		t.Fatalf("decode dead-letter message: %v", err)
	}
	if envelope.EventID != "event-integration" || envelope.Deliveries != 10 || string(envelope.Payload) != string(payload) {
		t.Fatalf("dead-letter envelope = %#v", envelope)
	}
}
