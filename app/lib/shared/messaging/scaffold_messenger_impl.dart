import 'package:craftsky_app/shared/messaging/app_messenger.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// The root ScaffoldMessenger key. Wired into every `MaterialApp` in
/// `app/lib/app.dart` so messages always go through the same messenger
/// regardless of which subtree the call site lives in.
final GlobalKey<ScaffoldMessengerState> appScaffoldMessengerKey =
    GlobalKey<ScaffoldMessengerState>();

/// The default production [AppMessenger]. Constructed once and reused —
/// the [GlobalKey] is the only piece of mutable state and Flutter owns it.
/// Not `const` because `GlobalKey()` is not a const expression.
final AppMessenger defaultAppMessenger = ScaffoldMessengerImpl(
  appScaffoldMessengerKey,
);

/// `AppMessenger` backed by Flutter's `ScaffoldMessenger`. Routes every
/// message through the global [appScaffoldMessengerKey] (never via
/// `ScaffoldMessenger.of(context)`), enforces the always-replace policy
/// (each call clears any current message before showing the new one), and
/// supplies the lifetime semantics declared on [AppMessenger].
class ScaffoldMessengerImpl implements AppMessenger {
  ScaffoldMessengerImpl(this._key);

  final GlobalKey<ScaffoldMessengerState> _key;

  static const Duration _infoDuration = Duration(seconds: 4);
  static const Duration _stickyDuration = Duration(days: 365);

  @override
  void info(String message, {MessageAction? action}) {
    _show(
      MessageSeverity.info,
      message,
      action,
      _infoDuration,
      sticky: false,
    );
  }

  @override
  void warning(String message, {MessageAction? action}) {
    _show(
      MessageSeverity.warning,
      message,
      action,
      _stickyDuration,
      sticky: true,
    );
  }

  @override
  void error(String message, {MessageAction? action}) {
    _show(
      MessageSeverity.error,
      message,
      action,
      _stickyDuration,
      sticky: true,
    );
  }

  @override
  void dismiss() {
    _key.currentState?.hideCurrentSnackBar();
  }

  void _show(
    MessageSeverity severity,
    String message,
    MessageAction? action,
    Duration duration, {
    required bool sticky,
  }) {
    final state = _key.currentState;
    if (state == null) return;

    final context = _key.currentContext;
    final semantic = context == null
        ? null
        : Theme.of(context).extension<SemanticColorsTheme>();
    final backgroundColor = semantic == null
        ? null
        : switch (severity) {
            MessageSeverity.info => semantic.infoSurface,
            MessageSeverity.warning => semantic.warningSurface,
            MessageSeverity.error => semantic.errorSurface,
          };

    state.clearSnackBars();

    final wrappedAction = action == null
        ? null
        : MessageAction(
            label: action.label,
            dismissOnTap: action.dismissOnTap,
            onPressed: () {
              if (action.dismissOnTap) {
                state.hideCurrentSnackBar();
              }
              action.onPressed();
            },
          );

    state.showSnackBar(
      SnackBar(
        behavior: SnackBarBehavior.floating,
        dismissDirection: DismissDirection.horizontal,
        duration: duration,
        backgroundColor: backgroundColor,
        content: CraftskySnackBarContent(
          severity: severity,
          message: message,
          action: wrappedAction,
          onDismiss: sticky ? state.hideCurrentSnackBar : null,
        ),
      ),
    );
  }
}
