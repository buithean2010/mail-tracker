package crypto_test

import (
	"strings"
	"testing"

	"github.com/buithean2010/mail-tracker/backend/internal/infra/crypto"
)

const testKey = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

func TestNewAES_ValidKey(t *testing.T) {
	enc, err := crypto.NewAES(testKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enc == nil {
		t.Fatal("expected non-nil encryptor")
	}
}

func TestNewAES_InvalidHex(t *testing.T) {
	_, err := crypto.NewAES("not-hex-at-all!!")
	if err == nil {
		t.Error("expected error for invalid hex key")
	}
}

func TestNewAES_WrongKeyLength(t *testing.T) {
	// 16 bytes = 32 hex chars — too short for AES-256
	_, err := crypto.NewAES("0102030405060708090a0b0c0d0e0f10")
	if err == nil {
		t.Error("expected error for 16-byte key, want 32")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	enc, _ := crypto.NewAES(testKey)
	plaintext := "super-secret-api-key-12345"

	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if ciphertext == plaintext {
		t.Error("ciphertext should not equal plaintext")
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("Decrypt() = %q, want %q", decrypted, plaintext)
	}
}

func TestEncrypt_Nondeterministic(t *testing.T) {
	enc, _ := crypto.NewAES(testKey)
	a, _ := enc.Encrypt("hello")
	b, _ := enc.Encrypt("hello")
	if a == b {
		t.Error("two encryptions of the same plaintext should produce different ciphertexts (random nonce)")
	}
}

func TestDecrypt_InvalidHex(t *testing.T) {
	enc, _ := crypto.NewAES(testKey)
	_, err := enc.Decrypt("not-valid-hex!!")
	if err == nil {
		t.Error("expected error for non-hex ciphertext")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	enc, _ := crypto.NewAES(testKey)
	_, err := enc.Decrypt("aabb")
	if err == nil {
		t.Error("expected error for ciphertext shorter than nonce")
	}
}

func TestDecrypt_Tampered(t *testing.T) {
	enc, _ := crypto.NewAES(testKey)
	ciphertext, _ := enc.Encrypt("hello")
	// Flip the last byte.
	tampered := ciphertext[:len(ciphertext)-2] + "ff"
	_, err := enc.Decrypt(tampered)
	if err == nil {
		t.Error("expected decryption to fail on tampered ciphertext")
	}
}

func TestEncryptDecrypt_EmptyString(t *testing.T) {
	enc, _ := crypto.NewAES(testKey)
	ct, err := enc.Encrypt("")
	if err != nil {
		t.Fatal(err)
	}
	pt, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatal(err)
	}
	if pt != "" {
		t.Errorf("expected empty string, got %q", pt)
	}
}

func TestEncryptDecrypt_LongString(t *testing.T) {
	enc, _ := crypto.NewAES(testKey)
	long := strings.Repeat("a", 10_000)
	ct, err := enc.Encrypt(long)
	if err != nil {
		t.Fatal(err)
	}
	pt, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatal(err)
	}
	if pt != long {
		t.Error("round-trip failed for long string")
	}
}
