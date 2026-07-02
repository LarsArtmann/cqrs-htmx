package webauthn

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-webauthn/webauthn/webauthn"
)

// W3C WebAuthn Level 3 spec test vectors for the "none" attestation format
// with ES256 (ECDSA P-256). These are the official conformance test vectors
// published by the W3C and also used by go-webauthn's own test suite.
//
// See: https://www.w3.org/TR/webauthn-3/#sctn-test-vectors
const (
	regAttestationObjectHex = "a363666d74646e6f6e656761747453746d74a068617574684461746158" +
		"a4bfabc37432958b063360d3ad6461c9c4735ae7f8edd46592a5e0f01452b2e4b5590" +
		"00000008446ccb9ab1db374750b2367ff6f3a1f0020f91f391db4c9b2fde0ea70189c" +
		"ba3fb63f579ba6122b33ad94ff3ec330084be4a5010203262001215820afefa16f97c" +
		"a9b2d23eb86ccb64098d20db90856062eb249c33a9b672f26df61225820930a56b87a" +
		"2fca66334b03458abf879717c12cc68ed73290af2e2664796b9220"
	regClientDataJSONHex = "7b2274797065223a22776562617574686e2e637265617465222c22636861" +
		"6c6c656e6765223a22414d4d507434557878475453746e6364713431375944774246" +
		"6938767049612d7077386f4f755657345441222c226f726967696e223a2268747470" +
		"733a2f2f6578616d706c652e6f7267222c2263726f73734f726967696e223a66616c" +
		"73652c22657874726144617461223a22636c69656e74446174614a534f4e206d6179" +
		"20626520657874656e6465642077697468206164646974696f6e616c206669656c64" +
		"7320696e20746865206675747572652c207375636820617320746869733a20426b51" +
		"65446a646354427258426941774a544c453551227d"
	regCredentialIDHex = "f91f391db4c9b2fde0ea70189cba3fb63f579ba6122b33ad94ff3ec330084be4"
	regChallengeHex    = "00c30fb78531c464d2b6771dab8d7b603c01162f2fa486bea70f283ae556e130"

	loginAuthenticatorDataHex = "bfabc37432958b063360d3ad6461c9c4735ae7f8edd46592a5e0f01452b2e4b51900000000"
	loginClientDataJSONHex    = "7b2274797065223a22776562617574686e2e676574222c226368616c6c656e6765223a224f63446e55685158756c5455506f334a5558543049393770767a7a59425039745a63685879617630314167222c226f726967696e223a2268747470733a2f2f6578616d706c652e6f7267222c2263726f73734f726967696e223a66616c73657d"
	loginSignatureHex         = "3046022100f50a4e2e4409249c4a853ba361282f09841df4dd4547a13a87780218deffcd380221008480ac0f0b93538174f575bf11a1dd5d78c6e486013f937295ea13653e331e87"
	loginChallengeHex         = "39c0e7521417ba54d43e8dc95174f423dee9bf3cd804ff6d65c857c9abf4d408"
)

func testProvider(t *testing.T) *Provider {
	t.Helper()
	p, err := New(Config{
		RPID:          "example.org",
		RPDisplayName: "Test App",
		RPOrigins:     []string{"https://example.org"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return p
}

func testUserJSON(t *testing.T, creds ...credentialData) []byte {
	t.Helper()
	data, err := json.Marshal(userData{
		ID:          "user-123",
		Email:       "user@example.org",
		DisplayName: "Test User",
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("marshal userJSON: %v", err)
	}
	return data
}

func overrideChallenge(t *testing.T, sessionData []byte, challengeHex string) []byte {
	t.Helper()
	var session webauthn.SessionData
	if err := json.Unmarshal(sessionData, &session); err != nil {
		t.Fatalf("unmarshal session: %v", err)
	}
	session.Challenge = base64.RawURLEncoding.EncodeToString(decodeHex(t, challengeHex))
	result, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	return result
}

func buildRegistrationBody(t *testing.T) []byte {
	t.Helper()
	credentialID := decodeHex(t, regCredentialIDHex)
	attObj := base64.RawURLEncoding.EncodeToString(decodeHex(t, regAttestationObjectHex))
	cdj := base64.RawURLEncoding.EncodeToString(decodeHex(t, regClientDataJSONHex))
	id := base64.RawURLEncoding.EncodeToString(credentialID)

	body := map[string]any{
		"id":    id,
		"rawId": id,
		"type":  "public-key",
		"response": map[string]any{
			"attestationObject": attObj,
			"clientDataJSON":    cdj,
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal registration body: %v", err)
	}
	return data
}

func buildLoginBody(t *testing.T) []byte {
	t.Helper()
	credentialID := decodeHex(t, regCredentialIDHex)
	authData := base64.RawURLEncoding.EncodeToString(decodeHex(t, loginAuthenticatorDataHex))
	cdj := base64.RawURLEncoding.EncodeToString(decodeHex(t, loginClientDataJSONHex))
	sig := base64.RawURLEncoding.EncodeToString(decodeHex(t, loginSignatureHex))
	id := base64.RawURLEncoding.EncodeToString(credentialID)

	body := map[string]any{
		"id":    id,
		"rawId": id,
		"type":  "public-key",
		"response": map[string]any{
			"authenticatorData": authData,
			"clientDataJSON":    cdj,
			"signature":         sig,
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}
	return data
}

func decodeHex(t *testing.T, s string) []byte {
	t.Helper()
	data, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	return data
}

// --- Tests ---

func TestNew_InvalidConfig(t *testing.T) {
	_, err := New(Config{}) // empty RPID
	if err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestNew_ValidConfig(t *testing.T) {
	p := testProvider(t)
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestProvider_BeginRegistration(t *testing.T) {
	p := testProvider(t)
	ctx := context.Background()
	userJSON := testUserJSON(t)

	options, sessionData, err := p.BeginRegistration(ctx, userJSON)
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}
	if len(options) == 0 {
		t.Error("expected non-empty options")
	}
	if len(sessionData) == 0 {
		t.Error("expected non-empty sessionData")
	}

	// Verify options are valid JSON with expected fields
	var opts map[string]any
	if err := json.Unmarshal(options, &opts); err != nil {
		t.Fatalf("options are not valid JSON: %v", err)
	}
	if opts["publicKey"] == nil {
		t.Error("expected publicKey in options")
	}

	// Verify sessionData is a valid SessionData
	var session webauthn.SessionData
	if err := json.Unmarshal(sessionData, &session); err != nil {
		t.Fatalf("sessionData is not valid SessionData JSON: %v", err)
	}
	if session.Challenge == "" {
		t.Error("expected non-empty challenge in session")
	}
}

func TestProvider_BeginRegistration_InvalidUserJSON(t *testing.T) {
	p := testProvider(t)
	ctx := context.Background()

	_, _, err := p.BeginRegistration(ctx, []byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "unmarshal user data") {
		t.Errorf("expected unmarshal error, got: %v", err)
	}
}

func TestProvider_FinishRegistration_InvalidSessionData(t *testing.T) {
	p := testProvider(t)
	ctx := context.Background()
	userJSON := testUserJSON(t)

	_, err := p.FinishRegistration(ctx, userJSON, []byte("{}"), []byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid session JSON")
	}
	if !strings.Contains(err.Error(), "unmarshal session data") {
		t.Errorf("expected session unmarshal error, got: %v", err)
	}
}

func TestProvider_FullCeremony(t *testing.T) {
	p := testProvider(t)
	ctx := context.Background()
	userJSON := testUserJSON(t)

	// --- Registration ---

	options, sessionData, err := p.BeginRegistration(ctx, userJSON)
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}
	_ = options

	// Override challenge to match W3C spec vector
	overriddenSession := overrideChallenge(t, sessionData, regChallengeHex)

	// Finish registration with spec vector attestation
	regBody := buildRegistrationBody(t)
	credJSON, err := p.FinishRegistration(ctx, userJSON, regBody, overriddenSession)
	if err != nil {
		t.Fatalf("FinishRegistration: %v", err)
	}

	// Verify credential
	var cred credentialData
	if err := json.Unmarshal(credJSON, &cred); err != nil {
		t.Fatalf("unmarshal credential: %v", err)
	}
	if len(cred.ID) == 0 {
		t.Error("expected non-empty credential ID")
	}
	if len(cred.PublicKey) == 0 {
		t.Error("expected non-empty public key")
	}
	if cred.AttestationType != "none" {
		t.Errorf("attestation type = %q, want %q", cred.AttestationType, "none")
	}

	// --- Login ---

	// Build userJSON WITH the registered credential
	loginUserJSON := testUserJSON(t, cred)

	_, loginSession, err := p.BeginLogin(ctx, loginUserJSON)
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}

	// Override login challenge to match W3C spec vector
	overriddenLoginSession := overrideChallenge(t, loginSession, loginChallengeHex)

	// Finish login with spec vector assertion
	loginBody := buildLoginBody(t)
	err = p.FinishLogin(ctx, loginUserJSON, loginBody, overriddenLoginSession)
	if err != nil {
		t.Fatalf("FinishLogin: %v", err)
	}
}

func TestProvider_FinishRegistration_WrongChallenge(t *testing.T) {
	p := testProvider(t)
	ctx := context.Background()
	userJSON := testUserJSON(t)

	_, sessionData, err := p.BeginRegistration(ctx, userJSON)
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}

	// Do NOT override the challenge — the W3C vector challenge won't match
	regBody := buildRegistrationBody(t)
	_, err = p.FinishRegistration(ctx, userJSON, regBody, sessionData)
	if err == nil {
		t.Fatal("expected error with wrong challenge")
	}
}

func TestProvider_FinishRegistration_ExpiredSession(t *testing.T) {
	p := testProvider(t)
	ctx := context.Background()
	userJSON := testUserJSON(t)

	_, sessionData, err := p.BeginRegistration(ctx, userJSON)
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}

	// Override challenge AND set expiry to the past
	overridden := overrideChallenge(t, sessionData, regChallengeHex)
	var session webauthn.SessionData
	if err := json.Unmarshal(overridden, &session); err != nil {
		t.Fatalf("unmarshal session: %v", err)
	}
	session.Expires = session.Expires.Add(-1 << 62) // far in the past
	overridden, _ = json.Marshal(session)

	regBody := buildRegistrationBody(t)
	_, err = p.FinishRegistration(ctx, userJSON, regBody, overridden)
	if err == nil {
		t.Fatal("expected error with expired session")
	}
}

func TestProvider_BeginLogin_NoCredentials(t *testing.T) {
	p := testProvider(t)
	ctx := context.Background()
	userJSON := testUserJSON(t) // no credentials

	_, _, err := p.BeginLogin(ctx, userJSON)
	if err == nil {
		t.Fatal("expected error when user has no credentials")
	}
}

func TestProvider_BeginLogin_InvalidUserJSON(t *testing.T) {
	p := testProvider(t)
	ctx := context.Background()

	_, _, err := p.BeginLogin(ctx, []byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestProvider_FinishLogin_WrongChallenge(t *testing.T) {
	p := testProvider(t)
	ctx := context.Background()

	// Register a credential first
	userJSON := testUserJSON(t)
	_, sessionData, err := p.BeginRegistration(ctx, userJSON)
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}
	overriddenSession := overrideChallenge(t, sessionData, regChallengeHex)
	credJSON, err := p.FinishRegistration(ctx, userJSON, buildRegistrationBody(t), overriddenSession)
	if err != nil {
		t.Fatalf("FinishRegistration: %v", err)
	}

	var cred credentialData
	json.Unmarshal(credJSON, &cred)
	loginUserJSON := testUserJSON(t, cred)

	_, loginSession, err := p.BeginLogin(ctx, loginUserJSON)
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}

	// Do NOT override — wrong challenge
	err = p.FinishLogin(ctx, loginUserJSON, buildLoginBody(t), loginSession)
	if err == nil {
		t.Fatal("expected error with wrong challenge")
	}
}

func TestCredentialConversion_RoundTrip(t *testing.T) {
	original := credentialData{
		ID:              []byte{0x01, 0x02, 0x03},
		PublicKey:       []byte{0x04, 0x05},
		AttestationType: "none",
		Transports:      []string{"internal", "usb"},
		AAGUID:          []byte{0x00, 0x01},
		SignCount:       42,
		BackupEligible:  true,
		BackupState:     false,
	}

	waCred := toWebAuthnCredential(original)
	converted := fromWebAuthnCredential(&waCred)

	if !bytes.Equal(original.ID, converted.ID) {
		t.Errorf("ID mismatch")
	}
	if !bytes.Equal(original.PublicKey, converted.PublicKey) {
		t.Errorf("PublicKey mismatch")
	}
	if original.AttestationType != converted.AttestationType {
		t.Errorf("AttestationType mismatch")
	}
	if len(original.Transports) != len(converted.Transports) {
		t.Errorf("Transports length mismatch")
	}
	if original.SignCount != converted.SignCount {
		t.Errorf("SignCount mismatch: %d vs %d", original.SignCount, converted.SignCount)
	}
	if original.BackupEligible != converted.BackupEligible {
		t.Errorf("BackupEligible mismatch")
	}
}

func TestTransportConversion(t *testing.T) {
	tests := []struct {
		name   string
		values []string
	}{
		{"nil", nil},
		{"empty", []string{}},
		{"single", []string{"internal"}},
		{"multiple", []string{"internal", "usb", "nfc"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol := toProtocolTransports(tt.values)
			back := fromProtocolTransports(protocol)
			if len(back) != len(tt.values) {
				t.Errorf("round-trip length mismatch: %d vs %d", len(back), len(tt.values))
			}
			for i, v := range tt.values {
				if back[i] != v {
					t.Errorf("transport[%d] = %q, want %q", i, back[i], v)
				}
			}
		})
	}
}

func TestParseUser_InvalidJSON(t *testing.T) {
	_, err := parseUser([]byte("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseSession_InvalidJSON(t *testing.T) {
	_, err := parseSession([]byte("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestWebauthnUserAdapter(t *testing.T) {
	user := &webauthnUser{data: userData{
		ID:          "test-id",
		Email:       "test@example.org",
		DisplayName: "Test User",
		Credentials: []credentialData{
			{ID: []byte{1}, PublicKey: []byte{2}, AttestationType: "none"},
		},
	}}

	if string(user.WebAuthnID()) != "test-id" {
		t.Errorf("WebAuthnID = %q, want %q", user.WebAuthnID(), "test-id")
	}
	if user.WebAuthnName() != "test@example.org" {
		t.Errorf("WebAuthnName = %q", user.WebAuthnName())
	}
	if user.WebAuthnDisplayName() != "Test User" {
		t.Errorf("WebAuthnDisplayName = %q", user.WebAuthnDisplayName())
	}
	creds := user.WebAuthnCredentials()
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
}

// Compile-time assertion that webauthnUser satisfies webauthn.User.
var _ webauthn.User = (*webauthnUser)(nil)
