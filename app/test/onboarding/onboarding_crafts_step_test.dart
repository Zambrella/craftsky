import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/widgets/onboarding_crafts_step.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'selection, accessible reorder and default follow profile order',
    (tester) async {
      var state = OnboardingFlowState.fromProfile(
        Profile(
          did: 'did:plc:alice',
          handle: 'alice.test',
          crafts: const ['quilting', 'sewing', 'future-craft'],
        ),
      ).copyWith(step: OnboardingStep.crafts);
      var requestMoreCalls = 0;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: SingleChildScrollView(
              child: StatefulBuilder(
                builder: (context, setState) {
                  return OnboardingCraftsStep(
                    state: state,
                    onToggle: (craft) => setState(() {
                      final next = state.selectedCraftIds.toSet();
                      next.contains(craft.id)
                          ? next.remove(craft.id)
                          : next.add(craft.id);
                      state = state.copyWith(selectedCraftIds: next);
                    }),
                    onReorder: (crafts) => setState(
                      () => state = state.copyWith(
                        selectedCraftIds: crafts
                            .map((craft) => craft.id)
                            .toSet(),
                      ),
                    ),
                    onRequestMore: () => requestMoreCalls++,
                  );
                },
              ),
            ),
          ),
        ),
      );
      Finder row(String id) => find.byKey(ValueKey('craft-choice-$id'));
      void expectDefault(String id) {
        expect(
          find.descendant(of: row(id), matching: find.text('Default')),
          findsOneWidget,
        );
        expect(find.byIcon(Icons.star), findsOneWidget);
      }

      expectDefault('quilting');
      expect(
        tester.getTopLeft(row('quilting')).dy,
        lessThan(tester.getTopLeft(row('sewing')).dy),
      );
      expect(find.byType(Checkbox), findsNWidgets(5));
      expect(find.text('future-craft'), findsNothing);
      expect(find.text('Weaving'), findsNothing);
      await tester.tap(find.byTooltip('Reorder Sewing'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Move up'));
      await tester.pumpAndSettle();
      expect(state.selectedCraftIds.toList(), ['sewing', 'quilting']);
      expectDefault('sewing');
      await tester.tap(row('knitting'));
      await tester.pumpAndSettle();
      expect(state.selectedCraftIds.toList(), [
        'sewing',
        'quilting',
        'knitting',
      ]);
      await tester.tap(row('sewing'));
      await tester.pumpAndSettle();
      expectDefault('quilting');
      await tester.tap(find.text('Request more'));
      expect(requestMoreCalls, 1);
    },
  );

  testWidgets(
    'picker fits a narrow screen with large text and locks while saving',
    (tester) async {
      tester.view.physicalSize = const Size(320, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      var changed = false;
      final state = OnboardingFlowState.fromProfile(
        Profile(
          did: 'did:plc:alice',
          handle: 'alice.test',
          crafts: const ['embroidery', 'knitting', 'quilting'],
        ),
      ).copyWith(saving: true);
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(context)
                .copyWith(textScaler: const TextScaler.linear(2)),
            child: child!,
          ),
          home: Scaffold(
            body: SingleChildScrollView(
              child: OnboardingCraftsStep(
                state: state,
                onToggle: (_) => changed = true,
                onReorder: (_) => changed = true,
                onRequestMore: () {},
              ),
            ),
          ),
        ),
      );
      expect(tester.takeException(), isNull);
      expect(
        tester.widget<Checkbox>(find.byType(Checkbox).first).onChanged,
        isNull,
      );
      await tester.tap(find.text('Embroidery'));
      await tester.pump();
      expect(changed, isFalse);
    },
  );

  for (final (hold, kind) in [
    (true, PointerDeviceKind.touch),
    (false, PointerDeviceKind.mouse),
    (false, PointerDeviceKind.touch),
  ]) {
    testWidgets('handle supports hold=$hold with $kind without a tooltip', (
      tester,
    ) async {
      var state = OnboardingFlowState.fromProfile(
        Profile(
          did: 'did:plc:alice',
          handle: 'alice.test',
          crafts: const ['sewing', 'knitting'],
        ),
      );
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: StatefulBuilder(
              builder: (context, setState) => OnboardingCraftsStep(
                state: state,
                onToggle: (_) {},
                onReorder: (crafts) => setState(
                  () => state = state.copyWith(
                    selectedCraftIds: crafts.map((craft) => craft.id).toSet(),
                  ),
                ),
                onRequestMore: () {},
              ),
            ),
          ),
        ),
      );
      final gesture = await tester.startGesture(
        tester.getCenter(
          find.descendant(
            of: find.byKey(const ValueKey('craft-choice-knitting')),
            matching: find.byWidgetPredicate(
              (widget) => widget is ReorderableDragStartListener,
            ),
          ),
        ),
        kind: kind,
      );
      if (hold) {
        await tester.pump(const Duration(milliseconds: 600));
        expect(find.byType(CraftskyCard), findsOneWidget);
        expect(find.text('Drag to reorder Knitting'), findsNothing);
      }
      await gesture.moveBy(const Offset(0, -25));
      await tester.pump();
      expect(find.byType(CraftskyCard), findsOneWidget);
      await gesture.moveBy(const Offset(0, -40));
      await tester.pump(const Duration(milliseconds: 300));
      await gesture.up();
      await tester.pumpAndSettle();
      expect(state.selectedCraftIds.toList(), ['knitting', 'sewing']);
    });
  }
}
