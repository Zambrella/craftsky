import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:material_ui/material_ui.dart';

/// Displayed before the owner explicitly enables CraftSky messaging.
class MessagingExplanation extends StatelessWidget {
  const new({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(l10n.chatSharedInboxDisclosure),
        const SizedBox(height: 16),
        Text(l10n.chatStorageDisclosure),
        const SizedBox(height: 16),
        Text(l10n.chatReportDisclosure),
        TextButton(
          onPressed: () => const SafetyRoute().push<void>(context),
          child: Text(l10n.safetyTitle),
        ),
      ],
    );
  }
}
