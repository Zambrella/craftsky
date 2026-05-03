import 'package:craftsky_app/shared/messaging/app_messenger.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';

part 'message_action.mapper.dart';

/// An optional action button shown alongside a message dispatched through
/// [AppMessenger]. `dismissOnTap` controls whether tapping the button
/// dismisses the message — defaults to `true`, matching Material's
/// `SnackBarAction`. Set it to `false` for actions whose effect should
/// leave the message in place (e.g. a "Retry" that triggers an async
/// operation and lets the same message reflect the next outcome).
@MappableClass()
class MessageAction with MessageActionMappable {
  const MessageAction({
    required this.label,
    required this.onPressed,
    this.dismissOnTap = true,
  });

  final String label;
  final VoidCallback onPressed;
  final bool dismissOnTap;
}
