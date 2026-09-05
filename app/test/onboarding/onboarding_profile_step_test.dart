import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/widgets/onboarding_profile_step.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/craftsky_field_scaffold.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-003 pre-fills editable identity and read-only handle', (
    tester,
  ) async {
    final state = OnboardingFlowState.fromProfile(
      Profile(
        did: 'did:plc:alice',
        handle: 'alice.test',
        displayName: 'Alice',
        description: 'Textile maker',
        crafts: const [],
      ),
    );
    String? changedName;
    String? changedBio;
    var avatarPicks = 0;

    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: SingleChildScrollView(
            child: OnboardingProfileStep(
              state: state,
              onDisplayNameChanged: (value) => changedName = value,
              onBioChanged: (value) => changedBio = value,
              onPickAvatar: () => avatarPicks++,
            ),
          ),
        ),
      ),
    );

    expect(find.text('Signed in as @alice.test'), findsOneWidget);
    expect(find.text('Alice'), findsOneWidget);
    expect(find.text('Textile maker'), findsOneWidget);
    expect(
      find.ancestor(
        of: find.byKey(const Key('onboarding-display-name')),
        matching: find.byType(BrandTextField),
      ),
      findsOneWidget,
    );
    expect(
      find.ancestor(
        of: find.byKey(const Key('onboarding-bio')),
        matching: find.byType(BrandTextField),
      ),
      findsOneWidget,
    );
    expect(find.text('5/64'), findsOneWidget);
    expect(find.text('13/256'), findsOneWidget);
    expect(
      find.descendant(
        of: find.byType(CraftskyFocusLift),
        matching: find.text('5/64'),
      ),
      findsNothing,
    );
    await tester.enterText(
      find.byKey(const Key('onboarding-display-name')),
      'Alicia',
    );
    await tester.pump();
    expect(find.text('6/64'), findsOneWidget);
    await tester.enterText(find.byKey(const Key('onboarding-bio')), 'New bio');
    await tester.tap(find.bySemanticsLabel('Change avatar'));
    expect(changedName, 'Alicia');
    expect(changedBio, 'New bio');
    expect(avatarPicks, 1);
    expect(find.byIcon(CraftskyIcons.camera), findsNothing);
    expect(find.textContaining('Remove'), findsNothing);
  });

  testWidgets('AT-004 shows avatar upload progress and failure feedback', (
    tester,
  ) async {
    final initial = OnboardingFlowState.fromProfile(
      Profile(
        did: 'did:plc:alice',
        handle: 'alice.test',
        crafts: const [],
      ),
    );

    await _pumpStep(tester, initial.copyWith(uploadingAvatar: true));
    expect(find.text('Uploading photo...'), findsOneWidget);
    expect(find.byType(LinearProgressIndicator), findsOneWidget);

    await _pumpStep(tester, initial.copyWith(avatarUploadFailed: true));
    expect(find.text('Photo upload failed. Try again.'), findsOneWidget);
    expect(find.byIcon(CraftskyIcons.error), findsOneWidget);
  });

  testWidgets('applies profile prefill that arrives after the first frame', (
    tester,
  ) async {
    final empty = OnboardingFlowState.fromProfile(
      Profile(
        did: 'did:plc:alice',
        handle: 'alice.test',
        crafts: const [],
      ),
    );
    await _pumpStep(tester, empty);

    await _pumpStep(
      tester,
      OnboardingFlowState.fromProfile(
        Profile(
          did: 'did:plc:alice',
          handle: 'alice.test',
          displayName: 'Alice',
          description: 'Textile maker',
          crafts: const [],
        ),
      ),
    );

    expect(
      tester
          .widget<TextField>(
            find.byKey(const Key('onboarding-display-name')),
          )
          .controller
          ?.text,
      'Alice',
    );
    expect(
      tester
          .widget<TextField>(find.byKey(const Key('onboarding-bio')))
          .controller
          ?.text,
      'Textile maker',
    );
  });
}

Future<void> _pumpStep(
  WidgetTester tester,
  OnboardingFlowState state,
) => tester.pumpWidget(
  MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(
      body: SingleChildScrollView(
        child: OnboardingProfileStep(
          state: state,
          onDisplayNameChanged: (_) {},
          onBioChanged: (_) {},
          onPickAvatar: () {},
        ),
      ),
    ),
  ),
);
