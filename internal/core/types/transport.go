package types

type SoloInput struct {
	UserID         string `json:"user_id"`
	RuntimeID      string `json:"runtime_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Text           string `json:"text"`
}

type RoomInput struct {
	UserID         string `json:"user_id"`
	RuntimeID      string `json:"runtime_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Text           string `json:"text"`
}

