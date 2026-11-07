// cqrs-query v1.0.0 — RFC 10008 QUERY upgrade for htmx 2.x (cqrs-htmx).
//
// Load after htmx: <script src="/ext/cqrs-query.js"></script>
// Activate per element: hx-ext="cqrs-query" + hx-query on an hx-post element.
//
//   <button hx-post="/contacts/search" hx-query hx-ext="cqrs-query">Search</button>
//
// - hx-post is the fallback transport and the trigger (body/params semantics
//   come from the normal hx-post machinery).
// - hx-query upgrades the request to the native QUERY method. An empty value
//   ("hx-query") reuses the hx-post URL; a value overrides the path.
// - On a 405/501 answer (proxy/WAF rejecting the unknown method), the origin
//   is remembered in sessionStorage (1h TTL) and the request is retried
//   immediately as POST + X-Http-Method-Override: QUERY; every later request
//   on this origin skips the native attempt entirely.
//
// The server side is one handler + one middleware: mount the route as
// "QUERY /contacts/search" (app.Query) and wrap the mux with
// cqrshtmx.MethodOverrideMiddleware so the fallback POST is rewritten before
// routing. See docs/guides/http-query-method.md.
(function () {
  "use strict";

  var MEM_KEY = "cqrs-query-override-origins";
  var TTL_MS = 60 * 60 * 1000;

  function readMemory() {
    try {
      return JSON.parse(sessionStorage.getItem(MEM_KEY) || "{}");
    } catch (err) {
      return {};
    }
  }

  function originDowngraded() {
    var known = readMemory()[location.origin];
    return typeof known === "number" && known > Date.now();
  }

  function rememberOrigin() {
    var memory = readMemory();
    memory[location.origin] = Date.now() + TTL_MS;
    try {
      sessionStorage.setItem(MEM_KEY, JSON.stringify(memory));
    } catch (err) {
      /* private mode / storage disabled: degrade to per-request retry */
    }
  }

  htmx.defineExtension("cqrs-query", {
    onEvent: function (name, evt) {
      var detail = evt.detail || {};
      var elt = detail.elt;

      if (!elt || !elt.getAttribute) {
        return;
      }

      if (name === "htmx:configRequest") {
        if (elt.getAttribute("hx-query") === null) {
          return;
        }

        var upgrade = elt.getAttribute("hx-query");
        var path = upgrade === "" ? detail.path : upgrade;

        if (originDowngraded()) {
          detail.path = path;
          detail.verb = "post";
          detail.headers["X-Http-Method-Override"] = "QUERY";
        } else {
          detail.path = path;
          detail.verb = "QUERY";
          elt.__cqrsQueryNative = true;
        }

        return;
      }

      if (name === "htmx:responseError") {
        if (!elt.__cqrsQueryNative) {
          return;
        }

        var xhr = detail.xhr;
        if (!xhr || (xhr.status !== 405 && xhr.status !== 501)) {
          return;
        }

        elt.__cqrsQueryNative = false;
        rememberOrigin();
        evt.preventDefault();

        var config = detail.requestConfig || {};
        htmx.ajax("POST", config.path, {
          source: elt,
          values: config.parameters,
          headers: { "X-Http-Method-Override": "QUERY" }
        });
      }
    }
  });
})();
