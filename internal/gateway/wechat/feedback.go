package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FeedbackRecord struct {
	UserID         string `json:"user_id"`
	Text           string `json:"text"`
	IdempotencyKey string `json:"idempotency_key"`
	CreatedAtUnix  int64  `json:"created_at_unix"`
}

type FileFeedbackResponder struct {
	Root string
	Now  func() time.Time
}

func (r FileFeedbackResponder) HandleFeedback(ctx context.Context, msg NormalizedMessage) (string, error) {
	_ = ctx

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return "请直接输入你的建议或需求。", nil
	}

	root := r.Root
	if root == "" {
		root = ".codex-runtime/wechat-feedback"
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("create feedback dir: %w", err)
	}

	fp := filepath.Join(root, "feedback.jsonl")
	f, err := os.OpenFile(fp, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("open feedback file: %w", err)
	}
	defer f.Close()

	record := FeedbackRecord{
		UserID:         msg.UserID,
		Text:           text,
		IdempotencyKey: msg.IdempotencyKey,
		CreatedAtUnix:  r.now().Unix(),
	}
	if err := json.NewEncoder(f).Encode(record); err != nil {
		return "", fmt.Errorf("append feedback record: %w", err)
	}

	return "已收到反馈，后续会并入新版需求整理。", nil
}

func (r FileFeedbackResponder) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

var _ FeedbackResponder = (*FileFeedbackResponder)(nil)
