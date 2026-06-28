# ADR 0024: Honest UI Protocol

**Status:** Accepted
**Date:** 2026-06-28
**Related:** [ADR 0023](0023-command-sync.md), [Brainstorming doc](../brainstorming/2026-06-27_offline-first-command-sync-research.html)

## Context

Standard "optimistic UI" lies to the user: it shows pending mutations as if they already succeeded. When the server rejects, the mutation silently disappears with no explanation. Screen readers announce pending state as confirmed truth (an accessibility violation).

The research session identified this as the "cardinal sin" of optimistic UI. The user explicitly requires the frontend to NOT lie — pending state must be communicated in a human-centric way.

## Decision

**Honest Optimistic UI**: the UI never presents pending state as final. Every mutation carries a visible sync-state lifecycle.

### Mutation lifecycle state machine

| State        | Meaning                                                  | UI treatment                             |
| ------------ | -------------------------------------------------------- | ---------------------------------------- |
| `idle`       | No in-flight mutation                                    | Normal                                   |
| `pending`    | Local state reflects intended outcome; request in flight | Muted/dashed, glanceable                 |
| `confirmed`  | Server confirmed; local state matches canonical          | Solid — transitions from pending         |
| `rejected`   | Server rejected or transport failed                      | Inline error with reason + retry/discard |
| `superseded` | A newer mutation replaced this operation's view          | Show diff, let user reconcile            |

### Sync-state as DOM attribute

Sync provenance is communicated via `data-sync-state="pending|confirmed|rejected"` attributes on rendered items. CSS classes react to these attributes:

- `.sync-pending` — dashed border, muted opacity (0.65), yellow sync dot
- `.sync-confirmed` — solid border, green sync dot
- `.sync-rejected` — red border, error background, inline retry button

### Global sync indicator

A global bar in the page header tracks aggregate counts: "All changes saved" (green), "3 pending — syncing..." (yellow), "1 failed — tap to retry" (red).

### Never-silent rollback

On rejection, the item NEVER disappears. It transitions to `data-sync-state="rejected"` with an inline error message and a retry button. The user's input is preserved.

### ACK-driven transitions

The [ACK protocol](0023-command-sync.md) drives state transitions. The `BroadcastOnAck()` hook broadcasts `{commandId, status, error}` over SSE. Client-side JavaScript listens for `sync:ack` events and flips the `data-sync-state` attribute on the matching item.

## Consequences

- **Trust**: users always know the provenance of what they see
- **Accessibility**: `aria-live="polite"` region announces only confirmed changes
- **No silent data loss**: rejected items stay visible with retry option
- **Principle**: "Pending state should be felt, not read" — glanceable visual shift, not verbose labels
