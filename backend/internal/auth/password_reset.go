package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	ResetTokenDuration = 15 * time.Minute
)

type resetTokenEntry struct {
	userID    string
	email     string
	expiresAt time.Time
}

type PasswordResetManager struct {
	mu     sync.RWMutex
	tokens map[string]*resetTokenEntry
}

func NewPasswordResetManager() *PasswordResetManager {
	return &PasswordResetManager{
		tokens: make(map[string]*resetTokenEntry),
	}
}

// GenerateToken membuat token acak aman 32-byte (64 karakter hex),
// mencatatnya ke log konsol (simulasi SMTP), dan menyimpannya selama 15 menit.
func (m *PasswordResetManager) GenerateToken(userID, email string) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure random token: %w", err)
	}

	token := hex.EncodeToString(bytes)

	m.mu.Lock()
	// Hapus token lama untuk email yang sama jika ada
	for t, entry := range m.tokens {
		if entry.email == email {
			delete(m.tokens, t)
		}
	}

	m.tokens[token] = &resetTokenEntry{
		userID:    userID,
		email:     email,
		expiresAt: time.Now().Add(ResetTokenDuration),
	}
	m.mu.Unlock()

	// Simulasi pengiriman email SMTP dengan mencetak token ke konsol (Requirement 2)
	log.Printf("================================================================================")
	log.Printf("[SIMULASI LAYANAN SMTP - TOKEN RESET KATA SANDI]")
	log.Printf("Tujuan  : %s", email)
	log.Printf("Token   : %s", token)
	log.Printf("Berlaku : %s (15 Menit)", time.Now().Add(ResetTokenDuration).Format(time.RFC3339))
	log.Printf("================================================================================")

	return token, nil
}

// ValidateAndConsumeToken memvalidasi token reset kata sandi,
// mengembalikan userID dan email jika valid, lalu langsung menghapusnya (single-use).
func (m *PasswordResetManager) ValidateAndConsumeToken(token string) (string, string, error) {
	if token == "" {
		return "", "", errors.New("token cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.tokens[token]
	if !exists {
		return "", "", errors.New("invalid or non-existent reset token")
	}

	if time.Now().After(entry.expiresAt) {
		delete(m.tokens, token)
		return "", "", errors.New("reset token has expired")
	}

	userID := entry.userID
	email := entry.email

	// Konsumsi token (single-use)
	delete(m.tokens, token)

	return userID, email, nil
}

