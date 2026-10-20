import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:material_ui/material_ui.dart';

class ChatSyncTimestamp extends StatelessWidget {
  const new({required this.at, super.key});
  final DateTime? at;
  @override
  Widget build(BuildContext context) {
    final date = at?.toLocal();
    if (date == null) return const SizedBox.shrink();
    final material = MaterialLocalizations.of(context);
    final dateLabel = material.formatCompactDate(date);
    final timeLabel = material.formatTimeOfDay(TimeOfDay.fromDateTime(date));
    final label = '$dateLabel $timeLabel';
    return Text(
      AppLocalizations.of(context).chatLastUpdated(label),
      style: Theme.of(context).textTheme.bodySmall,
    );
  }
}
