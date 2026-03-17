package types

import "time"

type ResponseMetadata struct {
	ID        string
	ModelID   string
	Timestamp time.Time
	Headers   map[string]string
}
