package entity

import "time"

type IdempotencyRecord struct {
	key          string
	responseCode int
	responseBody []byte
	createdAt    time.Time
}

func NewIdempotencyRecord(key string, code int, body []byte) *IdempotencyRecord {
	return &IdempotencyRecord{
		key:          key,
		responseCode: code,
		responseBody: body,
		createdAt:    time.Now(),
	}
}

func ReconstructIdempotencyRecord(key string, code int, body []byte, createdAt time.Time) *IdempotencyRecord {
	return &IdempotencyRecord{
		key:          key,
		responseCode: code,
		responseBody: body,
		createdAt:    createdAt,
	}
}

func (r *IdempotencyRecord) Key() string {
	return r.key
}

func (r *IdempotencyRecord) ResponseCode() int {
	return r.responseCode
}

func (r *IdempotencyRecord) ResponseBody() []byte {
	return r.responseBody
}

func (r *IdempotencyRecord) CreatedAt() time.Time {
	return r.createdAt
}
