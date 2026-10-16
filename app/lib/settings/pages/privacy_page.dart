import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/observability/error_reporting_consent_tile.dart';
import 'package:material_ui/material_ui.dart';

class PrivacyPage extends StatelessWidget {
  const new({super.key});

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: Text(AppLocalizations.of(context).privacyTitle)),
    body: ListView(children: const [ErrorReportingConsentTile()]),
  );
}
