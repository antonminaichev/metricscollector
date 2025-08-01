package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	if path == "" {
		return nil, nil // отключаем шифрование
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid public key PEM")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pub, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not RSA public key")
	}
	return pub, nil
}

func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	if path == "" {
		return nil, nil // отключаем шифрование
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid private key PEM")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}
	return nil, errors.New("not a valid RSA private key")
}

func EncryptRSA(pub *rsa.PublicKey, data []byte) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, pub, data)
}

func DecryptRSA(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, priv, data)
}
