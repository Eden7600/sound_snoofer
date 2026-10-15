# Preserve GUI work across structural updates

## Why
GUI audit items 3 and 6 remain after the TypeScript, content and dropdown work. `app.ts`
replaces the entire content tree when its structure key changes. The focused input/select
deferral protects the active editor, but the eventual rebuild still loses open details, search,
button focus, scroll and unfocused form drafts. Stream Deck's three destructive confirmations
use `window.confirm`, which blocks polling and resolves its action against current controls.

## What Changes
- Preserve local presentation state across structural updates on the same screen using explicit
  stable element identities and narrow edit-context boundaries.
- Keep existing focused-editor deferral, dropdown commitment and stale-list checks; restoration
  must never dispatch, apply a draft or substitute a fresh revision for a captured one.
- Reuse the styled dialog for Delete Page, Forget Deck and disabling Own Layout. Capture the
  target and action revision when opened, continue polling and invalidate obsolete confirmations.
- Cover rebuilds, removed targets, context switches, dialog cancellation and stale confirmation
  with isolated browser fixtures.

## Impact
Shared build/state/dialog helpers, targeted identity annotations and draft restoration in screen
builders, the existing confirmation markup and browser tests. Coordinate after `view-state-transport`
and the connection-recovery checkpoint because they own the shared poll/state boundary.
No framework, provider/API change, action queue, persistent session cache or live integration test.
This proposal authorizes planning only until implementation is requested.
