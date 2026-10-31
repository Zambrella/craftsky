import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:intl/intl.dart';
import 'package:material_ui/material_ui.dart';

bool startsChatTimeSection(DateTime current, DateTime? previous) =>
    previous == null ||
    !_sameDay(current.toLocal(), previous.toLocal()) ||
    current.difference(previous) >= const Duration(minutes: 5);

bool _sameDay(DateTime a, DateTime b) =>
    a.year == b.year && a.month == b.month && a.day == b.day;

/// A time section outside the bubble; only authoritative messages supply dates.
class ChatTimeHeader extends StatelessWidget {
  const new({required this.timestamp, this.now, super.key});

  final DateTime timestamp;
  final DateTime? now;

  @override
  Widget build(BuildContext context) {
    final local = timestamp.toLocal();
    final today = (now ?? DateTime.now()).toLocal();
    final yesterday = DateTime(today.year, today.month, today.day - 1);
    final l10n = AppLocalizations.of(context);
    final time = MaterialLocalizations.of(context).formatTimeOfDay(
      TimeOfDay.fromDateTime(local),
      alwaysUse24HourFormat: MediaQuery.alwaysUse24HourFormatOf(context),
    );
    final label = _sameDay(local, today)
        ? l10n.chatTimeToday(time)
        : _sameDay(local, yesterday)
        ? l10n.chatTimeYesterday(time)
        : l10n.chatTimeDate(
            DateFormat(
              local.year == today.year ? 'EEE d MMMM' : 'EEE d MMMM y',
              Localizations.localeOf(context).toString(),
            ).format(local),
            time,
          );
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Padding(
      padding: EdgeInsets.fromLTRB(spacing.sp3, 0, spacing.sp3, spacing.sp3),
      child: Center(
        child: Semantics(
          header: true,
          child: Text(
            label,
            textAlign: TextAlign.center,
            style: theme.textTheme.labelSmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
              fontWeight: FontWeight.w400,
              letterSpacing: 0,
            ),
          ),
        ),
      ),
    );
  }
}
