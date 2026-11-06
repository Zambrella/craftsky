import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/settings/logged_out_visibility_controller.dart';
import 'package:craftsky_app/settings/logged_out_visibility_provider.dart';
import 'package:craftsky_app/settings/pages/privacy_page.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
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
        overrides: [
          sharedPreferencesProvider.overrideWithValue(preferences),
          loggedOutVisibilityControllerProvider.overrideWith(
            (ref) => LoggedOutVisibilityController(
              read: () async => const VisibilitySetting(
                hideFromLoggedOut: false,
                expectedCid: 'known',
              ),
              save: (intent) async => VisibilitySetting(
                hideFromLoggedOut: intent.hideFromLoggedOut,
                expectedCid: 'saved',
              ),
            ),
          ),
        ],
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
  testWidgets('AT-008 unknown shared preference has no false off switch', (
    tester,
  ) async {
    var attempts = 0;
    final model = LoggedOutVisibilityController(
      read: () async {
        if (++attempts == 1) throw Exception('unavailable');
        return const VisibilitySetting(
          hideFromLoggedOut: true,
          expectedCid: 'current',
        );
      },
      save: (_) async => throw StateError('unexpected write'),
    );
    addTearDown(model.dispose);
    await _pumpVisibility(tester, model);
    expect(find.widgetWithText(SwitchListTile, _visibilityLabel), findsNothing);
    await tester.tap(find.text('Reload preference'));
    await tester.pumpAndSettle();
    expect(
      tester
          .widget<SwitchListTile>(
            find.widgetWithText(SwitchListTile, _visibilityLabel),
          )
          .value,
      isTrue,
    );
  });
  testWidgets(
    'AT-008 pending unhide keeps the confirmed switch and reporting consent',
    (tester) async {
      final intents = <VisibilityIntent>[];
      final model = LoggedOutVisibilityController(
        read: () async => const VisibilitySetting(
          hideFromLoggedOut: true,
          expectedCid: 'current',
        ),
        save: (intent) async {
          intents.add(intent);
          if (intents.length == 1) {
            throw const PdsMutationAmbiguousException(retryAfterSeconds: 1);
          }
          return const VisibilitySetting(
            hideFromLoggedOut: false,
            expectedCid: 'saved',
          );
        },
      );
      addTearDown(model.dispose);
      final preferences = await _pumpVisibility(tester, model);
      final reportingBefore = preferences.getBool(
        ErrorReportingController.preferenceKey,
      );
      final tile = find.widgetWithText(SwitchListTile, _visibilityLabel);
      await tester.tap(tile);
      await tester.pumpAndSettle();
      expect(tester.widget<SwitchListTile>(tile).value, isTrue);
      expect(tester.widget<SwitchListTile>(tile).onChanged, isNull);
      await tester.scrollUntilVisible(
        find.text('Retry save'),
        300,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.tap(find.text('Retry save'));
      await tester.pumpAndSettle();
      expect(intents, hasLength(2));
      expect(identical(intents[0], intents[1]), isTrue);
      await tester.scrollUntilVisible(
        tile,
        -300,
        scrollable: find.byType(Scrollable).first,
      );
      expect(tester.widget<SwitchListTile>(tile).value, isFalse);
      expect(
        preferences.getBool(ErrorReportingController.preferenceKey),
        reportingBefore,
      );
    },
  );
}

const _visibilityLabel =
    'Ask apps and sites not to show my account to logged-out users';
Future<SharedPreferences> _pumpVisibility(
  WidgetTester tester,
  LoggedOutVisibilityController model,
) async {
  SharedPreferences.setMockInitialValues({});
  final preferences = await SharedPreferences.getInstance();
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        sharedPreferencesProvider.overrideWithValue(preferences),
        loggedOutVisibilityControllerProvider.overrideWithValue(model),
      ],
      child: const MaterialApp(
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: PrivacyPage(),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return preferences;
}
