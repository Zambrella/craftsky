import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/settings/pages/privacy_page.dart';
import 'package:craftsky_app/shared/observability/error_reporting_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  testWidgets('Privacy keeps the persistent error reporting opt-in', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({});
    final preferences = await SharedPreferences.getInstance();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [sharedPreferencesProvider.overrideWithValue(preferences)],
        child: const MaterialApp(
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: PrivacyPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Privacy'), findsOneWidget);
    final reporting = find.byKey(const Key('settings-error-reporting'));
    expect(tester.widget<SwitchListTile>(reporting).value, isFalse);
    await tester.tap(reporting);
    await tester.pumpAndSettle();
    expect(
      preferences.getBool(ErrorReportingController.preferenceKey),
      isTrue,
    );
    expect(tester.widget<SwitchListTile>(reporting).value, isTrue);
    await tester.tap(reporting);
    await tester.pumpAndSettle();
    expect(
      preferences.getBool(ErrorReportingController.preferenceKey),
      isFalse,
    );
    expect(tester.widget<SwitchListTile>(reporting).value, isFalse);
  });
}
