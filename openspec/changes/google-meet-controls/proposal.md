# Google Meet controls

## Status
Proposed first-release scope awaiting the feasibility and transport design
checkpoints. The user's request authorizes production implementation within the
resolved scope; no additional implementation request is required. Implementation
and live probing remain unperformed. Live browser/call control requires an approved
session under AGENTS.md.

## Why
The user prioritizes controlling Google Meet microphone mute and camera state from
Stream Deck, with state visible as it is for Discord. Meetings currently contains
Discord controls only. Discord's mute deliberately shares Snoofer's global Mic Mute
preference; copying that link to Meet could propagate one call's mute into another.

## What Changes
- Add independently enabled Google Meet mic mute, camera and call status controls,
  available to the existing Stream Deck binding editor and a Meet card in Meetings.
- Keep Meet state local to one identified call. Keep Discord's existing global mute
  link and Snoofer's audio preferences unchanged.
- Require explicit selection when multiple Meet calls exist without a valid target;
  retain an existing selection when another call appears. Clear a lost target
  without falling through to another call or following browser/media focus.
- Reuse the browser extension's local transport after a narrow ownership/design
  checkpoint, so Meet works when Now Playing and Discord are disabled.
- Prove the adapter against a separately approved disposable call before choosing
  selectors and supported browser/layout coverage. Observe state on reconnect;
  never automatically unmute or turn a camera on.

## Impact
Expected implementation touches extension content/background routing and tests,
the existing browser bridge's ownership boundary, a Meet provider and registration,
Meetings lifecycle/attention presentation, semantic-control fixtures and deck/GUI
checks. Exact transport placement and lifecycle are a prerequisite design decision,
not authorization for a generic service framework. Update the Meetings portion of
the UI contract when implementing its intentional presentation change.

Controls become bindable first. This proposal does not change default deck positions
or any personal layout; placement is a separate user choice. No new cloud account,
audio/video capture, transcript, calendar or recording integration is proposed.
Screen sharing, leaving calls, push-to-talk and automatic meeting modes are outside
this release. Show Call, hand state and Step Away remain follow-up ideas in design.md.

## API rationale
The public [Meet REST reference](https://developers.google.com/workspace/meet/api/reference/rest/v2)
and [MeetAddonClient reference](https://developers.google.com/workspace/meet/add-ons/reference/websdk/addon_sdk.meetaddonclient)
reviewed on 2026-10-15 expose no documented local self-mic/camera setter. This is an
inference from those published surfaces, not a claim about private Google APIs.
The [Meet Media API](https://developers.google.com/workspace/meet/media-api/guides/overview)
consumes real-time media and currently accepts no new preview signups; it is not
needed for these controls. A small extension adapter is the candidate to validate.
