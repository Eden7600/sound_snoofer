# Disposable Google Meet observations

Approved probe date: 2026-10-15. Browser: Brave, Chrome 155 user agent. UI language:
English. One participant (the tester), using a disposable call. The top-frame
origin was `https://meet.google.com`. No participant names, call identifiers,
meeting URLs, device details, page content or captured audio/video are retained.

## Joined controls

There was exactly one joined root, `[role="region"][jsname="vNB5le"]`, containing
one native Leave button `button[jsname="CQylAd"]`. Its mic and camera controls
were native BUTTON elements with `jsname="hw0c9"` and `jsname="psRWwc"`
respectively. Each observed actionable control was unique, had `disabled=false`,
and had a visible nonempty rectangle. Both used literal `data-is-muted="true"`
or `"false"`. Mic true meant muted; camera false meant camera on.

| Probe action | Observed readback |
| --- | --- |
| One click to mute the mic | Mic attribute changed false to true; read back at 504 ms. |
| One click to unmute while the Meet tab was hidden | Mic attribute changed true to false; read back at 1004 ms; `document.hidden` was true. |
| One click to turn camera on | An attribute MutationObserver recorded camera false at 1574 ms. |
| Turn camera off immediately after that observation | Camera true was observed at 1608 ms total from the initial camera-on request. |

The camera startup timing is an attribute-observer measurement. The microphone
values are observed readback times, not precise transition timestamps. These
samples support the planned 3 s page observation / 5 s total command budgets for
this layout; they do not establish scheduling guarantees or implemented timeout
behavior. The probe observes local UI state only, not remote media delivery.

## Leave, prejoin and rejoin

Leaving disconnected the old joined root and left zero joined roots and zero
Leave buttons. Rejoin opened a new document: a temporary probe marker was absent.
Before joining, the mic/camera `jsname` values were reused, while joined-root and
Leave counts remained zero. Thus those button names alone cannot establish a call.

Prejoin defaults had both mic and camera on. Both were explicitly turned off before
joining. The new joined page retained both Off and exposed one fresh joined root
and one Leave button. This demonstrates the observed lifecycle signals; it does
not exercise an implemented call epoch or stale-command rejection.

## Cleanup and limits

The disposable call was left, DevTools was closed, and the test tab was closed.
The original music remained playing.

Only this observed Brave/English layout is the initial adapter candidate. Missing,
conflicting, disabled, duplicate or unreadable controls must become unknown and
unavailable; those implementation cases still need fixture tests. No implemented
extension/provider/deck acceptance is claimed. Old-command rejection, same-document
call replacement, two simultaneous calls, correlated results, heartbeat expiry,
connection recovery, upgrade behavior and physical Deck input remain unperformed
acceptance or automated verification work in tasks 3–7.
