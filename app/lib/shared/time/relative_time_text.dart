import 'package:flutter/material.dart';

/// Compact relative timestamp used by posts and other chronological surfaces.
///
/// The visible value stays deliberately terse (`now`, `3m`, `4h`, `2d`) while
/// the tooltip preserves the complete localized timestamp.
class RelativeTimeText extends StatelessWidget {
  const RelativeTimeText({
    required this.timestamp,
    this.now,
    this.style,
    this.textAlign,
    super.key,
  });

  final DateTime timestamp;
  final DateTime? now;
  final TextStyle? style;
  final TextAlign? textAlign;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Tooltip(
      message: _fullTimestamp(context, timestamp),
      child: Text(
        _relativeTime(timestamp, now ?? DateTime.now()),
        style:
            style ??
            theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.outline,
            ),
        textAlign: textAlign,
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
