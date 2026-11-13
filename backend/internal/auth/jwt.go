package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/models"
)

var (
	ErrMissingAuthHeader = errors.New("authorization header is required")
	ErrInvalidAuthFormat = errors.New("authorization header must be in format: Bearer <token>")
	ErrInvalidToken      = errors.New("token is invalid or expired")
	ErrUnexpectedAlg     = errors.New("unexpected signing algorithm, must be RS256")
	ErrNilPublicKey      = errors.New("rsa public key is not configured")
	ErrNilPrivateKey     = errors.New("rsa private key is not configured")
)

// CustomClaims aliases models.JWTClaims for backward compatibility with existing auth handlers.
type CustomClaims = models.JWTClaims

// JWTManager handles RS256 token issuance and validation.
type JWTManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// JWTService is an alias for JWTManager.
type JWTService = JWTManager

// Config holds keys or paths for initializing JWTManager/JWTService.
type Config struct {
	PublicKeyPEM   string
	PrivateKeyPEM  string
	PublicKeyPath  string
	PrivateKeyPath string
}

// NewJWTManager initializes JWTManager using PEM strings or generates an ephemeral keypair.
func NewJWTManager(privateKeyPEM, publicKeyPEM string) (*JWTManager, error) {
	return NewJWTService(Config{
		PrivateKeyPEM: privateKeyPEM,
		PublicKeyPEM:  publicKeyPEM,
	})
}

// NewJWTService creates a new JWTService with given RSA keys or paths.
func NewJWTService(cfg Config) (*JWTService, error) {
	svc := &JWTService{}

	// 1. Load Public Key from PEM string
	if cfg.PublicKeyPEM != "" {
		pubKey, err := ParseRSAPublicKeyFromPEM([]byte(cfg.PublicKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key PEM: %w", err)
		}
		svc.publicKey = pubKey
	} else if cfg.PublicKeyPath != "" {
		data, err := os.ReadFile(cfg.PublicKeyPath)
		if err == nil {
			pubKey, err := ParseRSAPublicKeyFromPEM(data)
			if err != nil {
				return nil, fmt.Errorf("failed to parse public key from %s: %w", cfg.PublicKeyPath, err)
			}
			svc.publicKey = pubKey
		}
	}

	// 2. Load Private Key from PEM string
	if cfg.PrivateKeyPEM != "" {
		privKey, err := ParseRSAPrivateKeyFromPEM([]byte(cfg.PrivateKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key PEM: %w", err)
		}
		svc.privateKey = privKey
	} else if cfg.PrivateKeyPath != "" {
		data, err := os.ReadFile(cfg.PrivateKeyPath)
		if err == nil {
			privKey, err := ParseRSAPrivateKeyFromPEM(data)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key from %s: %w", cfg.PrivateKeyPath, err)
			}
			svc.privateKey = privKey
		}
	}

	// Fallback to certs directory if paths were not specified
	if svc.publicKey == nil {
		for _, path := range []string{"certs/jwt_public.pem", "backend/certs/jwt_public.pem", "../certs/jwt_public.pem"} {
			if data, err := os.ReadFile(path); err == nil {
				if pubKey, err := ParseRSAPublicKeyFromPEM(data); err == nil {
					svc.publicKey = pubKey
					break
				}
			}
		}
	}
	if svc.privateKey == nil {
		for _, path := range []string{"certs/jwt_private.pem", "backend/certs/jwt_private.pem", "../certs/jwt_private.pem"} {
			if data, err := os.ReadFile(path); err == nil {
				if privKey, err := ParseRSAPrivateKeyFromPEM(data); err == nil {
					svc.privateKey = privKey
					break
				}
			}
		}
	}

	// Dynamic fallback generation if no key configured (for dev/testing)
	if svc.publicKey == nil || svc.privateKey == nil {
		if svc.privateKey != nil && svc.publicKey == nil {
			svc.publicKey = &svc.privateKey.PublicKey
		} else {
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				return nil, fmt.Errorf("failed to generate ephemeral RSA keypair: %w", err)
			}
			svc.privateKey = key
			svc.publicKey = &key.PublicKey
		}
	}

	return svc, nil
}

// NewJWTServiceWithKeys initializes JWTService directly with RSA keys.
func NewJWTServiceWithKeys(pubKey *rsa.PublicKey, privKey *rsa.PrivateKey) *JWTService {
	return &JWTService{
		publicKey:  pubKey,
		privateKey: privKey,
	}
}

// ParseRSAPublicKeyFromPEM decodes an RSA public key in PKCS#1, PKIX, or X.509 cert format.
func ParseRSAPublicKeyFromPEM(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing public key")
	}

	// Try parsing as PKIX (Standard SPKI)
	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if rsaPub, ok := pub.(*rsa.PublicKey); ok {
			return rsaPub, nil
		}
		return nil, errors.New("parsed key is not an RSA public key")
	}

	// Try parsing as PKCS#1 RSA Public Key
	if pub, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return pub, nil
	}

	// Try parsing as certificate
	if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
		if rsaPub, ok := cert.PublicKey.(*rsa.PublicKey); ok {
			return rsaPub, nil
		}
	}

	return nil, errors.New("failed to parse RSA public key (tried PKIX, PKCS#1, and X.509 cert)")
}

// ParseRSAPrivateKeyFromPEM decodes an RSA private key in PKCS#1 or PKCS#8 format.
func ParseRSAPrivateKeyFromPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing private key")
	}

	// Try parsing as PKCS#8
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaPriv, ok := key.(*rsa.PrivateKey); ok {
			return rsaPriv, nil
		}
		return nil, errors.New("parsed key is not an RSA private key")
	}

	// Try parsing as PKCS#1
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	return nil, errors.New("failed to parse RSA private key (tried PKCS#8 and PKCS#1)")
}

// GenerateToken generates and signs a JWT RS256 token for user login (Issue 11 signature).
func (m *JWTManager) GenerateToken(userID, tenantID, role string) (string, int, error) {
	const ttl = 15 * time.Minute
	expirationTime := time.Now().Add(ttl)

	claims := models.JWTClaims{
		Sub:  userID,
		Tid:  tenantID,
		Role: models.UserRole(role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	tokenString, err := m.GenerateTokenWithClaims(claims, ttl)
	if err != nil {
		return "", 0, err
	}
	return tokenString, int(ttl.Seconds()), nil
}

// GenerateTokenWithClaims generates and signs a JWT RS256 token with custom claims.
func (m *JWTManager) GenerateTokenWithClaims(claims models.JWTClaims, duration time.Duration) (string, error) {
	if m.privateKey == nil {
		return "", ErrNilPrivateKey
	}

	now := time.Now().UTC()
	if claims.IssuedAt == nil {
		claims.IssuedAt = jwt.NewNumericDate(now)
	}
	if claims.ExpiresAt == nil {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(duration))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

// ValidateToken parses and validates a JWT RS256 token string.
func (m *JWTManager) ValidateToken(tokenString string) (*models.JWTClaims, error) {
	if m.publicKey == nil {
		return nil, ErrNilPublicKey
	}

	claims := &models.JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("%w: %v", ErrUnexpectedAlg, t.Header["alg"])
		}
		return m.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.Sub == "" && claims.Subject != "" {
		claims.Sub = claims.Subject
	}
	if claims.Subject == "" && claims.Sub != "" {
		claims.Subject = claims.Sub
	}

	return claims, nil
}

// ExtractBearerToken extracts the raw token from the HTTP Authorization header.
func ExtractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrMissingAuthHeader
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", ErrInvalidAuthFormat
	}

	return strings.TrimSpace(parts[1]), nil
}

// ValidateRequest extracts and validates the Bearer token directly from an http.Request.
func (m *JWTManager) ValidateRequest(r *http.Request) (*models.JWTClaims, error) {
	rawToken, err := ExtractBearerToken(r)
	if err != nil {
		return nil, err
	}
	return m.ValidateToken(rawToken)
}
