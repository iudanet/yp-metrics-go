package encryption

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

func TestHybridDecryption(t *testing.T) {
	// Создаем временную RSA пару ключей для тестов
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Сохраняем приватный ключ во временный файл
	tmpPrivateFile, err := os.CreateTemp("", "test_private_key_*.pem")
	require.NoError(t, err)
	defer os.Remove(tmpPrivateFile.Name())

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	err = pem.Encode(tmpPrivateFile, privateKeyBlock)
	require.NoError(t, err)
	tmpPrivateFile.Close()

	// Сохраняем публичный ключ во временный файл для шифрования
	tmpPublicFile, err := os.CreateTemp("", "test_public_key_*.pem")
	require.NoError(t, err)
	defer os.Remove(tmpPublicFile.Name())

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	err = pem.Encode(tmpPublicFile, publicKeyBlock)
	require.NoError(t, err)
	tmpPublicFile.Close()

	testData := []byte(`{"id":"test","type":"gauge","value":42.5}`)

	t.Run("successful encryption and decryption", func(t *testing.T) {
		// Шифруем данные
		encryptedData, err := Hybrid(testData, tmpPublicFile.Name())
		require.NoError(t, err)
		assert.NotEmpty(t, encryptedData)
		assert.NotEqual(t, testData, encryptedData)

		// Расшифровываем данные
		decryptedData, err := HybridDecrypt(encryptedData, tmpPrivateFile.Name())
		require.NoError(t, err)
		assert.Equal(t, testData, decryptedData)
	})

	t.Run("decryption with invalid key", func(t *testing.T) {
		// Шифруем данные
		encryptedData, err := Hybrid(testData, tmpPublicFile.Name())
		require.NoError(t, err)

		// Создаем другой ключ для попытки расшифровки
		wrongPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		wrongTmpFile, err := os.CreateTemp("", "wrong_private_key_*.pem")
		require.NoError(t, err)
		defer os.Remove(wrongTmpFile.Name())

		wrongKeyBytes := x509.MarshalPKCS1PrivateKey(wrongPrivateKey)
		wrongKeyBlock := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: wrongKeyBytes,
		}

		err = pem.Encode(wrongTmpFile, wrongKeyBlock)
		require.NoError(t, err)
		wrongTmpFile.Close()

		// Попытка расшифровки неправильным ключом
		_, err = HybridDecrypt(encryptedData, wrongTmpFile.Name())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "RSA decryption of AES key failed")
	})

	t.Run("decryption with tampered data", func(t *testing.T) {
		// Шифруем данные
		encryptedData, err := Hybrid(testData, tmpPublicFile.Name())
		require.NoError(t, err)

		// Повреждаем зашифрованные данные
		tamperedData := make([]byte, len(encryptedData))
		copy(tamperedData, encryptedData)
		if len(tamperedData) > 100 {
			tamperedData[50] ^= 0xFF // Изменяем байт в середине
		}

		// Попытка расшифровки поврежденных данных
		_, err = HybridDecrypt(tamperedData, tmpPrivateFile.Name())
		require.Error(t, err)
	})

	t.Run("decryption with empty private key path", func(t *testing.T) {
		encryptedData, err := Hybrid(testData, tmpPublicFile.Name())
		require.NoError(t, err)

		_, err = HybridDecrypt(encryptedData, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no private key path configured")
	})

	t.Run("decryption with invalid encrypted packet", func(t *testing.T) {
		invalidData := []byte(`{"invalid": "data"}`)
		_, err := HybridDecrypt(invalidData, tmpPrivateFile.Name())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'key' field")
	})
}
func TestLoadRSAPrivateKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("load valid PKCS1 private key", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "pkcs1_private_*.pem")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
		privateKeyBlock := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBytes,
		}

		err = pem.Encode(tmpFile, privateKeyBlock)
		require.NoError(t, err)
		tmpFile.Close()

		loadedKey, err := loadRSAPrivateKey(tmpFile.Name())
		require.NoError(t, err)
		assert.NotNil(t, loadedKey)
	})

	t.Run("load valid PKCS8 private key", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "pkcs8_private_*.pem")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
		require.NoError(t, err)

		privateKeyBlock := &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateKeyBytes,
		}

		err = pem.Encode(tmpFile, privateKeyBlock)
		require.NoError(t, err)
		tmpFile.Close()

		loadedKey, err := loadRSAPrivateKey(tmpFile.Name())
		require.NoError(t, err)
		assert.NotNil(t, loadedKey)
	})

	t.Run("load non-RSA private key", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "invalid_private_*.pem")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte("-----BEGIN PRIVATE KEY-----\ninvalid\n-----END PRIVATE KEY-----"))
		require.NoError(t, err)
		tmpFile.Close()

		_, err = loadRSAPrivateKey(tmpFile.Name())
		require.Error(t, err)
	})

	t.Run("load non-existent file", func(t *testing.T) {
		_, err := loadRSAPrivateKey("/non/existent/file.pem")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read private key file")
	})
}

func TestDecryptAESGCM(t *testing.T) {
	testData := []byte("test data for AES-GCM decryption")
	aesKey := make([]byte, 32)
	rand.Read(aesKey)

	t.Run("successful encryption and decryption", func(t *testing.T) {
		nonce, ciphertext, err := encryptAESGCM(aesKey, testData)
		require.NoError(t, err)

		plaintext, err := decryptAESGCM(aesKey, nonce, ciphertext)
		require.NoError(t, err)
		assert.Equal(t, testData, plaintext)
	})

	t.Run("decryption with wrong key", func(t *testing.T) {
		nonce, ciphertext, err := encryptAESGCM(aesKey, testData)
		require.NoError(t, err)

		wrongKey := make([]byte, 32)
		rand.Read(wrongKey)

		_, err = decryptAESGCM(wrongKey, nonce, ciphertext)
		require.Error(t, err)
	})

	t.Run("decryption with wrong nonce", func(t *testing.T) {
		nonce, ciphertext, err := encryptAESGCM(aesKey, testData)
		require.NoError(t, err)

		wrongNonce := make([]byte, len(nonce))
		rand.Read(wrongNonce)

		_, err = decryptAESGCM(aesKey, wrongNonce, ciphertext)
		require.Error(t, err)
	})

	t.Run("decryption with tampered ciphertext", func(t *testing.T) {
		nonce, ciphertext, err := encryptAESGCM(aesKey, testData)
		require.NoError(t, err)

		// Повреждаем ciphertext
		if len(ciphertext) > 10 {
			ciphertext[5] ^= 0xFF
		}

		_, err = decryptAESGCM(aesKey, nonce, ciphertext)
		require.Error(t, err)
	})
}
