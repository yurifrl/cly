package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	keyFileName = "encryption.key"
	nonceSize   = 12
	keySize     = 32 // 256 bits
	magicHeader = "NSXENC:"
)

func BaseFolder() string {
	configPath := os.Getenv("NSX_CONFIG_PATH")
	if configPath != "" {
		return configPath
	}

	return filepath.Join(os.Getenv("HOME"), ".config", "nsx")
}

// getEncryptionKey retrieves the encryption key from the BaseFolder or creates a new one if it doesn't exist
func getEncryptionKey() ([]byte, error) {
	keyPath := filepath.Join(BaseFolder(), keyFileName)

	// Check if key file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		// Generate a new key
		key := make([]byte, keySize)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, fmt.Errorf("failed to generate encryption key: %w", err)
		}

		// Create the base directory if it doesn't exist
		if err := os.MkdirAll(BaseFolder(), 0o700); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}

		// Save the key to a file
		keyHex := hex.EncodeToString(key)
		if err := os.WriteFile(keyPath, []byte(keyHex), 0o600); err != nil {
			return nil, fmt.Errorf("failed to save encryption key: %w", err)
		}

		return key, nil
	}

	// Read existing key
	keyHex, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read encryption key: %w", err)
	}

	key, err := hex.DecodeString(string(keyHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption key: %w", err)
	}

	return key, nil
}

// Encrypt encrypts the provided data using AES-GCM
func Encrypt(data []byte) ([]byte, error) {
	key, err := getEncryptionKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create a nonce
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data
	ciphertext := aesGCM.Seal(nonce, nonce, data, nil)

	// Prepend the magic header to identify encrypted data
	return append([]byte(magicHeader), ciphertext...), nil
}

// Decrypt decrypts the provided data using AES-GCM
func Decrypt(data []byte) ([]byte, error) {
	// Check if data starts with the magic header
	if len(data) <= len(magicHeader) || string(data[:len(magicHeader)]) != magicHeader {
		return nil, errors.New("data is not encrypted with the expected format")
	}

	// Remove the magic header
	data = data[len(magicHeader):]

	key, err := getEncryptionKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	// Extract the nonce and ciphertext
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plaintext, nil
}

// IsEncrypted checks if the provided data is encrypted
func IsEncrypted(data []byte) bool {
	return len(data) > len(magicHeader) && string(data[:len(magicHeader)]) == magicHeader
}

func Load[C any](name string) (*C, error) {
	config := new(C)
	configPath := filepath.Join(BaseFolder(), name)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Check if the data is encrypted and decrypt if needed
	if IsEncrypted(data) {
		data, err = Decrypt(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt config file: %w", err)
		}
	}

	if _, err := toml.Decode(string(data), config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}
