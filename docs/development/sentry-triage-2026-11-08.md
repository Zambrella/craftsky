# Production Sentry triage — 2026-11-08

Scope: `craftsky/flutter-app` and `craftsky/app-view`, environments `prod` and
`production`. Authenticated `sentry issue list/view --json` inspected 101 open
issues (78 Flutter, 23 AppView), including latest exception chains, selected
operation context, release and timestamps. A second CLI pass retrieved 373
production events over 90 days (no additional pages), checking mixed-operation
groups including APP-VIEW-5, 16, 1W and Q. Both the server API/PDS diagnostics and
chat diagnostics are retained for those mixed groups. An unfiltered queue check also found
six dev-only AppView issues; the maintainer explicitly requested leaving those
unchanged: APP-VIEW-10, H, 12, 1A, 1B and 1C.

## Interpretation and changes

A resolution in the next release means a confirmed fix, a verified existing fix,
or improved/verified recurrence diagnostics under the requested triage policy.
It does **not** establish that upstream outages or Flutter engine failures have
been repaired, or that an unreleased change has reached a production device.

- Fix delayed requests reaching a retired HTTP adapter during account teardown.
  Active-lifetime transport failures keep their original error behavior.
- Skip like-feedback painting for singular transforms and detached render boxes.
  The retained-paint regression throws `ArgumentError` with the original code.
- Check the read audience/lifetime before search pagination applies post overlays.
  Build-time guards already exist in this checkout.
- Classify absent/loading active-account language preferences as cancellation;
  preserve real initialization failures and rebuild when initialization changes.
- Preserve native Riverpod cause/stack chains, known picker error codes, fixed
  draft validation reasons, and scheduler/lifecycle/accessibility state.
- Preserve only current/crashed native thread stacks, sanitized with the existing
  exception-stack filter; exclude names, registers, unknown payloads, source and
  locals while retaining native symbolication coordinates.
- Give chat synchronization stage-specific diagnostics and the durable failure
  count rather than reporting every failure as `poll`, attempt 1. Keep retries,
  backoff, claim fencing and authorization unchanged.
- Treat a refused stale OAuth callback after owner generation changes as an
  expected validation outcome, preserving refusal and local diagnostics.
- Distinguish OAuth credential-quarantine checks without exposing compared values;
  retain initial-token HTTP status and vetted provider code without prose.

No API, lexicon, infrastructure, release version, deployment or consent changes.
Raw event payloads and private account references are excluded from this report.

## Per-issue coverage

| Issue | Latest inspected release | Disposition | Evidence / recurrence coverage |
|---|---|---|---|
| [APP-VIEW-13](https://craftsky.sentry.io/issues/157501803/) | craftsky-appview@1.0.20 | Existing diagnostics | Existing server boundary retains operation/stage, typed cause, selected status/code and allowed correlation; evidence does not establish a local defect. |
| [APP-VIEW-14](https://craftsky.sentry.io/issues/157502497/) | craftsky-appview@1.0.20 | Existing diagnostics | Existing server boundary retains operation/stage, typed cause, selected status/code and allowed correlation; evidence does not establish a local defect. |
| [APP-VIEW-15](https://craftsky.sentry.io/issues/157503028/) | craftsky-appview@1.0.20 | Existing diagnostics | Existing server boundary retains operation/stage, typed cause, selected status/code and allowed correlation; evidence does not establish a local defect. |
| [APP-VIEW-16](https://craftsky.sentry.io/issues/157503081/) | craftsky-appview@1.1.1 | Diagnostics | Report actual failed sync boundary and durable attempt count; retain upstream status/code and typed cause; retry/claim behavior unchanged. |
| [APP-VIEW-17](https://craftsky.sentry.io/issues/157507539/) | craftsky-appview@1.0.20 | Existing diagnostics | Existing server boundary retains operation/stage, typed cause, selected status/code and allowed correlation; evidence does not establish a local defect. |
| [APP-VIEW-18](https://craftsky.sentry.io/issues/157508002/) | craftsky-appview@1.0.20 | Diagnostics | Report actual failed sync boundary and durable attempt count; retain upstream status/code and typed cause; retry/claim behavior unchanged. |
| [APP-VIEW-1D](https://craftsky.sentry.io/issues/157773929/) | craftsky-appview@1.1.1 | Diagnostics | Quarantine failures distinguish invalid input, changed purpose/state/attempt and provider metadata mismatch without OAuth capabilities. |
| [APP-VIEW-1E](https://craftsky.sentry.io/issues/157841307/) | craftsky-appview@1.1.1 | Diagnostics | Select numeric initial-token response status and reviewed OAuth error code from the pinned library format; preserve wrapped cause. |
| [APP-VIEW-1F](https://craftsky.sentry.io/issues/157984275/) | craftsky-appview@1.1.1 | Existing diagnostics | Existing server boundary retains operation/stage, typed cause, selected status/code and allowed correlation; evidence does not establish a local defect. |
| [APP-VIEW-1G](https://craftsky.sentry.io/issues/159195850/) | craftsky-appview@1.2.0 | Existing diagnostics | Existing server boundary retains operation/stage, typed cause, selected status/code and allowed correlation; evidence does not establish a local defect. |
| [APP-VIEW-1T](https://craftsky.sentry.io/issues/159223858/) | craftsky-appview@1.2.0 | Diagnostics | Report actual failed sync boundary and durable attempt count; retain upstream status/code and typed cause; retry/claim behavior unchanged. |
| [APP-VIEW-1W](https://craftsky.sentry.io/issues/159453305/) | craftsky-appview@1.2.0 | Diagnostics | Report actual failed sync boundary and durable attempt count; retain upstream status/code and typed cause; retry/claim behavior unchanged. |
| [APP-VIEW-1X](https://craftsky.sentry.io/issues/159459372/) | craftsky-appview@1.2.0 | Diagnostics | Report actual failed sync boundary and durable attempt count; retain upstream status/code and typed cause; retry/claim behavior unchanged. |
| [APP-VIEW-5](https://craftsky.sentry.io/issues/152673607/) | craftsky-appview@1.0.17 | Existing diagnostics | Existing server boundary retains operation/stage, typed cause, selected status/code and allowed correlation; evidence does not establish a local defect. |
| [APP-VIEW-G](https://craftsky.sentry.io/issues/154868893/) | craftsky-appview@1.0.17 | Existing diagnostics | Repository job diagnostics already retain lease reason, stage, real attempt, retry/exhaustion and original cause; no evidence supports changing fencing. |
| [APP-VIEW-N](https://craftsky.sentry.io/issues/154988063/) | craftsky-appview@1.2.0 | Diagnostics | Report actual failed sync boundary and durable attempt count; retain upstream status/code and typed cause; retry/claim behavior unchanged. |
| [APP-VIEW-P](https://craftsky.sentry.io/issues/154989159/) | craftsky-appview@1.2.0 | Existing diagnostics | Existing server boundary retains operation/stage, typed cause, selected status/code and allowed correlation; evidence does not establish a local defect. |
| [APP-VIEW-Q](https://craftsky.sentry.io/issues/155249553/) | craftsky-appview@1.2.0 | Diagnostics | Report actual failed sync boundary and durable attempt count; retain upstream status/code and typed cause; retry/claim behavior unchanged. |
| [APP-VIEW-R](https://craftsky.sentry.io/issues/155262652/) | craftsky-appview@1.0.18 | Fixed | Owner-generation changes invalidate an older OAuth callback by design; classify that refusal as expected validation while retaining diagnostics.
| [APP-VIEW-T](https://craftsky.sentry.io/issues/155483714/) | craftsky-appview@1.0.19 | Diagnostics | Quarantine failures distinguish invalid input, changed purpose/state/attempt and provider metadata mismatch without OAuth capabilities. |
| [APP-VIEW-V](https://craftsky.sentry.io/issues/155551958/) | craftsky-appview@1.0.18 | Existing diagnostics | Repository job diagnostics already retain lease reason, stage, real attempt, retry/exhaustion and original cause; no evidence supports changing fencing. |
| [APP-VIEW-W](https://craftsky.sentry.io/issues/155555091/) | craftsky-appview@1.0.18 | Existing diagnostics | Repository job diagnostics already retain lease reason, stage, real attempt, retry/exhaustion and original cause; no evidence supports changing fencing. |
| [APP-VIEW-Z](https://craftsky.sentry.io/issues/156732835/) | craftsky-appview@1.0.19 | Existing diagnostics | Existing server boundary retains operation/stage, typed cause, selected status/code and allowed correlation; evidence does not establish a local defect. |
| [FLUTTER-APP-1D](https://craftsky.sentry.io/issues/154597620/) | social.craftsky.app@1.8.0+13 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-1E](https://craftsky.sentry.io/issues/154818006/) | social.craftsky.app@1.8.0+13 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-1P](https://craftsky.sentry.io/issues/154883906/) | social.craftsky.app@1.8.0+13 | Existing fix | Legacy Android document picker was replaced by buffered native copy and path/isolate parsing; large-device import remains unverified. |
| [FLUTTER-APP-1Q](https://craftsky.sentry.io/issues/154884385/) | social.craftsky.app@1.7.5+12 | Existing fix | Legacy picker allocation path was removed; current plugin copies to a path rather than allocating the selected document. |
| [FLUTTER-APP-1R](https://craftsky.sentry.io/issues/154916991/) | social.craftsky.app@1.7.5+12 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-1S](https://craftsky.sentry.io/issues/154916997/) | social.craftsky.app@1.7.5+12 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-1T](https://craftsky.sentry.io/issues/154917001/) | social.craftsky.app@1.7.5+12 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-1X](https://craftsky.sentry.io/issues/154926566/) | social.craftsky.app@1.10.0+15 | Diagnostics | No proven app-level root cause. Retain scheduler/lifecycle/accessibility state, safe navigation history, native stacks and underlying provider causes. |
| [FLUTTER-APP-26](https://craftsky.sentry.io/issues/155326973/) | social.craftsky.app@1.7.5+12 | Fixed / diagnostics | Existing build audience guard plus pagination lifetime check prevent stale overlay work; native provider causes now survive filtering. |
| [FLUTTER-APP-27](https://craftsky.sentry.io/issues/155380642/) | social.craftsky.app@1.8.0+13 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-28](https://craftsky.sentry.io/issues/155381649/) | social.craftsky.app@1.8.0+13 | Diagnostics | Retain draft failure reason and closed validation detail (image limit, duplicate, stale media or MIME); do not guess at invalid input. |
| [FLUTTER-APP-29](https://craftsky.sentry.io/issues/155402329/) | social.craftsky.app@1.8.0+13 | Diagnostics | No proven app-level root cause. Retain scheduler/lifecycle/accessibility state, safe navigation history, native stacks and underlying provider causes. |
| [FLUTTER-APP-2A](https://craftsky.sentry.io/issues/155414830/) | social.craftsky.app@1.9.0+14 | Diagnostics | Preserve sanitized current/crashed thread frames and symbolication addresses; strip names, registers, unknown payloads, locals and source. |
| [FLUTTER-APP-2B](https://craftsky.sentry.io/issues/155541268/) | social.craftsky.app@1.10.0+15 | Diagnostics | Retain known native picker error codes without platform prose or details; preserve original stack. |
| [FLUTTER-APP-2C](https://craftsky.sentry.io/issues/155641454/) | social.craftsky.app@1.8.0+13 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-2D](https://craftsky.sentry.io/issues/155674207/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | Retired HTTP adapters cancel delayed dispatch/in-flight failures; active-client errors remain reportable with original causes. |
| [FLUTTER-APP-2E](https://craftsky.sentry.io/issues/155681231/) | social.craftsky.app@1.8.0+13 | Diagnostics | Retain draft failure reason and closed validation detail (image limit, duplicate, stale media or MIME); do not guess at invalid input. |
| [FLUTTER-APP-2F](https://craftsky.sentry.io/issues/155712037/) | social.craftsky.app@1.8.0+13 | Fixed / diagnostics | Retired HTTP adapters cancel delayed dispatch/in-flight failures; active-client errors remain reportable with original causes. |
| [FLUTTER-APP-2G](https://craftsky.sentry.io/issues/155856238/) | social.craftsky.app@1.8.0+13 | Diagnostics | No proven app-level root cause. Retain scheduler/lifecycle/accessibility state, safe navigation history, native stacks and underlying provider causes. |
| [FLUTTER-APP-2H](https://craftsky.sentry.io/issues/155865079/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | Retired HTTP adapters cancel delayed dispatch/in-flight failures; active-client errors remain reportable with original causes. |
| [FLUTTER-APP-2J](https://craftsky.sentry.io/issues/156023954/) | social.craftsky.app@1.9.0+14 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-2M](https://craftsky.sentry.io/issues/156078715/) | social.craftsky.app@1.9.0+14 | Diagnostics | No proven app-level root cause. Retain scheduler/lifecycle/accessibility state, safe navigation history, native stacks and underlying provider causes. |
| [FLUTTER-APP-2N](https://craftsky.sentry.io/issues/156078716/) | social.craftsky.app@1.9.0+14 | Diagnostics | No proven app-level root cause. Retain scheduler/lifecycle/accessibility state, safe navigation history, native stacks and underlying provider causes. |
| [FLUTTER-APP-2P](https://craftsky.sentry.io/issues/156186868/) | social.craftsky.app@1.9.0+14 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-2Q](https://craftsky.sentry.io/issues/156188515/) | social.craftsky.app@1.9.0+14 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-2V](https://craftsky.sentry.io/issues/156745124/) | social.craftsky.app@1.9.0+14 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-2W](https://craftsky.sentry.io/issues/156890936/) | social.craftsky.app@1.9.0+14 | Existing fix | Expired persisted deletion intents can be replaced; existing stage diagnostics and tests cover retry. |
| [FLUTTER-APP-2X](https://craftsky.sentry.io/issues/156891520/) | social.craftsky.app@1.9.0+14 | Diagnostics | No proven app-level root cause. Retain scheduler/lifecycle/accessibility state, safe navigation history, native stacks and underlying provider causes. |
| [FLUTTER-APP-2Y](https://craftsky.sentry.io/issues/156928861/) | social.craftsky.app@1.9.0+14 | Diagnostics | Retain known native picker error codes without platform prose or details; preserve original stack. |
| [FLUTTER-APP-2Z](https://craftsky.sentry.io/issues/157069913/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-30](https://craftsky.sentry.io/issues/157070064/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-31](https://craftsky.sentry.io/issues/157105310/) | social.craftsky.app@1.10.0+15 | Existing fix | Context menu captures the root navigator before the source row disappears; existing sheet tests cover the path. |
| [FLUTTER-APP-32](https://craftsky.sentry.io/issues/157255481/) | social.craftsky.app@1.11.1+17 | Fixed / diagnostics | Existing build audience guard plus pagination lifetime check prevent stale overlay work; native provider causes now survive filtering. |
| [FLUTTER-APP-33](https://craftsky.sentry.io/issues/157435480/) | social.craftsky.app@1.10.0+15 | Fixed | Skip feedback for singular transforms or detached boxes; preserve scrolling, timing and reduced motion. |
| [FLUTTER-APP-34](https://craftsky.sentry.io/issues/157468155/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | Retired HTTP adapters cancel delayed dispatch/in-flight failures; active-client errors remain reportable with original causes. |
| [FLUTTER-APP-35](https://craftsky.sentry.io/issues/157468159/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | Retired HTTP adapters cancel delayed dispatch/in-flight failures; active-client errors remain reportable with original causes. |
| [FLUTTER-APP-36](https://craftsky.sentry.io/issues/157501822/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-37](https://craftsky.sentry.io/issues/157502687/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-38](https://craftsky.sentry.io/issues/157508058/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-39](https://craftsky.sentry.io/issues/157508115/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-3A](https://craftsky.sentry.io/issues/157513596/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-3B](https://craftsky.sentry.io/issues/157774927/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-3C](https://craftsky.sentry.io/issues/157796962/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | Existing build audience guard plus pagination lifetime check prevent stale overlay work; native provider causes now survive filtering. |
| [FLUTTER-APP-3D](https://craftsky.sentry.io/issues/157796993/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | Existing build audience guard plus pagination lifetime check prevent stale overlay work; native provider causes now survive filtering. |
| [FLUTTER-APP-3E](https://craftsky.sentry.io/issues/157801494/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-3F](https://craftsky.sentry.io/issues/157801826/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | No-account/loading language policy is expected cancellation and rebuilds with initialization; genuine provider failures retain underlying causes. |
| [FLUTTER-APP-3G](https://craftsky.sentry.io/issues/157801827/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | No-account/loading language policy is expected cancellation and rebuilds with initialization; genuine provider failures retain underlying causes. |
| [FLUTTER-APP-3H](https://craftsky.sentry.io/issues/157803392/) | social.craftsky.app@1.10.0+15 | Diagnostics | No proven app-level root cause. Retain scheduler/lifecycle/accessibility state, safe navigation history, native stacks and underlying provider causes. |
| [FLUTTER-APP-3J](https://craftsky.sentry.io/issues/157803394/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | No-account/loading language policy is expected cancellation and rebuilds with initialization; genuine provider failures retain underlying causes. |
| [FLUTTER-APP-3K](https://craftsky.sentry.io/issues/157803399/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | No-account/loading language policy is expected cancellation and rebuilds with initialization; genuine provider failures retain underlying causes. |
| [FLUTTER-APP-3M](https://craftsky.sentry.io/issues/157804844/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | No-account/loading language policy is expected cancellation and rebuilds with initialization; genuine provider failures retain underlying causes. |
| [FLUTTER-APP-3N](https://craftsky.sentry.io/issues/157807344/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-3P](https://craftsky.sentry.io/issues/157807842/) | social.craftsky.app@1.11.0+16 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-3Q](https://craftsky.sentry.io/issues/157819141/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | Existing build audience guard plus pagination lifetime check prevent stale overlay work; native provider causes now survive filtering. |
| [FLUTTER-APP-3R](https://craftsky.sentry.io/issues/157819163/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | Existing build audience guard plus pagination lifetime check prevent stale overlay work; native provider causes now survive filtering. |
| [FLUTTER-APP-3S](https://craftsky.sentry.io/issues/157821024/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-3T](https://craftsky.sentry.io/issues/157821032/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-3V](https://craftsky.sentry.io/issues/157821068/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-3W](https://craftsky.sentry.io/issues/157841266/) | social.craftsky.app@1.10.0+15 | Diagnostics | No proven app-level root cause. Retain scheduler/lifecycle/accessibility state, safe navigation history, native stacks and underlying provider causes. |
| [FLUTTER-APP-3X](https://craftsky.sentry.io/issues/157841268/) | social.craftsky.app@1.10.0+15 | Diagnostics | No proven app-level root cause. Retain scheduler/lifecycle/accessibility state, safe navigation history, native stacks and underlying provider causes. |
| [FLUTTER-APP-3Y](https://craftsky.sentry.io/issues/157848801/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | No-account/loading language policy is expected cancellation and rebuilds with initialization; genuine provider failures retain underlying causes. |
| [FLUTTER-APP-3Z](https://craftsky.sentry.io/issues/157848802/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | No-account/loading language policy is expected cancellation and rebuilds with initialization; genuine provider failures retain underlying causes. |
| [FLUTTER-APP-40](https://craftsky.sentry.io/issues/157848804/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | No-account/loading language policy is expected cancellation and rebuilds with initialization; genuine provider failures retain underlying causes. |
| [FLUTTER-APP-41](https://craftsky.sentry.io/issues/157878159/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-42](https://craftsky.sentry.io/issues/157999482/) | social.craftsky.app@1.11.1+17 | Fixed / diagnostics | Retired HTTP adapters cancel delayed dispatch/in-flight failures; active-client errors remain reportable with original causes. |
| [FLUTTER-APP-43](https://craftsky.sentry.io/issues/158033338/) | social.craftsky.app@1.11.1+17 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-44](https://craftsky.sentry.io/issues/158035041/) | social.craftsky.app@1.11.1+17 | Fixed / diagnostics | Retired HTTP adapters cancel delayed dispatch/in-flight failures; active-client errors remain reportable with original causes. |
| [FLUTTER-APP-45](https://craftsky.sentry.io/issues/158037087/) | social.craftsky.app@1.11.1+17 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-46](https://craftsky.sentry.io/issues/158073272/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | No-account/loading language policy is expected cancellation and rebuilds with initialization; genuine provider failures retain underlying causes. |
| [FLUTTER-APP-47](https://craftsky.sentry.io/issues/158126913/) | social.craftsky.app@1.9.0+14 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-48](https://craftsky.sentry.io/issues/158151479/) | social.craftsky.app@1.11.1+17 | Fixed | Skip feedback for singular transforms or detached boxes; preserve scrolling, timing and reduced motion. |
| [FLUTTER-APP-49](https://craftsky.sentry.io/issues/158285941/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | Existing build audience guard plus pagination lifetime check prevent stale overlay work; native provider causes now survive filtering. |
| [FLUTTER-APP-4A](https://craftsky.sentry.io/issues/158360170/) | social.craftsky.app@1.10.0+15 | Fixed / diagnostics | Existing build audience guard plus pagination lifetime check prevent stale overlay work; native provider causes now survive filtering. |
| [FLUTTER-APP-4B](https://craftsky.sentry.io/issues/158377558/) | social.craftsky.app@1.11.1+17 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-4C](https://craftsky.sentry.io/issues/158832020/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-4D](https://craftsky.sentry.io/issues/158832021/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-4E](https://craftsky.sentry.io/issues/158832022/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |
| [FLUTTER-APP-4F](https://craftsky.sentry.io/issues/158832025/) | social.craftsky.app@1.10.0+15 | Diagnostics / retained evidence | Retain response kind, HTTP method/status, stable API code and valid AppView request UUID, plus new lifecycle/provider cause evidence. Upstream/edge failure is not a proven client defect. |

## Validation and Sentry state

- Complete Flutter suite: 3,971 passed, 44 skipped.
- Final affected diagnostics/API/language/feedback suites: 130 passed, one skip.
- Additional serialized runtime/thread/privacy assertions: eight passed.
- Flutter analysis: no issues. `git diff --check`: passed.
- Original retained paint callback with a singular transform fails with
  `ArgumentError` at `Matrix4.inverted`; the corrected widget suite passes.
- Complete AppView release gate: passed. Integration run: 7,246 cases;
  final race run: 7,261 cases; zero test skips or failures. Module/codegen,
  vet/Staticcheck, source/binary vulnerability scans, production/media image
  builds, migration checks, media memory checks and service smoke checks passed.
  Pinned Tap handshake passed; no deterministic Tap events were injected.
  Transcript: `/tmp/craftsky-sentry-appview-check-final.log`.
  Artifacts: `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.yRiLxQLFta/`.
  The initial run found a new fixture using the sensitive label `token_exchange`;
  production uses `finalization`. That fixture was corrected before the final
  gate. Serialized local/Sentry checks for all 14 new chat stages also pass.
- All 101 reviewed production issues (78 Flutter, 23 AppView) were resolved using
  `status=resolved`, `statusDetails.inNextRelease=true`, `substatus=null` through
  the Sentry CLI's authenticated project API. Each issue independently shows a
  latest `set_resolved_in_release` activity with empty target `version` and a
  current-release anchor. Sentry's issue readback expresses that anchor as
  `inRelease`; the empty activity target distinguishes next-release resolution.
- Fresh production queue verification: zero unresolved Flutter issues and zero
  unresolved AppView issues, with no further pages. The unfiltered AppView queue
  still contains exactly the six excluded dev-only issues.
- No running DTD/Flutter debug session was available for hot reload. Physical
  iOS/Android validation, live post-release delivery/symbolication, deployment
  and recurrence verification remain unexecuted. No synthetic production errors
  were injected. Changes remain local and uncommitted.
