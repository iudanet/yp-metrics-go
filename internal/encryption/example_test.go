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
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	// Сохраняем публичный ключ
	tmpFile, _ := os.CreateTemp("", "example_key_*.pem")
	defer os.Remove(tmpFile.Name())

	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	pem.Encode(tmpFile, publicKeyBlock)
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
	// Output: Encrypted data length: 512 bytes
}
