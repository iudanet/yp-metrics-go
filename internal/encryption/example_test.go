package encryption_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/iudanet/yp-metrics-go/internal/encryption"
)

func ExampleHybrid() {
	// Создаем временный RSA ключ для примера
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("key generation error: %v\n", err)
		return
	}

	// Сохраняем публичный ключ
	tmpFile, err := os.CreateTemp("", "example_key_*.pem")
	if err != nil {
		fmt.Printf("Temp file error: %v\n", err)
		return
	}
	defer os.Remove(tmpFile.Name())

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		fmt.Printf("Marshal error: %v\n", err)
		return
	}

	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	if err = pem.Encode(tmpFile, publicKeyBlock); err != nil {
		fmt.Printf("PEM encode error: %v\n", err)
		return
	}
	tmpFile.Close()

	// Данные для шифрования
	data := []byte(`{"id":"cpu_usage","value":75.5}`)

	// Шифруем данные
	encrypted, err := encryption.Hybrid(data, tmpFile.Name())
	if err != nil {
		fmt.Printf("Encryption error: %v\n", err)
		return
	}

	fmt.Printf("Encrypted data length: %d bytes\n", len(encrypted))
	// Output: Encrypted data length: 455 bytes
}
