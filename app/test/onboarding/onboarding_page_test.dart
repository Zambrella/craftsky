import 'dart:async';

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
import 'package:craftsky_app/theme/craftsky_icons.dart';
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
  _Flow(this.initial);
  final OnboardingFlowState initial;
  int completionCalls = 0;

  @override
  Future<OnboardingFlowState> build(ActiveAccountLease lease) async => initial;

  @override
  Future<void> complete() async => completionCalls++;
}

final class _PendingFlow extends OnboardingFlow {
  int completionCalls = 0;

  @override
  Future<OnboardingFlowState> build(ActiveAccountLease lease) =>
      Completer<OnboardingFlowState>().future;

  @override
  Future<void> complete() async => completionCalls++;
}

final class _ErrorFlow extends OnboardingFlow {
  int completionCalls = 0;

  @override
  Future<OnboardingFlowState> build(ActiveAccountLease lease) async =>
      throw StateError('offline');

  @override
  Future<void> complete() async => completionCalls++;
}

void main() {
  testWidgets(
    'AT-007 Skip remains available during prefill loading and error',
    (
      tester,
    ) async {
      Future<void> pump(OnboardingFlow flow) => tester.pumpWidget(
        ProviderScope(
          retry: (_, _) => null,
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

      final pending = _PendingFlow();
      await pump(pending);
      await tester.pump();
      expect(find.text('Skip'), findsOneWidget);
      await tester.tap(find.text('Skip'));
      await tester.pump();
      expect(pending.completionCalls, 1);

      await tester.pumpWidget(const SizedBox.shrink());
      final failed = _ErrorFlow();
      await pump(failed);
      await tester.pump();
      await tester.pump();
      expect(find.text('Skip'), findsOneWidget);
      await tester.tap(find.text('Skip'));
      await tester.pump();
      expect(failed.completionCalls, 1);
    },
  );

  testWidgets('shows localized sequential progress and deterministic Back', (
    tester,
  ) async {
    final profile = Profile(
      did: 'did:plc:alice',
      handle: 'alice.test',
      crafts: const [],
    );
    final flow = _Flow(OnboardingFlowState.fromProfile(profile));
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

    expect(find.text('Step 1 of 3'), findsOneWidget);
    expect(find.text('Signed in as @alice.test'), findsOneWidget);
    expect(find.byType(LinearProgressIndicator), findsOneWidget);

    await tester.tap(find.text('Next'));
    await tester.pump();
    expect(find.text('Step 2 of 3'), findsOneWidget);
    expect(find.text('What do you make?'), findsOneWidget);

    await tester.tap(find.byIcon(CraftskyIconsBold.back));
    await tester.pump();
    expect(find.text('Step 1 of 3'), findsOneWidget);
  });

  testWidgets('Skip completes immediately without writing a draft', (
    tester,
  ) async {
    final flow = _Flow(
      OnboardingFlowState.fromProfile(
        Profile(
          did: 'did:plc:alice',
          handle: 'alice.test',
          crafts: const [],
        ),
      ),
    );
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
    await tester.enterText(
      find.byKey(const Key('onboarding-display-name')),
      'Unsaved',
    );
    await tester.tap(find.text('Skip'));
    await tester.pump();
    expect(flow.completionCalls, 1);
  });

  testWidgets('complete flow content is centered and capped at tablet width', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(1400, 900);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.view.resetPhysicalSize);
    final flow = _Flow(
      OnboardingFlowState.fromProfile(
        Profile(
          did: 'did:plc:alice',
          handle: 'alice.test',
          crafts: const [],
        ),
      ),
    );

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

    final bounds = tester.getRect(
      find.byKey(const Key('onboarding-flow-content')),
    );
    expect(bounds.width, 900);
    expect(bounds.center.dx, 700);
    expect(tester.takeException(), isNull);
  });

  testWidgets('Request more opens support without changing the craft draft', (
    tester,
  ) async {
    final initial = OnboardingFlowState.fromProfile(
      Profile(
        did: 'did:plc:alice',
        handle: 'alice.test',
        crafts: const ['sewing', 'weaving'],
      ),
    ).copyWith(step: OnboardingStep.crafts);
    final flow = _Flow(initial);
    Uri? confirmedUri;
    Uri? launchedUri;
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
          home: OnboardingPage(
            confirmOpenLink: (context, uri) async {
              confirmedUri = uri;
              return true;
            },
            linkLauncher: (uri) async {
              launchedUri = uri;
              return true;
            },
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Request more'));
    await tester.pump();

    const expected =
        'https://userinput.app/s/did:plc:lmmx63zcns6gewgxqfdt4kof/'
        '3mpr5izppvt2k?lang=en';
    expect(confirmedUri.toString(), expected);
    expect(launchedUri.toString(), expected);
    expect(flow.state.requireValue.selectedCraftIds, initial.selectedCraftIds);
    expect(find.text('What do you make?'), findsOneWidget);
  });
}
