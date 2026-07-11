(function () {
  "use strict";

  var config = JSON.parse(document.getElementById("loginpage-config").textContent);

  // ── Base64URL helpers ────────────────────────────────────────────
  function b64uToBuf(b64u) {
    var b64 = b64u.replace(/-/g, "+").replace(/_/g, "/");
    var pad = b64.length % 4;
    if (pad) b64 += "=".repeat(4 - pad);
    var bin = atob(b64);
    var bytes = new Uint8Array(bin.length);
    for (var i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    return bytes.buffer;
  }

  function bufToB64u(buf) {
    var bytes = new Uint8Array(buf);
    var bin = "";
    for (var i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
    return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
  }

  // ── CSRF ────────────────────────────────────────────────────────
  function csrfToken() {
    var meta = document.querySelector('meta[name="csrf-token"]');
    return meta ? meta.getAttribute("content") : "";
  }

  function post(url, body) {
    return fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": csrfToken(),
      },
      body: typeof body === "string" ? body : JSON.stringify(body),
      credentials: "same-origin",
    });
  }

  async function postJSON(url, body) {
    var resp = await post(url, body);
    if (!resp.ok) throw await apiError(resp);
    return resp.json();
  }

  async function postRaw(url, body) {
    var resp = await post(url, body);
    if (!resp.ok) throw await apiError(resp);
    return resp;
  }

  async function apiError(resp) {
    var msg = "Request failed (" + resp.status + ")";
    try {
      var data = await resp.json();
      msg = data.error || data.detail || data.message || msg;
    } catch (_) {}
    var err = new Error(msg);
    err.status = resp.status;
    return err;
  }

  // ── WebAuthn option preparation ─────────────────────────────────
  function prepareLoginOptions(opts) {
    var p = Object.assign({}, opts);
    p.challenge = b64uToBuf(opts.challenge);
    if (opts.allowCredentials) {
      p.allowCredentials = opts.allowCredentials.map(function (c) {
        return { type: c.type, id: b64uToBuf(c.id), transports: c.transports };
      });
    }
    return p;
  }

  function prepareRegOptions(opts) {
    var p = Object.assign({}, opts);
    p.challenge = b64uToBuf(opts.challenge);
    if (opts.user) {
      p.user = Object.assign({}, opts.user, { id: b64uToBuf(opts.user.id) });
    }
    if (opts.excludeCredentials) {
      p.excludeCredentials = opts.excludeCredentials.map(function (c) {
        return { type: c.type, id: b64uToBuf(c.id), transports: c.transports };
      });
    }
    return p;
  }

  // ── Credential serialization ────────────────────────────────────
  function serializeAssertion(cred) {
    var r = cred.response;
    var out = {
      id: cred.id,
      rawId: bufToB64u(cred.rawId),
      type: cred.type,
      response: {
        authenticatorData: bufToB64u(r.authenticatorData),
        clientDataJSON: bufToB64u(r.clientDataJSON),
        signature: bufToB64u(r.signature),
      },
    };
    if (r.userHandle) out.response.userHandle = bufToB64u(r.userHandle);
    if (cred.getClientExtensionResults)
      out.clientExtensionResults = cred.getClientExtensionResults();
    return out;
  }

  function serializeAttestation(cred) {
    var r = cred.response;
    var out = {
      id: cred.id,
      rawId: bufToB64u(cred.rawId),
      type: cred.type,
      response: {
        attestationObject: bufToB64u(r.attestationObject),
        clientDataJSON: bufToB64u(r.clientDataJSON),
      },
    };
    if (cred.getClientExtensionResults)
      out.clientExtensionResults = cred.getClientExtensionResults();
    return out;
  }

  // ── Friendly error messages ─────────────────────────────────────
  function friendlyError(err) {
    if (err.name === "NotAllowedError") return "Passkey prompt was cancelled or timed out.";
    if (err.name === "SecurityError") return "This domain is not authorized for passkeys.";
    if (err.name === "AbortError") return "Operation was aborted.";
    return err.message || "An unexpected error occurred.";
  }

  function isWebAuthnError(err) {
    return (
      err.name === "NotAllowedError" || err.name === "SecurityError" || err.name === "AbortError"
    );
  }

  // ── UI helpers ──────────────────────────────────────────────────
  function showError(msg) {
    var el = document.getElementById("lp-error");
    if (el) {
      el.textContent = msg;
      el.classList.remove("lp-hidden");
    }
  }

  function hideError() {
    var el = document.getElementById("lp-error");
    if (el) el.classList.add("lp-hidden");
  }

  function setLoading(btnId, loading) {
    var btn = document.getElementById(btnId);
    if (!btn) return;
    btn.disabled = loading;
    btn.classList.toggle("lp-btn-loading", loading);
  }

  function showSection(id) {
    var sections = document.querySelectorAll(".lp-section");
    for (var i = 0; i < sections.length; i++) {
      sections[i].classList.toggle("lp-hidden", sections[i].id !== id);
    }
  }

  // ── Login flow ──────────────────────────────────────────────────
  async function doLogin(email) {
    hideError();
    setLoading("lp-login-btn", true);
    try {
      // 1. Begin login
      var begin = await postJSON(config.endpoints.loginBegin, { email: email });

      // 2. Browser prompt
      var cred = await navigator.credentials.get({
        publicKey: prepareLoginOptions(begin.options),
      });

      // 3. Finish login
      var finishUrl =
        config.endpoints.loginFinish + "?user_id=" + encodeURIComponent(begin.session_key);
      await postRaw(finishUrl, JSON.stringify(serializeAssertion(cred)));

      // 4. Redirect on success
      window.location.href = config.redirect;
    } catch (err) {
      showError(isWebAuthnError(err) ? friendlyError(err) : err.message);
    } finally {
      setLoading("lp-login-btn", false);
    }
  }

  // ── Registration flow ───────────────────────────────────────────
  async function doRegister(email, displayName) {
    hideError();
    setLoading("lp-register-btn", true);
    try {
      // 1. Register user (generates a UUID client-side)
      var userId =
        typeof crypto !== "undefined" && crypto.randomUUID
          ? crypto.randomUUID()
          : "u-" + Date.now() + "-" + Math.random().toString(36).slice(2);

      await postJSON(config.endpoints.register, {
        id: userId,
        email: email,
        display_name: displayName,
      });

      // 2. Begin WebAuthn registration
      var begin = await postJSON(config.endpoints.registerBegin, {
        user_id: userId,
      });

      // 3. Browser prompt
      var cred = await navigator.credentials.create({
        publicKey: prepareRegOptions(begin.options),
      });

      // 4. Finish registration
      var finishUrl =
        config.endpoints.registerFinish +
        "?user_id=" +
        encodeURIComponent(userId) +
        "&credential_name=" +
        encodeURIComponent(config.credentialName || "Passkey");
      await postRaw(finishUrl, JSON.stringify(serializeAttestation(cred)));

      // 5. Redirect on success
      window.location.href = config.redirect;
    } catch (err) {
      showError(isWebAuthnError(err) ? friendlyError(err) : err.message);
    } finally {
      setLoading("lp-register-btn", false);
    }
  }

  // ── Event wiring ────────────────────────────────────────────────
  document.addEventListener("DOMContentLoaded", function () {
    var loginForm = document.getElementById("lp-login-form");
    if (loginForm) {
      loginForm.addEventListener("submit", function (e) {
        e.preventDefault();
        var email = document.getElementById("lp-email").value.trim();
        if (!email) {
          showError("Please enter your email address.");
          return;
        }
        doLogin(email);
      });
    }

    var regForm = document.getElementById("lp-register-form");
    if (regForm) {
      regForm.addEventListener("submit", function (e) {
        e.preventDefault();
        var email = document.getElementById("lp-reg-email").value.trim();
        var name = document.getElementById("lp-reg-name").value.trim();
        if (!email) {
          showError("Please enter your email address.");
          return;
        }
        doRegister(email, name);
      });
    }

    var showReg = document.getElementById("lp-show-register");
    if (showReg) {
      showReg.addEventListener("click", function (e) {
        e.preventDefault();
        showSection("lp-register-section");
        hideError();
      });
    }

    var showLogin = document.getElementById("lp-show-login");
    if (showLogin) {
      showLogin.addEventListener("click", function (e) {
        e.preventDefault();
        showSection("lp-login-section");
        hideError();
      });
    }
  });
})();
