import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_flow_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

final _lease = ActiveAccountLease(
  session: AccountSessionLease(
    account: AccountKey('did:plc:alice'),
    sessionGeneration: 1,
  ),
  activationGeneration: 1,
);

final class _Flow extends OnboardingFlow {
  _Flow(this.initial);
  final OnboardingFlowState initial;
  int completionCalls = 0;

  @override
  Future<OnboardingFlowState> build(ActiveAccountLease lease) async => initial;

  @override
  Future<void> complete() async => completionCalls++;
}

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));
  testWidgets('Finish stays disabled until age and policy agreement', (
    tester,
  ) async {
    final flow = _Flow(
      OnboardingFlowState.fromProfile(
        Profile(did: 'did:plc:alice', handle: 'alice.test', crafts: const []),
      ).copyWith(step: OnboardingStep.guidelines),
    );
    await _pump(tester, flow);
    ChunkyButton button() => tester.widget(find.byType(ChunkyButton));
    expect(button().onPressed, isNull);
    await tester.tap(find.text('Finish'));
    expect(flow.completionCalls, 0);
    final declaration = find.byKey(
      const Key('onboarding-minimum-age-declaration'),
    );
    await tester.ensureVisible(declaration);
    await tester.tap(declaration);
    await tester.pumpAndSettle();
    expect(button().onPressed, isNotNull);
    final optional = tester.widget<CheckboxListTile>(
      find.byKey(const Key('onboarding-error-reporting')),
    );
    expect(optional.value, isFalse);
    await tester.tap(declaration);
    await tester.pumpAndSettle();
    expect(button().onPressed, isNull);
  });

  testWidgets('Finish completes onboarding from the guidelines step', (
    tester,
  ) async {
    final flow = _Flow(
      OnboardingFlowState.fromProfile(
        Profile(did: 'did:plc:alice', handle: 'alice.test', crafts: const []),
      ).copyWith(step: OnboardingStep.guidelines, meetsMinimumAge: true),
    );
    await _pump(tester, flow);

    expect(find.text('Finish'), findsOneWidget);
    await tester.tap(find.text('Finish'));
    await tester.pump();
    expect(flow.completionCalls, 1);
  });
}

Future<void> _pump(WidgetTester tester, _Flow flow) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        activeAccountInitializationProvider.overrideWith(
          (ref) => ActiveAccountInitialization(
            lease: _lease,
            languagePreferences: const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
            onboardingComplete: false,
          ),
        ),
        onboardingFlowProvider.overrideWith2((_) => flow),
      ],
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const OnboardingPage(),
      ),
    ),
  );
  await tester.pumpAndSettle();
}
