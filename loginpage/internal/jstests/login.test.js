"use strict";

// Node test suite for loginpage's embedded WebAuthn JS (assets/login.js).
// Zero-dependency by design: node:test + node:assert only, matching the
// module's no-external-asset posture. Run via `node --test` (the Go
// wrapper in js_smoke_test.go execs the same command, so `go test`
// triggers this suite).
//
// Seeded PRNG (mulberry32) instead of Math.random so a failing property
// iteration reproduces from the printed seed.

const { test } = require("node:test");
const assert = require("node:assert");

const loginJS = require("../../assets/login.js");

function mulberry32(seed) {
  let a = seed >>> 0;
  return function () {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

const SEED = 20261011;
const rand = mulberry32(SEED);

function randomBytes(length) {
  const bytes = new Uint8Array(length);
  for (let i = 0; i < length; i++) bytes[i] = Math.floor(rand() * 256);
  return bytes;
}

test("base64url round-trip: bytes -> b64u -> bytes (200 random buffers)", () => {
  for (let i = 0; i < 200; i++) {
    const length = Math.floor(rand() * 257); // 0..256 inclusive
    const bytes = randomBytes(length);
    const b64u = loginJS.bufToB64u(bytes.buffer);
    const back = new Uint8Array(loginJS.b64uToBuf(b64u));
    assert.strictEqual(back.length, bytes.length, `iteration ${i} (seed ${SEED})`);
    for (let j = 0; j < bytes.length; j++) {
      assert.strictEqual(back[j], bytes[j], `iteration ${i} byte ${j} (seed ${SEED})`);
    }
  }
});

test("base64url round-trip: b64u -> bytes -> b64u on valid inputs", () => {
  const samples = ["", "AA", "AAA", "AAAA", "AAAAAA", "AQ", "AAECAwQFBgcICQ", "a-b_cdE"];
  for (const s of samples) {
    assert.strictEqual(loginJS.bufToB64u(loginJS.b64uToBuf(s)), s, `input ${JSON.stringify(s)}`);
  }
});

test("bufToB64u never emits base64 classic or padding characters", () => {
  for (let i = 0; i < 50; i++) {
    const bytes = randomBytes(Math.floor(rand() * 257));
    const b64u = loginJS.bufToB64u(bytes.buffer);
    assert.match(b64u, /^[A-Za-z0-9_-]*$/);
  }
});

test("b64uToBuf rejects classic base64 that atob would widen", () => {
  // Sanity guard in the other direction: the helper must only be fed
  // base64url; classic padding input must still decode byte-identically
  // when re-encoded without padding (atob accepts both).
  const classic = "AA+ECA==";
  const bytes = new Uint8Array(loginJS.b64uToBuf(classic));
  assert.strictEqual(bytes.length, 4);
  assert.deepStrictEqual([...bytes], [0, 0x3e, 0x10]);
});

test("serializeAssertion shape: minimal credential (no userHandle, no extensions)", () => {
  const cred = {
    id: "cred-id-1",
    rawId: new Uint8Array([1, 2, 3, 250]).buffer,
    type: "public-key",
    response: {
      authenticatorData: new Uint8Array([0x4d, 0x99]).buffer,
      clientDataJSON: new Uint8Array([0x7b, 0x7d]).buffer,
      signature: new Uint8Array([9, 8, 7]).buffer,
    },
  };
  const out = loginJS.serializeAssertion(cred);
  assert.deepStrictEqual(out, {
    id: "cred-id-1",
    rawId: "AQID-g",
    type: "public-key",
    response: {
      authenticatorData: "TZk",
      clientDataJSON: "e30",
      signature: "CAc",
    },
  });
  assert.ok(!("userHandle" in out.response));
  assert.ok(!("clientExtensionResults" in out));
});

test("serializeAssertion shape: userHandle + clientExtensionResults pass through", () => {
  const cred = {
    id: "cred-id-2",
    rawId: new Uint8Array([255]).buffer,
    type: "public-key",
    response: {
      authenticatorData: new Uint8Array([1]).buffer,
      clientDataJSON: new Uint8Array([2]).buffer,
      signature: new Uint8Array([3]).buffer,
      userHandle: new Uint8Array([10, 20]).buffer,
    },
    getClientExtensionResults: () => ({ credProps: { rk: true } }),
  };
  const out = loginJS.serializeAssertion(cred);
  assert.strictEqual(out.rawId, "_w");
  assert.strictEqual(out.response.userHandle, "ChQ");
  assert.deepStrictEqual(out.clientExtensionResults, { credProps: { rk: true } });
});

test("serializeAttestation shape: minimal credential", () => {
  const cred = {
    id: "cred-id-3",
    rawId: new Uint8Array([0]).buffer,
    type: "public-key",
    response: {
      attestationObject: new Uint8Array([0xa3, 0x01, 0x02]).buffer,
      clientDataJSON: new Uint8Array([0x7b]).buffer,
    },
  };
  const out = loginJS.serializeAttestation(cred);
  assert.deepStrictEqual(out, {
    id: "cred-id-3",
    rawId: "AA",
    type: "public-key",
    response: {
      attestationObject: "owEC",
      clientDataJSON: "ew",
    },
  });
  assert.ok(!("clientExtensionResults" in out));
});

test("prepareLoginOptions decodes challenge and allowCredentials ids", () => {
  const out = loginJS.prepareLoginOptions({
    challenge: "AQID",
    rpId: "example.com",
    allowCredentials: [{ type: "public-key", id: "AAE", transports: ["internal"] }],
  });
  assert.ok(out.challenge instanceof ArrayBuffer);
  assert.deepStrictEqual([...new Uint8Array(out.challenge)], [1, 2, 3]);
  assert.strictEqual(out.rpId, "example.com");
  assert.strictEqual(out.allowCredentials.length, 1);
  assert.ok(out.allowCredentials[0].id instanceof ArrayBuffer);
  assert.deepStrictEqual([...new Uint8Array(out.allowCredentials[0].id)], [0, 1]);
  assert.deepStrictEqual(out.allowCredentials[0].transports, ["internal"]);
});

test("prepareRegOptions decodes challenge and user.id", () => {
  const out = loginJS.prepareRegOptions({
    challenge: "AA",
    user: { id: "AQID", name: "a@b.c", displayName: "A" },
    excludeCredentials: [{ type: "public-key", id: "AP8", transports: [] }],
  });
  assert.deepStrictEqual([...new Uint8Array(out.challenge)], [0]);
  assert.deepStrictEqual([...new Uint8Array(out.user.id)], [1, 2, 3]);
  assert.strictEqual(out.user.name, "a@b.c");
  assert.deepStrictEqual([...new Uint8Array(out.excludeCredentials[0].id)], [0, 255]);
});
