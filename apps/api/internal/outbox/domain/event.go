package domain

import "time"

type Event struct {
	ID          int64
	AggregateID string
	Type        string
	Payload     []byte
	Attempts    int
	CreatedAt   time.Time
}
