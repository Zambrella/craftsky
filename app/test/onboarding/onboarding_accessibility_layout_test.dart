import 'dart:ui' show Tristate;

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/onboarding/models/onboarding_action_state.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_flow_provider.dart';
import 'package:craftsky_app/onboarding/widgets/onboarding_bottom_action.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final _lease = ActiveAccountLease(
  session: AccountSessionLease(
    account: AccountKey('did:plc:alice'),
    sessionGeneration: 1,
  ),
  activationGeneration: 1,
);

final class _Flow extends OnboardingFlow {
  @override
  Future<OnboardingFlowState> build(ActiveAccountLease lease) async =>
      OnboardingFlowState.fromProfile(
        Profile(
          did: 'did:plc:alice',
          handle: 'alice.test',
          displayName: 'Alice',
          description: 'A long profile description for layout verification.',
          crafts: const [],
        ),
      );
}

void main() {
  testWidgets('AT-014 busy primary action preserves purpose and state', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: OnboardingBottomAction(
            state: const OnboardingActionState(
              kind: OnboardingActionKind.saveAndNext,
              canSubmit: false,
              canSkip: false,
              canGoBack: false,
              busy: true,
            ),
            onPressed: () {},
          ),
        ),
      ),
    );

    final action = tester.getSemantics(
      find.byKey(const Key('onboarding-primary-action-semantics')),
    );
    expect(action.label, 'Save & next');
    expect(action.value, 'Loading');
    expect(action.flagsCollection.isEnabled, Tristate.isFalse);
    semantics.dispose();
  });

  testWidgets('AT-014 compact large-text layout keeps actions operable', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(320, 480);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.view.resetPhysicalSize);
    final semantics = tester.ensureSemantics();

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
          onboardingFlowProvider.overrideWith2((_) => _Flow()),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(
              context,
            ).copyWith(textScaler: const TextScaler.linear(2)),
            child: child!,
          ),
          home: const OnboardingPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    final progress = tester.getSemantics(find.text('Step 1 of 3'));
    expect(progress.label, 'Onboarding step 1 of 3');
    expect(progress.value, 'Step 1 of 3');
    expect(find.text('Skip'), findsOneWidget);
    expect(find.text('Next'), findsOneWidget);
    await tester.tap(find.text('Next'));
    await tester.pump();
    expect(find.text('Step 2 of 3'), findsOneWidget);
    semantics.dispose();
  });
}
