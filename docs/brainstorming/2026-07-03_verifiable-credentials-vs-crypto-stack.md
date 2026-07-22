W3C Verifiable Credentials — Relevance to cqrs-htmx Brainstorming · Research 2026-07-03 W3C VC vs cqrs-htmx crypto stack Contents Executive Summary What Are VCs? The 3-Party Model Is This Interesting? go-cqrs-lite Crypto VCs vs signing/encryption How They'd Compose The Capability Gaps Decisions & Recommendations Glossary References BRAINSTORMING · 2026-07-03 
# W3C Verifiable Credentials vs the cqrs-htmx Crypto Stack 

An analysis of the W3C Verifiable Credentials Data Model v2.1 and its relationship to
 cqrs-htmx's authentication/authorization model and go-cqrs-lite's `signing/ `and `encryption/ `modules. Are they the same thing? Do
 they overlap? Should we integrate? 

Source: W3C VC Data Model v2.1 Projects: cqrs-htmx, go-cqrs-lite Verdict: orthogonal — no integration needed 
## ★ Executive Summary 

W3C Verifiable Credentials (VCs) and cqrs-htmx solve fundamentally different problems . VCs are a cross-organizational
 identity attestation standard — "issuer X says subject Y has attribute Z, here's
 the cryptographic proof." cqrs-htmx (and its go-cqrs-lite crypto modules) solve web application auth + event pipeline integrity/confidentiality . They
 share the same underlying math (digital signatures, AEAD) but operate at entirely
 different layers with different trust models, canonicalization formats, and purposes. 

0 Overlap between VC concepts and signing/encryption code 3 Crypto primitives in common (Ed25519, AEAD, signatures) 1 Potential future consumer use case (VC issuance from events) 0 Reasons to integrate now Bottom line: Conceptually adjacent, practically orthogonal. Not worth
 integrating. If a consumer ever needs "prove your diploma / membership / age from
 another system," that's when a `usermgmt/credentials `module behind an
 interface (like the existing TOTP/WebAuthn/OAuth2 pattern) would be the right seam.
 Until then it's YAGNI. 
## 01 What Is a Verifiable Credential? 

A VC is a cryptographically signed, machine-verifiable set of claims about a subject. Think: digital driver's license, university diploma, or health
 insurance card — but tamper-evident and verifiable without phoning home to the
 issuer. 

The core data model has three layers: 

Concept What it is Example Claim Subject → property → value `Pat —alumniOf→ Example University `Credential Set of claims + metadata (issuer, validity, type, status) `{"type":"DegreeCredential","issuer":"university.example",...} `Verifiable Credential Credential + cryptographic proof (signature) JSON-LD document with a `proof `block or wrapped as SD-JWT Verifiable Presentation Holder-derived subset, optionally with selective disclosure / ZKP "Here's my degree, but not my GPA or address" 
### Securing Mechanisms 

The spec recognizes two classes of proofs: 

#### Embedded Proof (Data Integrity) 

Signature lives inside the JSON document as a `proof `property. Uses
 JSON-LD canonicalization (URDNA2015) + Ed25519 or other suites. 

```
{
 "@context": [...],
 "type": ["VerifiableCredential", ...],
 "issuer": "did:example:123",
 "credentialSubject": { ... },
 "proof": {
 "type": "DataIntegrityProof",
 "cryptosuite": "eddsa-rdfc-2022",
 "proofValue": "z58DAd..."
 }
} 
```

#### Enveloping Proof (JOSE/COSE) 

Signature wraps the credential. SD-JWT, JWS, or COSE. Supports selective disclosure
 (reveal subset of claims). 

```
eyJhbGciOiJFUzM4NCIsImtpZCI6IkdOV...
.eyJAY29udGV4dCI6WyJodHRwczovL3d3dy53My...
.credentialSubject: {
 "_sd": ["SE8zxxnge3Mma0-CfKgeamk5Ej..."]
}
~WyJFX3F2V09NWVQ1Z3JNTkprOHNXN3BBIiwgImlkIiw...] 
```
Key insight: The spec defines what to sign (a credential
 document) and why (issuer attestation). The how is delegated to
 securing mechanism specs (Data Integrity, VC-JOSE-COSE). Those specs use standard crypto
 primitives — the same primitives go-cqrs-lite's `signing/ `module provides — but wrapped in credential-specific canonicalization, proof
 purposes, and key resolution. 
## 02 The Three-Party Trust Model 

VCs explicitly decouple the identity provider concept into two distinct
 roles. This is the fundamental architectural difference from cqrs-htmx's model. 

ISSUER — Asserts claims about a subject, signs a credential,
 sends to holder. 
Examples: universities, governments, employers HOLDER — Stores credentials in a wallet. Generates selective
 presentations. 
The holder IS the subject (usually). Holds their own data. VERIFIER — Receives a presentation, checks signature + status,
 validates claims. 
Does NOT contact the issuer. Verification is self-contained. 
A fourth supporting role: 

 - Verifiable Data Registry — Mediates identifier/key creation and
 verification (DIDs, status lists, schemas). Can be a trusted database, a distributed
 ledger, or a DID network. 
### Privacy-Enhancing Features 

 - Selective disclosure — Holder reveals only specific claims from
 a credential, not the whole thing. 
 - Unlinkable disclosure — Presentations cannot be correlated
 between verifiers (via zero-knowledge proofs). 
 - No phone-home — The spec explicitly forbids status checking
 mechanisms that enable issuer tracking of holder usage. 
## 03 Is This Interesting to cqrs-htmx? 

Conceptually adjacent, practically orthogonal. The project solves
 authentication + authorization for web apps. VCs solve portable, cross-organizational
 attestation. Different problems: 

Dimension cqrs-htmx today Verifiable Credentials Question answered "Are you who you say?" (auth) "Do you have attributes X, Y, Z, attested by Z?" State model Session cookie, server-held user store Self-contained signed document, no server callback Authorization Casbin RBAC role checks Claim-based authorization, selective disclosure Trust boundary Single organization / app Cross-organizational, decentralized Identity provider Server is the IdP (sessions) Decoupled into Issuer + Holder (no IdP) Key resolution Static server config DID documents, controlled identifier docs Revocation Session deletion Bitstring status lists, no phone-home Where they'd intersect: 
 - A cqrs-htmx app acting as Issuer — emit VCs from the
 event-sourced user aggregate. The event-sourcing model maps cleanly (e.g. a `CredentialIssued `event). 
 - A cqrs-htmx app acting as Verifier — accept a VP at login
 instead of / in addition to WebAuthn, derive roles from claims. 
 - WebAuthn and VCs complement — WebAuthn proves possession of a
 key; a VC proves attributes. Wallets use WebAuthn to unlock VC stores. 
## 04 go-cqrs-lite's Crypto Modules 

Before comparing, let's establish what `signing/ `and `encryption/ `actually do. Both are general-purpose infrastructure for
 event-sourcing pipelines. Neither has any identity, credential, or attestation concepts. 

signing/v3 — Event Integrity 
Purpose: Tamper detection + origin authenticity for events
 crossing untrusted boundaries (message brokers, edge, devices). 

Primitives: HMAC-SHA256 ( `NewHMAC `), Ed25519 ( `NewEd25519 `), multisig
 for hop-chains ( `signing/multisig `). 

Signs: Canonical byte representation of `event.Event `fields (ID, Type, AggregateID, Version, etc.) + SHA-256
 of payload. Not arbitrary documents. 

Wiring: `SignMiddleware `(publish), `VerifyMiddleware `(handle). Signature stored in event metadata under `"event.signature" `. 

encryption/v3 — Event Confidentiality 
Purpose: AEAD encryption of event payloads for data-at-rest /
 data-in-transit (GDPR, HIPAA, PCI-DSS compliance). 

Primitives: XChaCha20-Poly1305 ( `NewXChaCha20Poly1305 `), AES-256-GCM
 ( `NewAES256GCM `). HKDF-SHA256 key derivation ( `DeriveKey `)
 for multi-tenant. 

Encrypts: `[]byte `plaintext → `Ciphertext `( `[]byte `). All-or-nothing — no partial
 disclosure. 

Wiring: Bus middleware ( `EncryptMiddleware `/ `DecryptMiddleware `), store wrapper ( `NewEncryptedStore `),
 codec wrapper ( `NewCodec `). Composable with signing
 (sign-then-encrypt). 

Zero VC-related concepts in either module: No mentions of W3C, DID,
 JSON-LD, selective disclosure, SD-JWT, claims, credentials, holder binding, proof
 purposes, or revocation. Both are pure transport/storage security for the event
 pipeline. 
## 05 VCs vs signing/encryption — Side by Side 

They're different layers of the same crypto stack with zero overlap in
 code or vocabulary. Same math, entirely different problem domain. 

signing/ encryption/ W3C VC Data Model Layer Event integrity (point-to-point) Event confidentiality (at-rest) Credential attestation (3-party trust) Protects `event.Event `metadata + payload hash `event.Event `payload bytes Arbitrary claims about a subject Primitives HMAC-SHA256, Ed25519, multisig XChaCha20-Poly1305, AES-256-GCM Data Integrity proofs, JOSE/COSE (SD-JWT) Signature over Canonical event fields N/A (AEAD, not signature) Canonical JSON-LD of the credential Key model Server-held shared secret or keypair Server-held symmetric key Issuer keypair → resolved via DID or JWK Who verifies Trusted services sharing keys Trusted services with the key Anyone — cross-domain, no callback Trust boundary Inside your infrastructure Inside your infrastructure Across organizations, no phone-home Revocation None None Bitstring status lists (privacy-preserving) Selective disclosure No No (AEAD = all-or-nothing) Yes (SD-JWT, BBS+, ZKP) 
## 06 The Capability Gaps 

`signing/ `provides Ed25519 signatures over a canonical byte representation
 — that's a subset of what VC Data Integrity needs. But VCs add layers
 that neither module has any concept of: 

#### 1. Canonicalization 

`signing/ `canonicalizes event metadata fields (ID, Type, AggregateID...).
 VCs need JSON-LD canonicalization (URDNA2015) of arbitrary claim
 graphs. Completely different canonical form. The signing module's canonicalization is
 event-specific and cannot be reused. 

#### 2. Proof Purposes 

VC signatures declare purpose: `assertionMethod `, `authentication `, `capabilityDelegation `, etc. `signing/ `signatures have no purpose field — they just mean "this
 event wasn't tampered." 

#### 3. Holder Binding 

VCs are bound to a holder's key in Verifiable Presentations. A verifier checks that
 the presenter is the legitimate holder. Neither module has any subject/holder concept. 

#### 4. Selective Disclosure 

SD-JWT / BBS+ lets holders reveal subsets of claims. `encryption/ `does
 full-plaintext-or-nothing (AEAD). No mechanism for partial disclosure exists in the
 encryption module. 

#### 5. Status & Revocation 

VCs reference status lists (e.g. `BitstringStatusListEntry `) for
 revocation/suspension without phone-home. `signing/ `has no revocation. 

#### 6. DID / Key Resolution 

VCs resolve issuer keys via DIDs or controlled identifier docs. `signing/ `keys are configured statically — no resolution layer. 

Verdict: `signing/ `'s Ed25519 key gen and crypto ops are
 standard primitives, but the canonicalization and metadata format are event-specific and not reusable for VC signing. A VC implementation would need its own
 canonicalization (JSON-LD / URDNA2015), proof purpose model, and key resolution —
 essentially a full separate library. 
## 07 How They'd Compose (If Ever Needed) 

If a consumer wanted to issue VCs from a go-cqrs-lite event-sourced aggregate , the architecture
 would use both layers — they don't compete: 

Event Store 
(encrypted at rest via encryption/ — your infrastructure security) Event Replay → Projection 
(signed via signing/ in transit — pipeline integrity) Projection → Build Credential Payload 
(derive claims from aggregate state) VC Issuance — THE GAP 
Sign the credential JSON-LD with Data Integrity or SD-JWT 
Needs a VC library (JSON-LD canonicalization + proof purposes + key resolution) 
NOT signing/ (which only signs event.Event canonical form) Verifiable Credential 
Self-contained signed document, held by user, presented to verifiers 
The `signing/ `and `encryption/ `modules protect the event-sourcing infrastructure (your internal pipeline). VCs protect derived attestations shared with the outside world. You'd use both
 — they operate at different layers. 

If this ever becomes real: The right seam would be a `usermgmt/credentials `module behind an interface, following the same pattern
 as TOTP/WebAuthn/OAuth2. The module would define a `CredentialIssuer `interface with `[]byte `boundaries (like the
 existing auth strategy interfaces). A `vc-data-integrity/ `or `vc-jose/ `strategy module would implement it. Core `usermgmt `stays clean, consumers opt in only if they need VCs. 
## 08 Decisions & Recommendations 
✓ Do not integrate VCs into cqrs-htmx or go-cqrs-lite Different problem domain (decentralized identity vs web app auth). YAGNI. ✓ Document the relationship This document. Prevents future re-investigation of the same question. ✓ If a consumer needs VCs, add a strategy module Follow the existing TOTP/WebAuthn/OAuth2 pattern: interface in `usermgmt `with `[]byte `boundaries, implementation in a
 separate Go module ( `usermgmt/vc-data-integrity/ `or similar). Structural
 typing, compile-time assertions in integration_test. Core stays clean. ✓ Do not extend signing/ to support VC signing Wrong abstraction. `signing/ `canonicalizes event.Event fields, not
 JSON-LD claim graphs. VC signing needs URDNA2015 + proof purposes + key resolution
 — a full separate concern. Forcing it into `signing/ `would corrupt the module's single responsibility. ✓ Do not extend encryption/ to support selective disclosure AEAD is all-or-nothing by design. Selective disclosure requires fundamentally
 different cryptography (BBS+, Merkle trees, or SD-JWT disclosure arrays). 
## 09 Glossary 
Verifiable Credential (VC) A tamper-evident credential whose authorship can be cryptographically verified.
 Contains claims + metadata + proof. Claim A statement about a subject, expressed as subject → property → value. Issuer The entity that asserts claims, creates a VC, and transmits it to a holder. Holder The entity that possesses VCs and generates verifiable presentations. Often (but not
 always) the subject. Verifier The entity that receives a VP, verifies the cryptographic proof, and validates claims
 against business rules. Verifiable Presentation (VP) A holder-derived subset of one or more VCs, presented to a verifier. May use selective
 disclosure or ZKP. Data Integrity Proof An embedded proof (signature inside the JSON document). Uses JSON-LD canonicalization
 (URDNA2015). SD-JWT Selective Disclosure JWT. An enveloping proof that allows revealing a subset of claims
 from a credential. DID (Decentralized Identifier) A portable URL-based identifier (e.g. `did:example:123456abcdef `) used to
 identify subjects and issuers in VCs. Selective Disclosure The ability of a holder to share only specific claims from a credential, not the whole
 document. Unlinkable Disclosure A presentation that cannot be correlated between verifiers (via zero-knowledge
 proofs). BitstringStatusList A privacy-preserving revocation mechanism using a compact bitstring indexed by
 credential. No phone-home to issuer. HMAC-SHA256 Symmetric key signature used in go-cqrs-lite `signing/ `. Same key signs and
 verifies. Ed25519 Asymmetric signature algorithm used in both `signing/ `and VC Data
 Integrity (eddsa-rdfc-2022 suite). AEAD (Authenticated Encryption with Associated Data) Encryption mode used in `encryption/ `(XChaCha20-Poly1305, AES-256-GCM).
 All-or-nothing — no partial disclosure. URDNA2015 Universal RDF Dataset Canonicalization Algorithm. The JSON-LD canonicalization used by
 VC Data Integrity proofs. `signing/ `uses its own event-specific canonical
 form instead. HKDF-SHA256 Key derivation function (RFC 5869) used in `encryption/ `for multi-tenant
 key derivation. Not related to VCs. 
## 10 References 

 - W3C Verifiable Credentials Data Model v2.1 [https://w3c.github.io/vc-data-model/]— the spec analyzed in this document 
 - VC Data Integrity [https://w3c.github.io/vc-data-integrity/]— embedded proof securing mechanism (Ed25519, JSON-LD canonicalization) 
 - VC JOSE/COSE [https://w3c.github.io/vc-jose-cose/]— enveloping proof securing mechanism (SD-JWT, JWS, COSE) 
 - go-cqrs-lite `signing/v3 `— `~/projects/go-cqrs-lite/signing/ `
 - go-cqrs-lite `encryption/v3 `— `~/projects/go-cqrs-lite/encryption/ `
 - cqrs-htmx ADR-0011: Event Signing & Encryption — `docs/adr/0011-event-signing-encryption.md `
 - cqrs-htmx auth strategy extraction pattern — ADR-0035, `docs/migrations/v3-to-v4.md `Generated 2026-07-03 · cqrs-htmx brainstorming docs · W3C VC Data Model v2.1
 analysis 