import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/observability/error_reporting_consent_tile.dart';
import 'package:craftsky_app/shared/observability/error_reporting_controller.dart';
import 'package:craftsky_app/shared/observability/error_reporting_provider.dart';
import 'package:craftsky_app/shared/observability/sentry_config.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  testWidgets('onboarding opt-in and settings share saved device consent', (
    tester,
  ) async {
    final prefs = await SharedPreferences.getInstance();
    var closes = 0;
    final controller = ErrorReportingController(
      preferences: prefs,
      config: SentryConfig.fromValues(environment: 'production'),
      adapter: const SentryFlutterBootstrapAdapter(),
      closeSdk: () async {
        closes++;
      },
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          errorReportingControllerProvider.overrideWith((ref) => controller),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const Scaffold(
            body: Column(
              children: [
                ErrorReportingConsentTile(onboarding: true),
                ErrorReportingConsentTile(),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final checkbox = find.byKey(const Key('onboarding-error-reporting'));
    final toggle = find.byKey(const Key('settings-error-reporting'));
    expect(tester.widget<CheckboxListTile>(checkbox).value, isFalse);
    expect(tester.widget<SwitchListTile>(toggle).value, isFalse);
    await tester.tap(checkbox);
    await tester.pumpAndSettle();
    expect(tester.widget<CheckboxListTile>(checkbox).value, isTrue);
    expect(tester.widget<SwitchListTile>(toggle).value, isTrue);
    expect(prefs.getBool(ErrorReportingController.preferenceKey), isTrue);
    await tester.tap(toggle);
    await tester.pumpAndSettle();
    expect(tester.widget<CheckboxListTile>(checkbox).value, isFalse);
    expect(tester.widget<SwitchListTile>(toggle).value, isFalse);
    expect(prefs.getBool(ErrorReportingController.preferenceKey), isFalse);
    expect(closes, 1);
  });
}
