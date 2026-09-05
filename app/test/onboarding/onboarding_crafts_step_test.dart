import 'dart:ui' show Tristate;

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/widgets/onboarding_crafts_step.dart';
import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-005 shows the stable catalog and selected craft semantics', (
    tester,
  ) async {
    Craft? toggled;
    var requestMoreCalls = 0;
    final semantics = tester.ensureSemantics();
    final state = OnboardingFlowState.fromProfile(
      Profile(
        did: 'did:plc:alice',
        handle: 'alice.test',
        crafts: const ['sewing', 'future-craft'],
      ),
    ).copyWith(step: OnboardingStep.crafts);

    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: SingleChildScrollView(
            child: OnboardingCraftsStep(
              state: state,
              onToggle: (craft) => toggled = craft,
              onRequestMore: () => requestMoreCalls++,
            ),
          ),
        ),
      ),
    );

    expect(find.text('Sewing'), findsOneWidget);
    final sewingSemantics = tester.getSemantics(find.text('Sewing'));
    expect(sewingSemantics.label, 'Sewing');
    expect(
      sewingSemantics.flagsCollection.isButton,
      isTrue,
    );
    expect(
      sewingSemantics.flagsCollection.isSelected,
      Tristate.isTrue,
    );
    expect(
      canonicalSelectableCrafts.map((craft) => craft.id),
      ['sewing', 'knitting', 'crochet', 'quilting', 'embroidery'],
    );
    expect(find.byType(CraftIcon), findsNWidgets(5));
    expect(find.text('Weaving'), findsNothing);
    await tester.tap(find.text('Sewing'));
    expect(toggled, Craft.sewing);
    await tester.tap(find.text('Request more'));
    expect(requestMoreCalls, 1);
    expect(toggled, Craft.sewing);
    semantics.dispose();
  });
}
