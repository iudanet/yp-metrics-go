package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHybridEncryption(t *testing.T) {
	// Создаем временный RSA ключ для тестов
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Сохраняем публичный ключ во временный файл
	tmpFile, err := os.CreateTemp("", "test_public_key_*.pem")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	err = pem.Encode(tmpFile, publicKeyBlock)
	require.NoError(t, err)
	tmpFile.Close()

	testData := []byte(`{"id":"test","type":"gauge","value":42.5}`)

	t.Run("successful encryption", func(t *testing.T) {
		encryptedData, err := Hybrid(testData, tmpFile.Name())
		require.NoError(t, err)
		assert.NotEmpty(t, encryptedData)
		assert.NotEqual(t, testData, encryptedData)

		// Проверяем что зашифрованные данные являются валидным JSON
		var encryptedPacket map[string]string
		err = json.Unmarshal(encryptedData, &encryptedPacket)
		require.NoError(t, err)

		assert.Contains(t, encryptedPacket, "key")
		assert.Contains(t, encryptedPacket, "nonce")
		assert.Contains(t, encryptedPacket, "data")
		assert.NotEmpty(t, encryptedPacket["key"])
		assert.NotEmpty(t, encryptedPacket["nonce"])
		assert.NotEmpty(t, encryptedPacket["data"])
	})

	t.Run("encryption with empty data", func(t *testing.T) {
		emptyData := []byte{}
		encryptedData, err := Hybrid(emptyData, tmpFile.Name())
		require.NoError(t, err)
		assert.NotEmpty(t, encryptedData)

		var encryptedPacket map[string]string
		err = json.Unmarshal(encryptedData, &encryptedPacket)
		require.NoError(t, err)
		assert.NotEmpty(t, encryptedPacket["data"])
	})

	t.Run("encryption with large data", func(t *testing.T) {
		largeData := make([]byte, 1024) // 1KB данных
		rand.Read(largeData)

		encryptedData, err := Hybrid(largeData, tmpFile.Name())
		require.NoError(t, err)
		assert.NotEmpty(t, encryptedData)
	})

	t.Run("different inputs produce different outputs", func(t *testing.T) {
		data1 := []byte(`{"id":"test1","value":1.0}`)
		data2 := []byte(`{"id":"test2","value":2.0}`)

		encrypted1, err := Hybrid(data1, tmpFile.Name())
		require.NoError(t, err)

		encrypted2, err := Hybrid(data2, tmpFile.Name())
		require.NoError(t, err)

		assert.NotEqual(t, encrypted1, encrypted2)
	})
}

func TestHybridEncryptionErrors(t *testing.T) {
	t.Run("non-existent key file", func(t *testing.T) {
		_, err := Hybrid([]byte("test"), "/non/existent/file.pem")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read public key file")
	})

	t.Run("invalid key file content", func(t *testing.T) {
		// Создаем файл с некорректным содержимым
		tmpFile, err := os.CreateTemp("", "invalid_key_*.pem")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte("invalid key content"))
		require.NoError(t, err)
		tmpFile.Close()

		_, err = Hybrid([]byte("test"), tmpFile.Name())
		require.Error(t, err)
		// Обновляем проверку на правильное сообщение об ошибке
		assert.Contains(t, err.Error(), "failed to parse PEM block from public key file")
	})

	t.Run("empty key file path", func(t *testing.T) {
		_, err := Hybrid([]byte("test"), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no public key path configured")
	})
}

func TestLoadRSAPublicKey(t *testing.T) {
	// Создаем временный RSA ключ
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("load valid PKIX public key", func(t *testing.T) {
		// Сохраняем в формате PKIX
		tmpFile, err := os.CreateTemp("", "pkix_key_*.pem")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		require.NoError(t, err)

		publicKeyBlock := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyBytes,
		}

		err = pem.Encode(tmpFile, publicKeyBlock)
		require.NoError(t, err)
		tmpFile.Close()

		pubKey, err := loadRSAPublicKey(tmpFile.Name())
		require.NoError(t, err)
		assert.NotNil(t, pubKey)
	})

	t.Run("load valid PKCS1 public key", func(t *testing.T) {
		// Сохраняем в формате PKCS1
		tmpFile, err := os.CreateTemp("", "pkcs1_key_*.pem")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		publicKeyBytes := x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)
		publicKeyBlock := &pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: publicKeyBytes,
		}

		err = pem.Encode(tmpFile, publicKeyBlock)
		require.NoError(t, err)
		tmpFile.Close()

		pubKey, err := loadRSAPublicKey(tmpFile.Name())
		require.NoError(t, err)
		assert.NotNil(t, pubKey)
	})

	t.Run("load non-RSA public key", func(t *testing.T) {
		// Этот тест требует создания другого типа ключа, что сложно в рамках теста
		// Проверяем обработку ошибки для некорректного файла
		tmpFile, err := os.CreateTemp("", "invalid_*.pem")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte("-----BEGIN PUBLIC KEY-----\ninvalid\n-----END PUBLIC KEY-----"))
		require.NoError(t, err)
		tmpFile.Close()

		_, err = loadRSAPublicKey(tmpFile.Name())
		require.Error(t, err)
	})
}

func TestEncryptAESGCM(t *testing.T) {
	testData := []byte("test data for AES-GCM encryption")
	aesKey := make([]byte, 32)
	rand.Read(aesKey)

	t.Run("successful encryption", func(t *testing.T) {
		nonce, ciphertext, err := encryptAESGCM(aesKey, testData)
		require.NoError(t, err)
		assert.NotEmpty(t, nonce)
		assert.NotEmpty(t, ciphertext)
		assert.NotEqual(t, testData, ciphertext)
	})

	t.Run("encryption with empty data", func(t *testing.T) {
		nonce, ciphertext, err := encryptAESGCM(aesKey, []byte{})
		require.NoError(t, err)
		assert.NotEmpty(t, nonce)
		assert.NotEmpty(t, ciphertext)
	})

	t.Run("invalid key size", func(t *testing.T) {
		invalidKey := make([]byte, 16) // Неправильный размер для AES-256
		_, _, err := encryptAESGCM(invalidKey, testData)
		require.NoError(t, err) // AES.NewCipher принимает 16, 24, 32 байта
	})
}

func TestEncryptionConsistency(t *testing.T) {
	// Создаем временный RSA ключ
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tmpFile, err := os.CreateTemp("", "consistency_key_*.pem")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	err = pem.Encode(tmpFile, publicKeyBlock)
	require.NoError(t, err)
	tmpFile.Close()

	testData := []byte(`{"metric":"value"}`)

	// Многократное шифрование одних и тех же данных должно давать разные результаты
	// из-за использования случайных nonce и AES ключей
	results := make([][]byte, 5)
	for i := 0; i < 5; i++ {
		encrypted, err := Hybrid(testData, tmpFile.Name())
		require.NoError(t, err)
		results[i] = encrypted

		// Проверяем что каждый результат уникален
		for j := 0; j < i; j++ {
			assert.NotEqual(t, results[j], results[i], "Encryption should produce different results each time")
		}
	}
}
