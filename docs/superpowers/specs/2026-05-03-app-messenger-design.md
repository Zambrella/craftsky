# App Messenger

## Summary

Replace scattered `ScaffoldMessenger.of(context).showSnackBar(...)` calls with a small, swappable `AppMessenger` interface that exposes three semantic methods — `info`, `warning`, `error` — each accepting an optional `MessageAction`. Info auto-dismisses; warning and error are sticky until dismissed by the user. Consumers call it via a `BuildContext` extension (`context.showInfo(...)`). The default implementation is backed by `ScaffoldMessenger` and a `GlobalKey<ScaffoldMessengerState>` attached to `MaterialApp.scaffoldMessengerKey`; the interface lets us swap that for a custom overlay later without touching call sites.

## Why now

Four files reach into `ScaffoldMessenger` directly today:

- `app/lib/settings/widgets/clear_image_cache_tile.dart`
- `app/lib/auth/pages/sign_in_page.dart`
- `app/lib/profile/pages/profile_page.dart`
- `app/lib/profile/pages/edit_profile_dialog.dart`

Each constructs a bare `SnackBar(content: Text(...))` with no consistent styling, no semantic distinction between informational and error messages, and no way for an error message to remain visible until the user acknowledges it. `sign_in_page.dart` already manually calls `clearSnackBars()` before `showSnackBar(...)` to enforce a replace policy — a pattern other call sites would need to repeat.

A central abstraction gives us:
- One place to enforce "info auto-times out, warning/error stick until dismissed".
- One place to apply consistent visual treatment (semantic-coloured leading icon, optional action, close affordance).
- A test seam: widget tests for the four call sites can override the messenger and assert intent (`info('Saved')` was called) instead of poking SnackBar internals.
- A migration path: switching to a custom overlay later changes the impl; consumers stay unchanged.

## Non-goals (v1)

- **Context-free messaging.** All current call sites have `BuildContext` in scope. No support for surfacing messages from notifiers/services without context. The interface stays compatible with adding that later (the impl already uses a global key, not `ScaffoldMessenger.of`).
- **Stacked messages.** ScaffoldMessenger shows one at a time; the always-replace policy means we never queue. Stacking would require the custom-overlay impl, which is out of scope.
- **Priority routing** (e.g. errors superseding info while letting info supersede info). Always-replace is the only policy.
- **Localisation of action labels and dismiss affordances.** Messages themselves are already produced by call sites (often via `l10n`); the messenger does not own copy. The only string the messenger owns is the close icon's `Semantics` label, which we localise.
- **Multiple actions per message.** A single optional action only.
- **Theming hooks beyond what `SemanticColorsTheme` already provides.**

## Architecture

### File layout

```
app/lib/shared/messaging/
  app_messenger.dart                   # abstract interface
  message_action.dart                  # @freezed value object
  message_action.freezed.dart          # generated
  scaffold_messenger_impl.dart         # default impl + global key
  app_messenger_provider.dart          # @riverpod provider
  app_messenger_provider.g.dart        # generated
  context_messenger_extension.dart     # BuildContext extension
  widgets/
    craftsky_snack_bar.dart            # [icon · text · action? · close?] layout
```

### Interface

```dart
// app_messenger.dart
abstract interface class AppMessenger {
  void info(String message, {MessageAction? action});
  void warning(String message, {MessageAction? action});
  void error(String message, {MessageAction? action});

  /// Dismisses the currently visible message, if any. No-op otherwise.
  void dismiss();
}
```

### Action value object

```dart
// message_action.dart
@freezed
class MessageAction with _$MessageAction {
  const factory MessageAction({
    required String label,
    required VoidCallback onPressed,
    @Default(true) bool dismissOnTap,
  }) = _MessageAction;
}
```

`dismissOnTap` defaults to `true` (matches Material's `SnackBarAction`). Consumers set it to `false` for actions whose effect should leave the snackbar in place — e.g. a "Retry" that triggers an async operation and lets the same snackbar reflect the next outcome.

### Provider

```dart
// app_messenger_provider.dart
@Riverpod(keepAlive: true)
AppMessenger appMessenger(Ref ref) => ScaffoldMessengerImpl(appScaffoldMessengerKey);
```

`keepAlive: true` because the messenger is a singleton for the app's lifetime; there's no scenario where disposing it makes sense.

### `BuildContext` extension

```dart
// context_messenger_extension.dart
extension AppMessengerX on BuildContext {
  void showInfo(String message, {MessageAction? action}) =>
      _resolve(this).info(message, action: action);
  void showWarning(String message, {MessageAction? action}) =>
      _resolve(this).warning(message, action: action);
  void showError(String message, {MessageAction? action}) =>
      _resolve(this).error(message, action: action);
  void dismissMessage() => _resolve(this).dismiss();
}

AppMessenger _resolve(BuildContext context) =>
    ProviderScope.containerOf(context).read(appMessengerProvider);
```

Reaching the provider via `ProviderScope.containerOf(context)` keeps the call site free of `WidgetRef` while still routing through Riverpod, so test overrides on `appMessengerProvider` are honoured.

### Global key wiring

```dart
// scaffold_messenger_impl.dart
final appScaffoldMessengerKey = GlobalKey<ScaffoldMessengerState>();

class ScaffoldMessengerImpl implements AppMessenger {
  ScaffoldMessengerImpl(this._key);
  final GlobalKey<ScaffoldMessengerState> _key;
  // ...
}
```

`main.dart` (or wherever `MaterialApp` is constructed) passes `appScaffoldMessengerKey` to `MaterialApp.scaffoldMessengerKey`. The impl never calls `ScaffoldMessenger.of(context)`; every operation goes through `_key.currentState`. This centralises messages on the root messenger regardless of which subtree the call site lives in, and is what makes future context-free callers possible without touching the interface.

## Behaviour

### Lifetime

| Severity | Duration | Auto-dismiss? | Close affordance |
|---|---|---|---|
| info | 4 seconds | yes | none |
| warning | indefinite (`Duration(days: 365)`) | no | trailing close icon |
| error | indefinite (`Duration(days: 365)`) | no | trailing close icon |

The 4-second info default matches Material's `SnackBar` default. The "indefinite" sentinel for sticky messages is the standard Flutter idiom; the impl never relies on the duration actually elapsing, only on the user (or another `info`/`warning`/`error` call) ending the message.

### Always-replace policy

Every call to `info`/`warning`/`error` first invokes `messengerState.clearSnackBars()`, then `showSnackBar(...)`. This:

- Removes the manual `clearSnackBars()` dance from `sign_in_page.dart`.
- Prevents the failure mode where a sticky error blocks every subsequent message indefinitely (which would be the case under stock FIFO queueing).
- Is invisible to consumers — they call `context.showError(...)` and trust that it appears.

### Action wrapping

When an action is provided, the impl wraps the consumer's `onPressed` so that — if `dismissOnTap` is true — `messengerState.hideCurrentSnackBar()` runs immediately before the consumer callback. This lets the consumer write `MessageAction(label: 'Retry', onPressed: _retry)` without each call site repeating the dismiss call.

### Dismissal paths

For sticky (warning/error) messages, the user can dismiss via:
1. Tapping the trailing close icon in `CraftskySnackBarContent`.
2. Tapping the action button (when one is provided and `dismissOnTap` is true).
3. Swiping the snackbar horizontally (`DismissDirection.horizontal`).
4. A subsequent `info`/`warning`/`error`/`dismiss` call (always-replace).

Info messages additionally dismiss themselves after 4 seconds.

## Visual treatment

### `CraftskySnackBarContent`

The snackbar's `content` is a single `CraftskySnackBarContent` widget rather than a bare `Text`. It owns the row:

```
[ leading icon ][ message text ][ action TextButton? ][ close icon? ]
```

- **Leading icon** is always rendered, sized 20px, tinted from `SemanticColorsTheme`:
  - info → `Icons.info_outline` in `semanticColors.info`
  - warning → `Icons.warning_amber_rounded` in `semanticColors.warning`
  - error → `Icons.error_outline` in `semanticColors.error`
- **Message text** uses `Theme.of(context).textTheme.bodyMedium`. No truncation; it wraps.
- **Action button** is a `TextButton` rendered only when `MessageAction` is non-null. The label uses `textTheme.labelLarge`.
- **Close icon** is `Icons.close` rendered only for warning/error severity, regardless of whether an action is supplied. Tapping it calls `dismiss()`. Its `Semantics.label` is the localised "Dismiss" string from `l10n`.

Spacing between elements pulls from `SpacingTheme` (`sp2` between text and trailing items, `sp3` between the leading icon and text). The SnackBar's `backgroundColor` stays Material default — the semantic colour appears via the leading icon, not the surface, so contrast against the message text is unaffected.

### SnackBar configuration

The default impl produces SnackBars with:
- `behavior: SnackBarBehavior.floating`
- `dismissDirection: DismissDirection.horizontal`
- `action: null` (the action button is rendered inside `content` so the layout can include the close icon and we keep full control of ordering)
- `duration` per the lifetime table above.

## Migration

Four call sites migrate. Each becomes a one-line change at the call site (the existing message construction is unchanged).

| File | Before | After |
|---|---|---|
| `clear_image_cache_tile.dart` | `ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Image cache cleared')))` | `context.showInfo('Image cache cleared')` |
| `clear_image_cache_tile.dart` | `ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Could not clear cache: $error')))` | `context.showError('Could not clear cache: $error')` |
| `sign_in_page.dart` | `ScaffoldMessenger.of(context)..clearSnackBars()..showSnackBar(SnackBar(content: Text(message)))` | `context.showError(message)` |
| `profile_page.dart` (×2) | `ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(l10n.profileFollowComingSoon)))` | `context.showInfo(l10n.profileFollowComingSoon)` |
| `edit_profile_dialog.dart` | `ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(l10n.editProfileSaveError)))` | `context.showError(l10n.editProfileSaveError)` |

Behavioural deltas worth noting:
- `clear_image_cache_tile.dart`'s error path becomes **sticky**. This is intentional — under the old behaviour the error vanished after 4 seconds whether or not the user noticed it.
- `edit_profile_dialog.dart`'s save error becomes **sticky** for the same reason.
- `sign_in_page.dart` loses the manual `clearSnackBars()` call (now redundant under the always-replace policy).

`app/lib/app.dart` (which constructs the three `MaterialApp` / `MaterialApp.router` instances at lines 47, 72, and 88) gains one line on each: `scaffoldMessengerKey: appScaffoldMessengerKey`. The fallback `MaterialApp` instances (the splash and error states above the router) get the same key so messages still appear during early-app failure paths.

## Testing

### Default-impl tests

`test/shared/messaging/scaffold_messenger_impl_test.dart`:

- Shows a `SnackBar` whose duration is 4 seconds for `info`.
- Shows a `SnackBar` whose duration is `Duration(days: 365)` for `warning` and `error`.
- A second `info`/`warning`/`error` call replaces the first (i.e. only one `SnackBar` is on screen at any time).
- Action `onPressed` runs on tap; with `dismissOnTap: true` the snackbar is gone afterwards; with `dismissOnTap: false` the snackbar remains.
- The trailing close icon is rendered for warning/error and not for info, regardless of whether an action is provided.
- Tapping the close icon dismisses the message.

These tests pump a small `MaterialApp` whose `scaffoldMessengerKey` is the same key the impl was constructed with, then drive the impl directly (no widget under test needs to consume the messenger).

### Call-site tests

A `RecordingMessenger` test double in `test/shared/messaging/recording_messenger.dart`:

```dart
class RecordingMessenger implements AppMessenger {
  final calls = <(String severity, String message, MessageAction? action)>[];
  @override void info(String m, {MessageAction? action}) => calls.add(('info', m, action));
  @override void warning(String m, {MessageAction? action}) => calls.add(('warning', m, action));
  @override void error(String m, {MessageAction? action}) => calls.add(('error', m, action));
  @override void dismiss() => calls.add(('dismiss', '', null));
}
```

Call-site tests override `appMessengerProvider` with a `RecordingMessenger` and assert against `calls`. The existing `clear_image_cache_tile_test.dart` is updated to use this pattern; tests for the other three migrated call sites are out of scope for this spec (they can adopt the recording fake when they're written).

### Riverpod override pattern

```dart
ProviderScope(
  overrides: [appMessengerProvider.overrideWith((ref) => recording)],
  child: …,
);
```

The repo's Riverpod rules reserve `overrideWithValue` for `FutureProvider` / `StreamProvider` (per Riverpod 3.0); synchronous function providers like `appMessengerProvider` use `overrideWith((ref) => …)`.

## Open questions

None blocking implementation.
