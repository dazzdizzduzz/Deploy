package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	ServerPort    string
	JWTPrivateKey string
	JWTPublicKey  string
}

// LoadEnv memuat file .env atau .env.production ke environment variable proses jika belum di-set.
// Mencari di lokasi default prioritas (".env.production", ".env", "backend/.env.production", dst).
func LoadEnv(filenames ...string) {
	if len(filenames) == 0 {
		filenames = []string{
			".env.production",
			".env",
			"backend/.env.production",
			"backend/.env",
			"../.env.production",
			"../.env",
			"../../.env.production",
			"../../.env",
		}
	}

	for _, filename := range filenames {
		if err := parseEnvFile(filename); err == nil {
			break
		}
	}
}

func parseEnvFile(filename string) error {
	cleanPath := filepath.Clean(filename)
	file, err := os.Open(cleanPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	// Buffer cukup besar untuk membaca kunci privat/publik multiline atau base64
	const maxCapacity = 1024 * 1024 // 1 MB
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	var currentKey string
	var currentVal strings.Builder
	var inMultiline bool
	var multilineQuote byte

	for scanner.Scan() {
		line := scanner.Text()

		if inMultiline {
			trimmedRight := strings.TrimRight(line, "\r \t")
			if strings.HasSuffix(trimmedRight, string(multilineQuote)) {
				// Akhir dari multiline quoted string
				currentVal.WriteString("\n")
				currentVal.WriteString(trimmedRight[:len(trimmedRight)-1])
				inMultiline = false

				val := unescapeEnvValue(currentVal.String(), multilineQuote == '"')
				if os.Getenv(currentKey) == "" {
					_ = os.Setenv(currentKey, val)
				}
				currentKey = ""
				currentVal.Reset()
			} else {
				currentVal.WriteString("\n")
				currentVal.WriteString(line)
			}
			continue
		}

		trimmed := strings.TrimSpace(line)
		// Abaikan baris kosong dan baris komentar
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Pisahkan KEY=VALUE
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Single-line quoted string ("..." atau '...')
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			quote := val[0]
			val = val[1 : len(val)-1]
			val = unescapeEnvValue(val, quote == '"')
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		} else if len(val) >= 1 && (val[0] == '"' || val[0] == '\'') && !strings.HasSuffix(val, string(val[0])) {
			// Awal dari multiline quote
			inMultiline = true
			multilineQuote = val[0]
			currentKey = key
			currentVal.Reset()
			currentVal.WriteString(val[1:])
		} else {
			// Unquoted value
			val = unescapeEnvValue(val, false)
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
	}

	// Tangani kasus quote multiline yang belum ditutup hingga EOF
	if inMultiline && currentKey != "" {
		val := unescapeEnvValue(currentVal.String(), multilineQuote == '"')
		if os.Getenv(currentKey) == "" {
			_ = os.Setenv(currentKey, val)
		}
	}

	return scanner.Err()
}

func unescapeEnvValue(val string, wasDoubleQuoted bool) string {
	if wasDoubleQuoted || strings.Contains(val, `\n`) {
		val = strings.ReplaceAll(val, `\n`, "\n")
		val = strings.ReplaceAll(val, `\r`, "\r")
		val = strings.ReplaceAll(val, `\t`, "\t")
		if wasDoubleQuoted {
			val = strings.ReplaceAll(val, `\"`, `"`)
			val = strings.ReplaceAll(val, `\'`, `'`)
			val = strings.ReplaceAll(val, `\\`, `\`)
		}
	}
	return val
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func LoadConfig() *Config {
	LoadEnv()

	return &Config{
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "admin"),
		DBPassword:    getEnv("DB_PASSWORD", "adminpassword"),
		DBName:        getEnv("DB_NAME", "ontrack_db"),
		DBSSLMode:     getEnv("DB_SSLMODE", "disable"),
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		JWTPrivateKey: os.Getenv("JWT_PRIVATE_KEY"),
		JWTPublicKey:  os.Getenv("JWT_PUBLIC_KEY"),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}
