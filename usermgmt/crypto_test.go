package usermgmt

import (
	"testing"
)

func TestGenerateToken_ReturnsNonEmpty(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
	if len(token) < 32 {
		t.Errorf("token too short: %d chars", len(token))
	}
}

func TestGenerateToken_Uniqueness(t *testing.T) {
	seen := make(map[string]bool, 100)
	for range 100 {
		token, err := GenerateToken()
		if err != nil {
			t.Fatalf("GenerateToken: %v", err)
		}
		if seen[token] {
			t.Fatal("duplicate token generated")
		}
		seen[token] = true
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	pepper := []byte("test-pepper-32-bytes-long-xxxxxxx")
	token := "my-secret-api-token"

	hash1 := HashToken(token, pepper)
	hash2 := HashToken(token, pepper)

	if len(hash1) != 32 {
		t.Errorf("expected 32-byte hash, got %d", len(hash1))
	}
	if string(hash1) != string(hash2) {
		t.Error("hash is not deterministic")
	}
}

func TestHashToken_DifferentPeppersProduceDifferentHashes(t *testing.T) {
	token := "my-secret-api-token"
	hash1 := HashToken(token, []byte("pepper-A"))
	hash2 := HashToken(token, []byte("pepper-B"))
	if string(hash1) == string(hash2) {
		t.Error("different peppers should produce different hashes")
	}
}

func TestVerifyToken_RoundTrip(t *testing.T) {
	pepper := []byte("test-pepper-32-bytes-long-xxxxxxx")
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	hash := HashToken(token, pepper)
	if !VerifyToken(token, hash, pepper) {
		t.Error("VerifyToken should return true for correct token+pepper")
	}
}

func TestVerifyToken_WrongToken(t *testing.T) {
	pepper := []byte("test-pepper-32-bytes-long-xxxxxxx")
	token, _ := GenerateToken()
	hash := HashToken(token, pepper)
	wrongToken := "definitely-wrong-token"
	if VerifyToken(wrongToken, hash, pepper) {
		t.Error("VerifyToken should return false for wrong token")
	}
}

func TestVerifyToken_WrongPepper(t *testing.T) {
	pepper := []byte("test-pepper-32-bytes-long-xxxxxxx")
	token, _ := GenerateToken()
	hash := HashToken(token, pepper)
	wrongPepper := []byte("wrong-pepper-32-bytes-long-xxxxxxx")
	if VerifyToken(token, hash, wrongPepper) {
		t.Error("VerifyToken should return false for wrong pepper")
	}
}
