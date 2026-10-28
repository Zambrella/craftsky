import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:material_ui/material_ui.dart';

/// Compact relative timestamp used by posts and other chronological surfaces.
///
/// The visible value stays deliberately terse (`now`, `3m`, `4h`, `2d`) while
/// the tooltip preserves the complete localized timestamp. Edited posts include
/// their disclosure and include both timestamps in the same tooltip.
class RelativeTimeText extends StatefulWidget {
  const new({
    required this.timestamp,
    this.now,
    this.editedAt,
    this.style,
    this.textAlign,
    super.key,
  });

  final DateTime timestamp;
  final DateTime? now;
  final DateTime? editedAt;
  final TextStyle? style;
  final TextAlign? textAlign;

  @override
  State<RelativeTimeText> createState() => _RelativeTimeTextState();
}

class _RelativeTimeTextState extends State<RelativeTimeText> {
  final _tooltipKey = GlobalKey<TooltipState>();

  void _showTimestamp() => _tooltipKey.currentState?.ensureTooltipVisible();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final style =
        widget.style ??
        theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.outline,
        );
    final relative = _relativeTime(
      widget.timestamp,
      widget.now ?? DateTime.now(),
    );
    final published = _fullTimestamp(context, widget.timestamp);
    final editedAt = widget.editedAt;
    if (editedAt == null) {
      return Tooltip(
        message: published,
        child: Text(relative, style: style, textAlign: widget.textAlign),
      );
    }
    final l10n = AppLocalizations.of(context);
    final message =
        '$published\n${l10n.postEditTime(_fullTimestamp(context, editedAt))}';
    return Semantics(
      label: message,
      excludeSemantics: true,
      button: true,
      onTap: _showTimestamp,
      child: FocusableActionDetector(
        onFocusChange: (focused) {
          if (focused) _showTimestamp();
        },
        actions: {
          ActivateIntent: CallbackAction<ActivateIntent>(
            onInvoke: (_) {
              _showTimestamp();
              return null;
            },
          ),
        },
        child: Tooltip(
          key: _tooltipKey,
          message: message,
          triggerMode: TooltipTriggerMode.longPress,
          excludeFromSemantics: true,
          child: GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: _showTimestamp,
            child: Text.rich(
              TextSpan(
                children: [
                  WidgetSpan(
                    alignment: PlaceholderAlignment.middle,
                    child: Icon(
                      CraftskyIcons.edit,
                      size: 12,
                      color: style?.color,
                    ),
                  ),
                  TextSpan(text: ' $relative'),
                ],
              ),
              style: style,
              textAlign: widget.textAlign,
            ),
          ),
        ),
      ),
    );
  }
}

String _fullTimestamp(BuildContext context, DateTime when) {
  final local = when.toLocal();
  final material = MaterialLocalizations.of(context);
  final date = material.formatFullDate(local);
  final time = material.formatTimeOfDay(
    TimeOfDay.fromDateTime(local),
    alwaysUse24HourFormat: MediaQuery.alwaysUse24HourFormatOf(context),
  );
  return '$date at $time ${local.timeZoneName}';
}

String _relativeTime(DateTime when, DateTime now) {
  final delta = now.difference(when);
  if (delta.inMinutes < 1) return 'now';
  if (delta.inMinutes < 60) return '${delta.inMinutes}m';
  if (delta.inHours < 24) return '${delta.inHours}h';
  return '${delta.inDays}d';
}
