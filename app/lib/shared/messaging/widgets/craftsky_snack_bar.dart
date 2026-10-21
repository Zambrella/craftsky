import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

/// Severity levels surfaced by `AppMessenger`. Drives the leading icon and
/// (in the impl) the snackbar's lifetime.
enum MessageSeverity { info, warning, error }

/// The visual payload of every message dispatched through `AppMessenger`.
/// Shares `[icon · text · action? · close?]` with persistent banners, moving
/// the action below the message on narrow screens or at larger text sizes.
///
/// `onDismiss` controls whether a trailing close icon is rendered:
/// `null` → no close icon (info messages don't need one because they
/// auto-dismiss); non-null → close icon visible, and tapping it invokes
/// `onDismiss` (which the impl wires to
/// `messengerState.hideCurrentSnackBar()`).
class CraftskySnackBarContent extends StatelessWidget {
  const new({
    required this.severity,
    required this.message,
    this.action,
    this.actionEnabled = true,
    this.onDismiss,
    super.key,
  });

  final MessageSeverity severity;
  final String message;
  final MessageAction? action;
  final bool actionEnabled;
  final VoidCallback? onDismiss;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final semantic = theme.extension<SemanticColorsTheme>()!;

    return LayoutBuilder(
      builder: (context, constraints) {
        final stackAction =
            action != null &&
            (constraints.maxWidth < 320 ||
                MediaQuery.textScalerOf(context).scale(14) > 20);
        final notice = Row(
          children: [
            Icon(
              _iconFor(severity),
              size: 20,
              // The light warning surface is itself butter yellow.
              color:
                  severity == MessageSeverity.warning &&
                      theme.brightness == Brightness.light
                  ? theme.colorScheme.onSurface
                  : _colorFor(severity, semantic),
            ),
            SizedBox(width: spacing.sp3),
            Expanded(
              child: Text(message, style: theme.textTheme.bodyMedium),
            ),
            if (action != null && !stackAction) ...[
              SizedBox(width: spacing.sp2),
              _MessageActionButton(action: action!, enabled: actionEnabled),
            ],
            if (onDismiss != null) ...[
              SizedBox(width: spacing.sp2),
              IconButton(
                icon: const Icon(CraftskyIconsBold.close, size: 20),
                tooltip: l10n.messengerDismiss,
                onPressed: onDismiss,
              ),
            ],
          ],
        );
        if (!stackAction) return notice;
        return Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            notice,
            SizedBox(height: spacing.sp2),
            Align(
              alignment: AlignmentDirectional.centerEnd,
              child: _MessageActionButton(
                action: action!,
                enabled: actionEnabled,
              ),
            ),
          ],
        );
      },
    );
  }

  static IconData _iconFor(MessageSeverity s) => switch (s) {
    MessageSeverity.info => CraftskyIcons.info,
    MessageSeverity.warning => CraftskyIcons.warning,
    MessageSeverity.error => CraftskyIcons.error,
  };

  static Color _colorFor(MessageSeverity s, SemanticColorsTheme c) =>
      switch (s) {
        MessageSeverity.info => c.info,
        MessageSeverity.warning => c.warning,
        MessageSeverity.error => c.error,
      };
}

class _MessageActionButton extends StatelessWidget {
  const new({required this.action, required this.enabled});

  final MessageAction action;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return TextButton(
      onPressed: enabled ? action.onPressed : null,
      child: Text(action.label, style: theme.textTheme.labelLarge),
    );
  }
}
