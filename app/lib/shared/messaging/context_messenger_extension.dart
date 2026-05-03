import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:flutter/widgets.dart';

/// Terse wrappers over `MessengerScope.of(context)`. The extension is the
/// preferred call surface for widgets — `context.showInfo('Saved')` is
/// shorter than the equivalent `MessengerScope.of(context).info('Saved')`
/// and reads naturally next to `Theme.of(context)`.
extension AppMessengerX on BuildContext {
  void showInfo(String message, {MessageAction? action}) =>
      MessengerScope.of(this).info(message, action: action);

  void showWarning(String message, {MessageAction? action}) =>
      MessengerScope.of(this).warning(message, action: action);

  void showError(String message, {MessageAction? action}) =>
      MessengerScope.of(this).error(message, action: action);

  void dismissMessage() => MessengerScope.of(this).dismiss();
}
