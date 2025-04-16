package validate

import (
	"crypto/rand"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test Helpers ---

// mustGenerateKey generates a random key or fails the test.
func mustGenerateKey(t *testing.T, size int) []byte {
	t.Helper() // Marks this as a test helper function
	key := make([]byte, size)
	_, err := rand.Read(key)
	require.NoError(t, err, "Failed to generate random key")
	return key
}

// Simple struct for testing marshalling/unmarshalling
type testPayload struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// --- Low-Level Function Tests ---

func TestEncryptDecrypt(t *testing.T) {
	key := mustGenerateKey(t, AesKeySize)
	plaintext := []byte("this is a secret message")

	t.Run("Success Round Trip", func(t *testing.T) {
		nonce, ciphertext, err := encrypt(plaintext, key)
		require.NoError(t, err)
		require.NotNil(t, nonce)
		require.NotNil(t, ciphertext)
		assert.Len(t, nonce, NonceSize)
		assert.NotEqual(t, plaintext, ciphertext) // Ciphertext shouldn't be plaintext

		decrypted, err := decrypt(nonce, ciphertext, key)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted, "Decrypted text should match original")
	})

	t.Run("Fail Incorrect Key Size Encrypt", func(t *testing.T) {
		badKey := []byte{1, 2, 3}
		_, _, err := encrypt(plaintext, badKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("Fail Incorrect Key Size Decrypt", func(t *testing.T) {
		nonce, ciphertext, err := encrypt(plaintext, key) // Encrypt with good key
		require.NoError(t, err)

		badKey := []byte{1, 2, 3}
		_, err = decrypt(nonce, ciphertext, badKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("Fail Incorrect Nonce Size Decrypt", func(t *testing.T) {
		_, ciphertext, err := encrypt(plaintext, key)
		require.NoError(t, err)

		badNonce := []byte{1, 2, 3} // Too short
		_, err = decrypt(badNonce, ciphertext, key)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidInput) // Error indicates invalid input due to nonce size
	})

	t.Run("Fail Incorrect Key Decrypt", func(t *testing.T) {
		nonce, ciphertext, err := encrypt(plaintext, key)
		require.NoError(t, err)

		wrongKey := mustGenerateKey(t, AesKeySize)
		_, err = decrypt(nonce, ciphertext, wrongKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDecryptionFailed, "Expected decryption failure with wrong key")
	})

	t.Run("Fail Tampered Ciphertext Decrypt", func(t *testing.T) {
		nonce, ciphertext, err := encrypt(plaintext, key)
		require.NoError(t, err)

		// Tamper with ciphertext (GCM includes auth tag at the end)
		if len(ciphertext) > 0 {
			ciphertext[len(ciphertext)-1] ^= 0xff // Flip last byte (part of the tag)
		} else {
			t.Skip("Ciphertext too short to tamper")
		}

		_, err = decrypt(nonce, ciphertext, key)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDecryptionFailed, "Expected decryption failure with tampered ciphertext")
	})
}

func TestSignVerifyHMAC(t *testing.T) {
	key := mustGenerateKey(t, HmacKeySize)
	data := []byte("data to be signed")

	t.Run("Success Round Trip", func(t *testing.T) {
		signature, err := signHMAC(data, key)
		require.NoError(t, err)
		require.NotEmpty(t, signature)

		err = verifyHMAC(data, signature, key)
		assert.NoError(t, err, "Verification should succeed with correct signature and key")
	})

	t.Run("Fail Invalid Signature", func(t *testing.T) {
		signature, err := signHMAC(data, key)
		require.NoError(t, err)

		// Tamper with signature
		if len(signature) > 0 {
			signature[0] ^= 0xff
		} else {
			t.Skip("Signature too short to tamper")
		}

		err = verifyHMAC(data, signature, key)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthenticationFailed, "Verification should fail with bad signature")
	})

	t.Run("Fail Invalid Key", func(t *testing.T) {
		signature, err := signHMAC(data, key)
		require.NoError(t, err)

		wrongKey := mustGenerateKey(t, HmacKeySize)
		err = verifyHMAC(data, signature, wrongKey)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthenticationFailed, "Verification should fail with wrong key")
	})

	t.Run("Fail Tampered Data", func(t *testing.T) {
		signature, err := signHMAC(data, key)
		require.NoError(t, err)

		tamperedData := append([]byte{}, data...)
		if len(tamperedData) > 0 {
			tamperedData[0] ^= 0xff
		} else {
			t.Skip("Data too short to tamper")
		}

		err = verifyHMAC(tamperedData, signature, key) // Verify original sig against tampered data
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthenticationFailed, "Verification should fail with tampered data")
	})

	t.Run("Fail Empty Key Sign", func(t *testing.T) {
		_, err := signHMAC(data, []byte{})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("Fail Empty Key Verify", func(t *testing.T) {
		signature, err := signHMAC(data, key) // Sign with good key
		require.NoError(t, err)
		err = verifyHMAC(data, signature, []byte{}) // Verify with empty key
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKey) // Check underlying error from verify trying to sign
	})
}

// --- High-Level API Tests ---

func TestSecureAndValidateOpen(t *testing.T) {
	encKey := mustGenerateKey(t, AesKeySize)
	signKey := mustGenerateKey(t, HmacKeySize)
	originalData := testPayload{Name: "Alice", Age: 30}

	t.Run("Success Round Trip", func(t *testing.T) {
		securedBytes, err := Secure(&originalData, encKey, signKey)
		require.NoError(t, err)
		require.NotEmpty(t, securedBytes)

		// Verify it's likely JSON
		var temp map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(securedBytes, &temp), "Secured data should be valid JSON")
		assert.Contains(t, temp, "n", "JSON should contain nonce")
		assert.Contains(t, temp, "c", "JSON should contain ciphertext")
		assert.Contains(t, temp, "s", "JSON should contain signature")

		var recoveredData testPayload
		err = ValidateAndOpen(securedBytes, encKey, signKey, &recoveredData) // Pass pointer
		require.NoError(t, err, "ValidateAndOpen failed on valid data")

		assert.Equal(t, originalData, recoveredData, "Recovered data does not match original")
	})

	t.Run("Fail Wrong Encryption Key", func(t *testing.T) {
		securedBytes, err := Secure(&originalData, encKey, signKey)
		require.NoError(t, err)

		wrongEncKey := mustGenerateKey(t, AesKeySize)
		var recoveredData testPayload
		err = ValidateAndOpen(securedBytes, wrongEncKey, signKey, &recoveredData) // Correct sign key, wrong enc key
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDecryptionFailed, "Expected decryption failure with wrong encryption key")
	})

	t.Run("Fail Wrong Signing Key", func(t *testing.T) {
		securedBytes, err := Secure(&originalData, encKey, signKey)
		require.NoError(t, err)

		wrongSignKey := mustGenerateKey(t, HmacKeySize)
		var recoveredData testPayload
		err = ValidateAndOpen(securedBytes, encKey, wrongSignKey, &recoveredData) // Correct enc key, wrong sign key
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthenticationFailed, "Expected authentication failure with wrong signing key")
	})

	t.Run("Fail Tampered Ciphertext", func(t *testing.T) {
		securedBytes, err := Secure(&originalData, encKey, signKey)
		require.NoError(t, err)

		var payload SecuredPayload
		require.NoError(t, json.Unmarshal(securedBytes, &payload))

		// Tamper ciphertext
		if len(payload.Ciphertext) > 0 {
			payload.Ciphertext[0] ^= 0xff
		} else {
			t.Skip("Ciphertext too short to tamper")
		}

		tamperedSecuredBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		var recoveredData testPayload
		err = ValidateAndOpen(tamperedSecuredBytes, encKey, signKey, &recoveredData)
		require.Error(t, err)
		// Since Nonce+Ciphertext is signed, tampering Ciphertext should cause signature failure
		assert.ErrorIs(t, err, ErrAuthenticationFailed, "Expected authentication failure with tampered ciphertext")
	})

	t.Run("Fail Tampered Nonce", func(t *testing.T) {
		securedBytes, err := Secure(&originalData, encKey, signKey)
		require.NoError(t, err)

		var payload SecuredPayload
		require.NoError(t, json.Unmarshal(securedBytes, &payload))

		// Tamper nonce
		if len(payload.Nonce) > 0 {
			payload.Nonce[0] ^= 0xff
		} else {
			t.Skip("Nonce too short to tamper")
		}

		tamperedSecuredBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		var recoveredData testPayload
		err = ValidateAndOpen(tamperedSecuredBytes, encKey, signKey, &recoveredData)
		require.Error(t, err)
		// Since Nonce+Ciphertext is signed, tampering Nonce should cause signature failure
		assert.ErrorIs(t, err, ErrAuthenticationFailed, "Expected authentication failure with tampered nonce")
	})

	t.Run("Fail Tampered Signature", func(t *testing.T) {
		securedBytes, err := Secure(&originalData, encKey, signKey)
		require.NoError(t, err)

		var payload SecuredPayload
		require.NoError(t, json.Unmarshal(securedBytes, &payload))

		// Tamper signature
		if len(payload.Signature) > 0 {
			payload.Signature[0] ^= 0xff
		} else {
			t.Skip("Signature too short to tamper")
		}

		tamperedSecuredBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		var recoveredData testPayload
		err = ValidateAndOpen(tamperedSecuredBytes, encKey, signKey, &recoveredData)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthenticationFailed, "Expected authentication failure with tampered signature")
	})

	t.Run("Fail ValidateAndOpen Empty Input", func(t *testing.T) {
		var recoveredData testPayload
		err := ValidateAndOpen([]byte{}, encKey, signKey, &recoveredData)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("Fail ValidateAndOpen Nil Input", func(t *testing.T) {
		var recoveredData testPayload
		err := ValidateAndOpen(nil, encKey, signKey, &recoveredData)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("Fail ValidateAndOpen Nil Target", func(t *testing.T) {
		securedBytes, err := Secure(&originalData, encKey, signKey)
		require.NoError(t, err)
		err = ValidateAndOpen(securedBytes, encKey, signKey, nil) // Pass nil target
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target interface cannot be nil")
	})

	t.Run("Fail ValidateAndOpen Malformed JSON Payload", func(t *testing.T) {
		malformedBytes := []byte(`{"n":"bad", "c":"json"`) // Invalid JSON
		var recoveredData testPayload
		err := ValidateAndOpen(malformedBytes, encKey, signKey, &recoveredData)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidInput) // Should fail unmarshalling the payload struct
		assert.Contains(t, err.Error(), "failed to unmarshal secured payload")
	})

	t.Run("Fail ValidateAndOpen Incomplete Payload (Missing Nonce)", func(t *testing.T) {
		// Create a valid payload, then remove a field after marshalling
		securedBytes, err := Secure(&originalData, encKey, signKey)
		require.NoError(t, err)
		var temp map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(securedBytes, &temp))
		delete(temp, "n") // Remove nonce field
		incompleteBytes, err := json.Marshal(temp)
		require.NoError(t, err)

		var recoveredData testPayload
		err = ValidateAndOpen(incompleteBytes, encKey, signKey, &recoveredData)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidInput)
		assert.Contains(t, err.Error(), "incomplete secured payload structure")
	})

	t.Run("Fail ValidateAndOpen Wrong Nonce Size in Payload", func(t *testing.T) {
		// Create a valid payload, then modify nonce size
		securedBytes, err := Secure(&originalData, encKey, signKey)
		require.NoError(t, err)
		var payload SecuredPayload
		require.NoError(t, json.Unmarshal(securedBytes, &payload))

		payload.Nonce = []byte{1, 2, 3} // Set invalid size nonce
		badNonceSizeBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		var recoveredData testPayload
		err = ValidateAndOpen(badNonceSizeBytes, encKey, signKey, &recoveredData)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidInput)
		assert.Contains(t, err.Error(), "incomplete secured payload structure") // Caught by size check
	})

	t.Run("Fail Secure with Bad Key Size", func(t *testing.T) {
		badKey := []byte{1, 2, 3}
		_, err := Secure(&originalData, badKey, signKey) // Bad Enc key
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKey)

		_, err = Secure(&originalData, encKey, []byte{}) // Bad Sign key (empty)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("Fail Secure with Unmarshallable Data", func(t *testing.T) {
		// Channels cannot be marshalled to JSON
		unmarshallableData := make(chan int)
		_, err := Secure(unmarshallableData, encKey, signKey)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal input data")
	})
}
