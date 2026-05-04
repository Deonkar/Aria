package handlers

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const maxQuestionLen = 500

// NormalizeQuestion trims, rejects empty/whitespace-only, enforces max length (TASK-035).
func NormalizeQuestion(q string) (string, error) {
	s := strings.TrimSpace(q)
	if s == "" {
		return "", fmt.Errorf("question required")
	}
	if len(s) > maxQuestionLen {
		return "", fmt.Errorf("question too long")
	}
	return s, nil
}

// ValidateSessionUUID returns an error if s is non-empty and not a valid UUID.
func ValidateSessionUUID(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	if _, err := uuid.Parse(strings.TrimSpace(s)); err != nil {
		return fmt.Errorf("invalid session_id")
	}
	return nil
}
