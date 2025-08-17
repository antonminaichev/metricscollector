package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPublicKey(t *testing.T) {
	t.Run("empty path returns nil", func(t *testing.T) {
		key, err := LoadPublicKey("")
		assert.NoError(t, err)
		assert.Nil(t, key)
	})

	t.Run("non-existent file returns error", func(t *testing.T) {
		_, err := LoadPublicKey("non-existent-file.pem")
		assert.Error(t, err)
	})

	t.Run("invalid PEM content returns error", func(t *testing.T) {
		tempFile := createTempFile(t, "invalid content")
		defer os.Remove(tempFile)

		_, err := LoadPublicKey(tempFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid public key PEM")
	})

	t.Run("valid RSA public key loads correctly", func(t *testing.T) {
		// Generate test key pair
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		// Create public key file
		pubKeyFile := createTempPublicKeyFile(t, &privateKey.PublicKey)
		defer os.Remove(pubKeyFile)

		loadedKey, err := LoadPublicKey(pubKeyFile)
		assert.NoError(t, err)
		assert.NotNil(t, loadedKey)
		assert.Equal(t, privateKey.PublicKey.N, loadedKey.N)
		assert.Equal(t, privateKey.PublicKey.E, loadedKey.E)
	})

	t.Run("invalid key type returns error", func(t *testing.T) {
		// Create a file with non-RSA key PEM
		tempFile := createTempFile(t, "-----BEGIN PUBLIC KEY-----\nInvalidKeyData\n-----END PUBLIC KEY-----")
		defer os.Remove(tempFile)

		_, err := LoadPublicKey(tempFile)
		assert.Error(t, err)
	})
}

func TestLoadPrivateKey(t *testing.T) {
	t.Run("empty path returns nil", func(t *testing.T) {
		key, err := LoadPrivateKey("")
		assert.NoError(t, err)
		assert.Nil(t, key)
	})

	t.Run("non-existent file returns error", func(t *testing.T) {
		_, err := LoadPrivateKey("non-existent-file.pem")
		assert.Error(t, err)
	})

	t.Run("invalid PEM content returns error", func(t *testing.T) {
		tempFile := createTempFile(t, "invalid content")
		defer os.Remove(tempFile)

		_, err := LoadPrivateKey(tempFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid private key PEM")
	})

	t.Run("valid RSA private key loads correctly", func(t *testing.T) {
		// Generate test key pair
		originalKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		// Create private key file
		privKeyFile := createTempPrivateKeyFile(t, originalKey)
		defer os.Remove(privKeyFile)

		loadedKey, err := LoadPrivateKey(privKeyFile)
		assert.NoError(t, err)
		assert.NotNil(t, loadedKey)
		assert.Equal(t, originalKey.N, loadedKey.N)
		assert.Equal(t, originalKey.E, loadedKey.E)
	})

	t.Run("invalid key format returns error", func(t *testing.T) {
		tempFile := createTempFile(t, "-----BEGIN PRIVATE KEY-----\nInvalidKeyData\n-----END PRIVATE KEY-----")
		defer os.Remove(tempFile)

		_, err := LoadPrivateKey(tempFile)
		assert.Error(t, err)
		// Просто проверяем, что есть ошибка
	})
}

func TestEncryptRSA(t *testing.T) {
	// Generate test key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("successful encryption", func(t *testing.T) {
		plaintext := []byte("Hello, World!")
		ciphertext, err := EncryptRSA(&privateKey.PublicKey, plaintext)
		assert.NoError(t, err)
		assert.NotEmpty(t, ciphertext)
		assert.NotEqual(t, plaintext, ciphertext)
	})
}

func TestDecryptRSA(t *testing.T) {
	// Generate test key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("successful decryption", func(t *testing.T) {
		plaintext := []byte("Hello, World!")
		ciphertext, err := EncryptRSA(&privateKey.PublicKey, plaintext)
		require.NoError(t, err)

		decrypted, err := DecryptRSA(privateKey, ciphertext)
		assert.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("invalid ciphertext returns error", func(t *testing.T) {
		invalidCiphertext := []byte("invalid ciphertext")
		_, err := DecryptRSA(privateKey, invalidCiphertext)
		assert.Error(t, err)
	})
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	// Generate test key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	testCases := [][]byte{
		[]byte("Hello, World!"),
		[]byte(""),
		[]byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit."),
		[]byte("1234567890"),
		[]byte("Special chars: !@#$%^&*()"),
	}

	for _, plaintext := range testCases {
		t.Run("round trip", func(t *testing.T) {
			ciphertext, err := EncryptRSA(&privateKey.PublicKey, plaintext)
			require.NoError(t, err)

			decrypted, err := DecryptRSA(privateKey, ciphertext)
			require.NoError(t, err)

			assert.Equal(t, plaintext, decrypted)
		})
	}
}

// Helper functions

func createTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "test_*.pem")
	require.NoError(t, err)
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	return tmpFile.Name()
}

func createTempPublicKeyFile(t *testing.T, publicKey *rsa.PublicKey) string {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	require.NoError(t, err)

	pubKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}

	pubKeyPEM := pem.EncodeToMemory(pubKeyBlock)
	return createTempFile(t, string(pubKeyPEM))
}

func createTempPrivateKeyFile(t *testing.T, privateKey *rsa.PrivateKey) string {
	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	privKeyBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	}

	privKeyPEM := pem.EncodeToMemory(privKeyBlock)
	return createTempFile(t, string(privKeyPEM))
}
