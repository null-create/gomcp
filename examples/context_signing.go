package examples

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gomcp/context"
	"github.com/gomcp/validate"
)

// generateKey generates a random key of the specified size (bytes).
func generateKey(size int) ([]byte, error) {
	key := make([]byte, size)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func ContextSigning() {
	// --- Key Generation (IMPORTANT: In production, use secure key management!) ---
	// Generate securely random keys ONCE and store/distribute them securely.
	// DO NOT generate keys every time like this example.
	encryptionKey, err := generateKey(validate.AesKeySize) // AES-256 requires 32 bytes
	if err != nil {
		log.Fatalf("Failed to generate encryption key: %v", err)
	}
	signingKey, err := generateKey(validate.HmacKeySize) // E.g., 32 bytes for HMAC-SHA256
	if err != nil {
		log.Fatalf("Failed to generate signing key: %v", err)
	}

	log.Printf("Encryption Key (Base64): %s", base64.StdEncoding.EncodeToString(encryptionKey))
	log.Printf("Signing Key (Base64):    %s", base64.StdEncoding.EncodeToString(signingKey))
	fmt.Println("---")

	// --- Example Data ---
	originalContext := &context.Context{
		ID:        "conv-123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Memory: []*context.MemoryBlock{
			{ID: "mem-1", Role: "user", Content: "Hello there", Time: time.Now().Add(-5 * time.Minute)},
			{ID: "mem-2", Role: "assistant", Content: "Hi! How can I help?", Time: time.Now().Add(-4 * time.Minute)},
		},
		Metadata: map[string]string{"client_id": "webapp-v2", "session_id": "xyz789"},
	}

	// --- Secure the Data ---
	log.Println("Securing original context...")
	securedBytes, err := validate.Secure(originalContext, encryptionKey, signingKey)
	if err != nil {
		log.Fatalf("Failed to secure context: %v", err)
	}
	log.Printf("Secured Payload (Base64): %s\n", base64.StdEncoding.EncodeToString(securedBytes))
	fmt.Println("---")

	// --- Simulate Transport (securedBytes would be sent/received) ---

	// --- Validate and Open the Data ---
	log.Println("Validating and opening received payload...")
	var receivedContext context.Context // Target struct to unmarshal into

	// Pass a POINTER to the target struct
	err = validate.ValidateAndOpen(securedBytes, encryptionKey, signingKey, &receivedContext)
	if err != nil {
		log.Fatalf("Failed to validate and open context: %v", err)
	}

	log.Println("Successfully validated and opened context!")
	log.Printf("Received Context ID: %s", receivedContext.ID)
	log.Printf("Received Metadata: %v", receivedContext.Metadata)
	log.Printf("Received Memory Blocks: %d", len(receivedContext.Memory))
	if len(receivedContext.Memory) > 0 {
		log.Printf(" -> First Memory Block Content: %s", receivedContext.Memory[0].Content)
	}
	fmt.Println("---")

	// --- Example: Tampering Detection ---
	log.Println("Simulating tampering (modifying ciphertext)...")
	var tempPayload validate.SecuredPayload
	_ = json.Unmarshal(securedBytes, &tempPayload) // Decode to modify
	if len(tempPayload.Ciphertext) > 0 {
		tempPayload.Ciphertext[0] = tempPayload.Ciphertext[0] ^ 0xff // Flip first byte
	}
	tamperedBytes, _ := json.Marshal(tempPayload) // Re-encode

	log.Println("Attempting to validate tampered payload...")
	var tamperedContext context.Context
	err = validate.ValidateAndOpen(tamperedBytes, encryptionKey, signingKey, &tamperedContext)
	if err != nil {
		log.Printf("Correctly failed to validate tampered payload: %v", err)
		// Expecting "signature verification failed" OR "decryption failed"
		if errors.Is(err, validate.ErrAuthenticationFailed) || errors.Is(err, validate.ErrDecryptionFailed) {
			log.Println("Tampering detected as expected.")
		} else {
			log.Println("Unexpected error type for tampering.")
		}
	} else {
		log.Fatal("!!! Tampered data was incorrectly validated !!!")
	}
}
