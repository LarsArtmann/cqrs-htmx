package usermgmt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// W3C WebAuthn Level 3 spec test vectors for the "none" attestation format
// with ES256 (ECDSA P-256). These are the official conformance test vectors
// published by the W3C and also used by go-webauthn's own test suite.
//
// See: https://www.w3.org/TR/webauthn-3/#sctn-test-vectors
const (
	// Registration vector
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

	// Login vector (same credential, different challenge)
	loginAuthenticatorDataHex = "bfabc37432958b063360d3ad6461c9c4735ae7f8edd46592a5e0f01452b2e4b51900000000"
	loginClientDataJSONHex    = "7b2274797065223a22776562617574686e2e676574222c226368616c6c656e6765223a224f63446e55685158756c5455506f334a5558543049393770767a7a59425039745a63685879617630314167222c226f726967696e223a2268747470733a2f2f6578616d706c652e6f7267222c2263726f73734f726967696e223a66616c73657d"
	loginSignatureHex         = "3046022100f50a4e2e4409249c4a853ba361282f09841df4dd4547a13a87780218deffcd380221008480ac0f0b93538174f575bf11a1dd5d78c6e486013f937295ea13653e331e87"
	loginChallengeHex         = "39c0e7521417ba54d43e8dc95174f423dee9bf3cd804ff6d65c857c9abf4d408"
)

// TestWebAuthnVirtualAuthenticator_FullFlow exercises the complete WebAuthn
// registration → login ceremony using W3C spec test vectors as a virtual
// authenticator. This is the closest thing to an integration test without
// a real browser or hardware security key.
//
// Flow:
//  1. Register a user (email only)
//  2. BeginRegistration → override session challenge to match spec vector
//  3. FinishRegistration with spec attestation → credential stored as event
//  4. BeginLogin → override session challenge to match login spec vector
//  5. FinishLogin with spec assertion → session created
func TestWebAuthnVirtualAuthenticator_FullFlow(t *testing.T) {
	svc := newWebAuthnTestServiceWithExampleOrg(t)
	ctx := context.Background()

	// Step 1: Register a user
	reg := registerTestUser(t, svc, "va1", "va1@test.com")

	// Step 2: Begin registration
	beginResp, err := svc.BeginRegistration(ctx, reg.User.ID)
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}
	if beginResp.Options == nil {
		t.Fatal("expected non-nil options")
	}

	// Override the session challenge to match the W3C spec vector
	overrideSessionChallenge(t, svc, reg.User.ID, regChallengeHex)

	// Step 3: Finish registration with spec vector attestation
	regReq := buildRegistrationRequest(t)
	if err := svc.FinishRegistration(ctx, reg.User.ID, regReq, "My Passkey"); err != nil {
		t.Fatalf("FinishRegistration: %v", err)
	}

	// Verify the credential was persisted
	user, ok := svc.readModel.FindByUserID(reg.User.ID)
	if !ok {
		t.Fatal("user not found after registration")
	}
	if len(user.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(user.Credentials))
	}
	if user.Credentials[0].Name != "My Passkey" {
		t.Errorf("credential name = %q, want %q", user.Credentials[0].Name, "My Passkey")
	}

	// Step 4: Begin login
	loginResp, err := svc.BeginLogin(ctx, "va1@test.com")
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}
	if loginResp.Options == nil {
		t.Fatal("expected non-nil login options")
	}

	// Override the login session challenge
	overrideSessionChallenge(t, svc, reg.User.ID, loginChallengeHex)

	// Step 5: Finish login with spec vector assertion
	loginReq := buildLoginRequest(t)
	result, err := svc.FinishLogin(ctx, reg.User.ID, loginReq)
	if err != nil {
		t.Fatalf("FinishLogin: %v", err)
	}
	if result.Session == nil {
		t.Fatal("expected non-nil session")
	}
	if result.User.Email != "va1@test.com" {
		t.Errorf("user email = %q", result.User.Email)
	}
}

// TestWebAuthnVirtualAuthenticator_RegistrationReplay verifies that a
// replayed attestation is rejected (session consumed after first use).
func TestWebAuthnVirtualAuthenticator_RegistrationReplay(t *testing.T) {
	svc := newWebAuthnTestServiceWithExampleOrg(t)
	ctx := context.Background()

	reg := registerTestUser(t, svc, "va2", "va2@test.com")

	beginResp, err := svc.BeginRegistration(ctx, reg.User.ID)
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}
	_ = beginResp

	overrideSessionChallenge(t, svc, reg.User.ID, regChallengeHex)

	regReq := buildRegistrationRequest(t)
	if err := svc.FinishRegistration(ctx, reg.User.ID, regReq, "First"); err != nil {
		t.Fatalf("FinishRegistration first: %v", err)
	}

	// Second attempt should fail — session was deleted
	regReq2 := buildRegistrationRequest(t)
	err = svc.FinishRegistration(ctx, reg.User.ID, regReq2, "Second")
	if err == nil {
		t.Fatal("expected error on replayed registration")
	}
}

// TestWebAuthnVirtualAuthenticator_LoginWrongChallenge verifies that
// using the wrong challenge in the login assertion fails.
func TestWebAuthnVirtualAuthenticator_LoginWrongChallenge(t *testing.T) {
	svc := newWebAuthnTestServiceWithExampleOrg(t)
	ctx := context.Background()

	reg := registerTestUser(t, svc, "va3", "va3@test.com")

	// Register credential first
	beginResp, err := svc.BeginRegistration(ctx, reg.User.ID)
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}
	_ = beginResp

	overrideSessionChallenge(t, svc, reg.User.ID, regChallengeHex)
	regReq := buildRegistrationRequest(t)
	if err := svc.FinishRegistration(ctx, reg.User.ID, regReq, "Key"); err != nil {
		t.Fatalf("FinishRegistration: %v", err)
	}

	// Begin login but DON'T override the challenge — the spec vector
	// challenge won't match, so the signature verification should fail.
	_, err = svc.BeginLogin(ctx, "va3@test.com")
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}
	// Deliberately leave the real challenge (not overridden)

	loginReq := buildLoginRequest(t)
	_, err = svc.FinishLogin(ctx, reg.User.ID, loginReq)
	if err == nil {
		t.Fatal("expected error with wrong challenge")
	}
}

// --- helpers ---

func newWebAuthnTestServiceWithExampleOrg(t *testing.T) *Service {
	t.Helper()
	return newWebAuthnTestServiceWithConfig(t, &WebAuthnConfig{
		RPID:          "example.org",
		RPDisplayName: "Test App",
		RPOrigins:     []string{"https://example.org"},
	})
}

func overrideSessionChallenge(t *testing.T, svc *Service, userID UserID, challengeHex string) {
	t.Helper()
	challengeBytes := decodeHex(t, challengeHex)
	challenge := base64RawURL(challengeBytes)

	session, err := svc.webauthnSessions.Get(userID.Get())
	if err != nil {
		t.Fatalf("get session for override: %v", err)
	}
	session.Challenge = challenge
	svc.webauthnSessions.Save(userID.Get(), session)
}

func buildRegistrationRequest(t *testing.T) *http.Request {
	t.Helper()
	credentialID := decodeHex(t, regCredentialIDHex)
	attObj := base64RawURL(decodeHex(t, regAttestationObjectHex))
	cdj := base64RawURL(decodeHex(t, regClientDataJSONHex))
	id := base64RawURL(credentialID)

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
	return &http.Request{
		Body: io.NopCloser(bytes.NewReader(data)),
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}
}

func buildLoginRequest(t *testing.T) *http.Request {
	t.Helper()
	credentialID := decodeHex(t, regCredentialIDHex)
	authData := base64RawURL(decodeHex(t, loginAuthenticatorDataHex))
	cdj := base64RawURL(decodeHex(t, loginClientDataJSONHex))
	sig := base64RawURL(decodeHex(t, loginSignatureHex))
	id := base64RawURL(credentialID)

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
	return &http.Request{
		Body: io.NopCloser(bytes.NewReader(data)),
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}
}

func decodeHex(t *testing.T, s string) []byte {
	t.Helper()
	data, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	return data
}

func base64RawURL(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}
