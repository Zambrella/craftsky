import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Severity levels surfaced by `AppMessenger`. Drives the leading icon and
/// (in the impl) the snackbar's lifetime.
enum MessageSeverity { info, warning, error }

/// The visual payload of every message dispatched through `AppMessenger`.
/// Owns the row layout `[icon · text · action? · close?]`.
///
/// `onDismiss` controls whether a trailing close icon is rendered:
/// `null` → no close icon (info messages don't need one because they
/// auto-dismiss); non-null → close icon visible, and tapping it invokes
/// `onDismiss` (which the impl wires to
/// `messengerState.hideCurrentSnackBar()`).
class CraftskySnackBarContent extends StatelessWidget {
  const CraftskySnackBarContent({
    required this.severity,
    required this.message,
    this.action,
    this.onDismiss,
    super.key,
  });

  final MessageSeverity severity;
  final String message;
  final MessageAction? action;
  final VoidCallback? onDismiss;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final semantic = theme.extension<SemanticColorsTheme>()!;

    return Row(
      children: [
        Icon(
          _iconFor(severity),
          size: 20,
          color: _colorFor(severity, semantic),
        ),
        SizedBox(width: spacing.sp3),
        Expanded(
          child: Text(message, style: theme.textTheme.bodyMedium),
        ),
        if (action != null) ...[
          SizedBox(width: spacing.sp2),
          _MessageActionButton(action: action!),
        ],
        if (onDismiss != null) ...[
          SizedBox(width: spacing.sp2),
          IconButton(
            icon: const Icon(Icons.close, size: 20),
            tooltip: l10n.messengerDismiss,
            onPressed: onDismiss,
          ),
        ],
      ],
    );
  }

  static IconData _iconFor(MessageSeverity s) => switch (s) {
    MessageSeverity.info => Icons.info_outline,
    MessageSeverity.warning => Icons.warning_amber_rounded,
    MessageSeverity.error => Icons.error_outline,
  };

  static Color _colorFor(MessageSeverity s, SemanticColorsTheme c) =>
      switch (s) {
        MessageSeverity.info => c.info,
        MessageSeverity.warning => c.warning,
        MessageSeverity.error => c.error,
      };
}

class _MessageActionButton extends StatelessWidget {
  const _MessageActionButton({required this.action});

  final MessageAction action;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return TextButton(
      onPressed: action.onPressed,
      child: Text(action.label, style: theme.textTheme.labelLarge),
    );
  }
}
