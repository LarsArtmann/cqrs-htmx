// T24 F176 — idiomorph-on-tables spike (scratch experiment; NOT shipped in
// adminui; idiomorph is deliberately NOT an adminui dependency).
//
// Mirrors the adminui users table shape: 200 rows, stable tr[id], per-row
// email cell, one row mutated + one inserted + one removed per swap round,
// plus a per-row focus target (the delete button) to probe focus retention.
import { Idiomorph } from "./idiomorph.esm.js";

const ROWS = 200;
const ROUNDS = 50;

function rows(version) {
  const out = [];
  for (let i = 0; i < ROWS; i++) {
    out.push(`
      <tr id="u${i}">
        <td>user-${i}</td><td>user-${i}@example.com</td>
        <td>rendered ${version}</td>
        <td><button id="btn-u${i}" data-focus-probe>remove</button></td>
      </tr>`);
  }
  return out.join("");
}

function render(tbodyId, version) {
  const tb = document.getElementById(tbodyId);
  tb.innerHTML = rows(version);
}

function morph(tbodyId, version) {
  const tb = document.getElementById(tbodyId);
  const next = document.createElement("tbody");
  next.setAttribute("id", tbodyId);
  next.innerHTML = rows(version);
  Idiomorph.morph(tb, next);
}

// MutationObserver churn counter: added/removed DOM nodes across swaps.
function makeCounter() {
  let added = 0, removed = 0;
  const obs = new MutationObserver((muts) => {
    for (const m of muts) {
      added += m.addedNodes.length;
      removed += m.removedNodes.length;
    }
  });
  return {
    start(target) { obs.observe(target, { childList: true, subtree: true }); },
    stop() { obs.disconnect(); return { added, removed }; },
  };
}

async function runVariant(name, swapFn, tbodyId, counterTarget) {
  render(tbodyId, 0);
  const counter = makeCounter();
  counter.start(counterTarget);
  let focusKept = 0;
  for (let v = 1; v <= ROUNDS; v++) {
    // Focus a row control, swap, ask whether the SAME element is still
    // focused (innerHTML rebuild) or its id-identical replacement (morph).
    const btn = document.getElementById(`btn-u${v % ROWS}`);
    btn.focus();
    const before = document.activeElement;
    swapFn(tbodyId, v);
    const after = document.activeElement;
    if (after === before || (before && after && after.id === before.id)) focusKept++;
  }
  // MutationObserver delivers records on a microtask — flush before reading.
  await new Promise((r) => setTimeout(r, 50));
  const churn = counter.stop();
  return { variant: name, rounds: ROUNDS, nodesAdded: churn.added, nodesRemoved: churn.removed, focusKept, focusKeptPct: Math.round((100 * focusKept) / ROUNDS) };
}

const results = [
  await runVariant("innerHTML", render, "a-body", document.getElementById("a-table")),
  await runVariant("idiomorph", morph, "b-body", document.getElementById("b-table")),
];

// Timing (coarse; shared machine, advisory only).
function timed(fn, tbodyId) {
  const t0 = performance.now();
  for (let v = 1; v <= ROUNDS; v++) fn(tbodyId, v);
  return Math.round(performance.now() - t0);
}
results[0].swapMs = timed(render, "a-body");
results[1].swapMs = timed(morph, "b-body");

document.getElementById("result").textContent = JSON.stringify(results, null, 2);
window.__SPIKE_DONE = true;
