import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:material_ui/material_ui.dart';

/// Latest edit disclosure; it exposes no history or previous version.
class PostEditIndicator extends StatefulWidget {
  const new({super.key, this.editedAt});

  final DateTime? editedAt;

  @override
  State<PostEditIndicator> createState() => _PostEditIndicatorState();
}

class _PostEditIndicatorState extends State<PostEditIndicator> {
  final _tooltipKey = GlobalKey<TooltipState>();

  void _showTimestamp() => _tooltipKey.currentState?.ensureTooltipVisible();

  @override
  Widget build(BuildContext context) {
    final when = widget.editedAt?.toLocal();
    if (when == null) return const SizedBox.shrink();
    final material = MaterialLocalizations.of(context);
    final date = material.formatFullDate(when);
    final time = material.formatTimeOfDay(
      TimeOfDay.fromDateTime(when),
      alwaysUse24HourFormat: MediaQuery.alwaysUse24HourFormatOf(context),
    );
    final timestamp = '$date, $time ${when.timeZoneName}';
    final l10n = AppLocalizations.of(context);
    final label = l10n.postEditTime(timestamp);
    return Semantics(
      label: label,
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
          message: label,
          triggerMode: TooltipTriggerMode.tap,
          excludeFromSemantics: true,
          child: SizedBox.square(
            dimension: 24,
            child: Center(
              child: Icon(
                CraftskyIcons.edit,
                size: 14,
                color: Theme.of(context).colorScheme.outline,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
