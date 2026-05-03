import 'package:craftsky_app/shared/messaging/app_messenger.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';

/// Test double for [AppMessenger] that records every call. Wrap the widget
/// under test in `MessengerScope(messenger: RecordingMessenger(), ...)`
/// and assert against [calls].
///
/// Each entry is `(severity, message, action)`. For `dismiss()` the
/// severity is `'dismiss'` and `message` is the empty string.
class RecordingMessenger implements AppMessenger {
  final List<(String severity, String message, MessageAction? action)> calls =
      [];

  @override
  void info(String message, {MessageAction? action}) =>
      calls.add(('info', message, action));

  @override
  void warning(String message, {MessageAction? action}) =>
      calls.add(('warning', message, action));

  @override
  void error(String message, {MessageAction? action}) =>
      calls.add(('error', message, action));

  @override
  void dismiss() => calls.add(('dismiss', '', null));
}
