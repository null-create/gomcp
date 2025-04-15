package validate

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrAuthenticationFailed indicates an HMAC signature mismatch.
	ErrAuthenticationFailed = errors.New("message authentication failed")
	// ErrDecryptionFailed indicates a failure during AES-GCM decryption (often due to tampering or wrong key).
	ErrDecryptionFailed = errors.New("message decryption failed")
	// ErrInvalidInput indicates bad input data for validation/opening.
	ErrInvalidInput = errors.New("invalid input data for validation")
	// ErrInvalidKey indicates an incorrect key size.
	ErrInvalidKey = errors.New("invalid key size")
)

const (
	// AES-256 key size
	AesKeySize = 32
	// GCM standard nonce size
	NonceSize = 12
	// Recommended HMAC key size (can vary, but often same as hash output or block size)
	HmacKeySize = 32
)

// SecuredPayload defines the structure for the data during transport.
type SecuredPayload struct {
	Nonce      []byte `json:"n"` // Nonce for AES-GCM (12 bytes)
	Ciphertext []byte `json:"c"` // Encrypted original data (JSON of Context/ContextUpdate)
	Signature  []byte `json:"s"` // HMAC-SHA256 signature of Nonce + Ciphertext
}

// encrypt encrypts plaintext using AES-GCM with the given key.
// It generates a random nonce suitable for GCM.
func encrypt(plaintext []byte, key []byte) (nonce, ciphertext []byte, err error) {
	if len(key) != AesKeySize {
		return nil, nil, fmt.Errorf("%w: expected %d bytes for AES key", ErrInvalidKey, AesKeySize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Never use more than 2^32 random nonces with a given key because of the risk of collisions.
	nonce = make([]byte, NonceSize)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal encrypts and authenticates plaintext. Nonce is unique for key & plaintext.
	// We pass nil for additionalData as GCM's tag already covers the ciphertext.
	// The nonce is returned separately to be stored alongside the ciphertext.
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)

	return nonce, ciphertext, nil
}

// decrypt decrypts ciphertext using AES-GCM with the given key and nonce.
// It also verifies the GCM authenticity tag.
func decrypt(nonce, ciphertext []byte, key []byte) (plaintext []byte, err error) {
	if len(key) != AesKeySize {
		return nil, fmt.Errorf("%w: expected %d bytes for AES key", ErrInvalidKey, AesKeySize)
	}
	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("%w: expected %d bytes for nonce", ErrInvalidInput, NonceSize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Open decrypts and authenticates ciphertext. If the nonce or tag is invalid, it returns an error.
	plaintext, err = gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// This error often means the data was tampered with or the wrong key/nonce was used.
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// signHMAC calculates the HMAC-SHA256 signature for the given data.
func signHMAC(data []byte, key []byte) ([]byte, error) {
	if len(key) == 0 { // Basic check, could enforce key size too
		return nil, fmt.Errorf("%w: HMAC key cannot be empty", ErrInvalidKey)
	}
	mac := hmac.New(sha256.New, key)
	_, err := mac.Write(data)
	if err != nil {
		// Should not happen with standard hash, but check anyway
		return nil, fmt.Errorf("failed to write data to HMAC: %w", err)
	}
	return mac.Sum(nil), nil
}

// verifyHMAC checks if the received signature matches the calculated signature for the data.
// Uses constant-time comparison.
func verifyHMAC(data, receivedSignature []byte, key []byte) error {
	if len(key) == 0 {
		return fmt.Errorf("%w: HMAC key cannot be empty", ErrInvalidKey)
	}
	expectedSignature, err := signHMAC(data, key)
	if err != nil {
		return fmt.Errorf("failed to calculate expected signature: %w", err)
	}

	if !hmac.Equal(receivedSignature, expectedSignature) {
		return ErrAuthenticationFailed
	}
	return nil
}

// Secure marshals the input data, encrypts it, signs the result,
// and packages it into a SecuredPayload, returning the marshalled payload bytes.
// Input 'data' should be a pointer to your Context or ContextUpdate struct.
func Secure(data any, encryptionKey, signingKey []byte) ([]byte, error) {
	// 1. Marshal the original data structure to JSON
	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input data: %w", err)
	}

	// 2. Encrypt the JSON data
	nonce, ciphertext, err := encrypt(plaintext, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// 3. Sign the Nonce + Ciphertext combination
	// Signing both ensures that neither can be replaced independently.
	dataToSign := append([]byte{}, nonce...)
	dataToSign = append(dataToSign, ciphertext...)
	signature, err := signHMAC(dataToSign, signingKey)
	if err != nil {
		return nil, fmt.Errorf("signing failed: %w", err)
	}

	// 4. Create the secured payload structure
	payload := SecuredPayload{
		Nonce:      nonce,
		Ciphertext: ciphertext,
		Signature:  signature,
	}

	// 5. Marshal the secured payload for transport
	securedBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secured payload: %w", err)
	}

	return securedBytes, nil
}

// ValidateAndOpen validates the signature, decrypts the content of the secured payload,
// and unmarshals the original data structure into the 'target' pointer.
// 'securedData' is the raw bytes received from transport (marshalled SecuredPayload).
// 'target' must be a pointer to the expected struct type (e.g., *mcp.Context).
func ValidateAndOpen(securedData []byte, encryptionKey, signingKey []byte, target any) error {
	if len(securedData) == 0 {
		return fmt.Errorf("%w: input securedData cannot be empty", ErrInvalidInput)
	}
	if target == nil {
		return errors.New("target interface cannot be nil")
	}

	// 1. Unmarshal the secured payload structure
	var payload SecuredPayload
	if err := json.Unmarshal(securedData, &payload); err != nil {
		return fmt.Errorf("%w: failed to unmarshal secured payload: %w", ErrInvalidInput, err)
	}

	// Basic checks on payload content
	if payload.Nonce == nil || len(payload.Nonce) != NonceSize || payload.Ciphertext == nil || payload.Signature == nil {
		return fmt.Errorf("%w: incomplete secured payload structure", ErrInvalidInput)
	}

	// 2. Verify the HMAC signature (Nonce + Ciphertext)
	dataToCheck := append([]byte{}, payload.Nonce...)
	dataToCheck = append(dataToCheck, payload.Ciphertext...)
	if err := verifyHMAC(dataToCheck, payload.Signature, signingKey); err != nil {
		// Authentication failed! Do not proceed.
		return fmt.Errorf("signature verification failed: %w", err) // err is ErrAuthenticationFailed
	}

	// --- Signature Verified ---

	// 3. Decrypt the ciphertext
	plaintext, err := decrypt(payload.Nonce, payload.Ciphertext, encryptionKey)
	if err != nil {
		// Decryption or GCM auth tag check failed!
		return fmt.Errorf("decryption failed: %w", err) // err includes ErrDecryptionFailed
	}

	// --- Decryption Successful ---

	// 4. Unmarshal the original JSON data into the target struct
	if err := json.Unmarshal(plaintext, target); err != nil {
		return fmt.Errorf("failed to unmarshal decrypted data into target: %w", err)
	}

	return nil // Success!
}
