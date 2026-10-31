import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:material_ui/material_ui.dart';

/// Publishes state transitions through the existing AppMessenger after layout.
/// Adds no surface or layout and never dismisses unrelated messages.
class ChatMessageEffect extends StatefulWidget {
  const new({
    required this.message,
    this.severity = MessageSeverity.info,
    this.action,
    this.actionEnabled = true,
    this.child = const SizedBox.shrink(),
    super.key,
  });

  final String? message;
  final MessageSeverity severity;
  final MessageAction? action;
  final bool actionEnabled;
  final Widget child;

  @override
  State<ChatMessageEffect> createState() => _ChatMessageEffectState();
}

class _ChatMessageEffectState extends State<ChatMessageEffect> {
  (String, MessageSeverity)? _published;
  int _version = 0;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _schedule();
  }

  @override
  void didUpdateWidget(covariant ChatMessageEffect oldWidget) {
    super.didUpdateWidget(oldWidget);
    _schedule();
  }

  void _schedule() {
    final version = ++_version;
    final message = widget.message;
    if (message == null) {
      _published = null;
      return;
    }
    final signature = (message, widget.severity);
    if (_published == signature) return;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted ||
          version != _version ||
          !(ModalRoute.of(context)?.isCurrent ?? true)) {
        return;
      }
      _published = signature;
      final action = widget.action == null
          ? null
          : MessageAction(
              label: widget.action!.label,
              dismissOnTap: widget.action!.dismissOnTap,
              onPressed: () {
                if (mounted &&
                    widget.actionEnabled &&
                    (ModalRoute.of(context)?.isCurrent ?? true)) {
                  widget.action?.onPressed();
                }
              },
            );
      switch (widget.severity) {
        case MessageSeverity.info:
          context.showInfo(message, action: action);
        case MessageSeverity.warning:
          context.showWarning(message, action: action);
        case MessageSeverity.error:
          context.showError(message, action: action);
      }
    });
  }

  @override
  Widget build(BuildContext context) => widget.child;
}
