import 'package:craftsky_app/shared/messaging/message_action.dart';

/// Dispatches semantic messages (snackbars in the default impl) to the user.
///
/// Three severities, three lifetime policies:
/// - [info] auto-dismisses after a short timeout.
/// - [warning] stays until the user dismisses it.
/// - [error] stays until the user dismisses it.
///
/// Each method accepts an optional [MessageAction] (e.g. "Retry"). Callers
/// reach an `AppMessenger` through `MessengerScope.of(context)`, or via the
/// `BuildContext` extension `context.showInfo(...)` etc.
abstract interface class AppMessenger {
  /// Shows a transient informational message. Auto-dismisses after ~4s.
  void info(String message, {MessageAction? action});

  /// Shows a sticky warning. Stays until the user dismisses it (close icon,
  /// swipe, action tap, or a subsequent message).
  void warning(String message, {MessageAction? action});

  /// Shows a sticky error. Same dismissal model as [warning].
  void error(String message, {MessageAction? action});

  /// Dismisses the currently visible message, if any. No-op otherwise.
  void dismiss();
}
