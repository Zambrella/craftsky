import 'package:craftsky_app/profile/widgets/profile_entrance.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

Widget _scene({bool reducedMotion = false, String label = 'Profile'}) =>
    MaterialApp(
      home: MediaQuery(
        data: MediaQueryData(disableAnimations: reducedMotion),
        child: Scaffold(
          body: ProfileEntrance(
            child: Column(
              children: [
                for (final stage in ProfileEntranceStage.values)
                  ProfileEntranceFade(
                    stage: stage,
                    child: Text('$label ${stage.name}'),
                  ),
              ],
            ),
          ),
        ),
      ),
    );

double _opacity(WidgetTester tester, ProfileEntranceStage stage) => tester
    .widget<FadeTransition>(
      find.descendant(
        of: find.byWidgetPredicate(
          (widget) => widget is ProfileEntranceFade && widget.stage == stage,
        ),
        matching: find.byType(FadeTransition),
      ),
    )
    .opacity
    .value;

void main() {
  testWidgets('details fade from top to bottom within 800 ms without moving', (
    tester,
  ) async {
    await tester.pumpWidget(_scene());
    final original = {
      for (final stage in ProfileEntranceStage.values)
        stage: tester.getRect(find.text('Profile ${stage.name}')),
    };
    await tester.pump(const Duration(milliseconds: 160));
    expect(
      _opacity(tester, ProfileEntranceStage.summary),
      greaterThan(_opacity(tester, ProfileEntranceStage.actions)),
    );
    expect(_opacity(tester, ProfileEntranceStage.actions), greaterThan(0));
    expect(_opacity(tester, ProfileEntranceStage.bio), 0);
    expect(_opacity(tester, ProfileEntranceStage.content), 0);
    await tester.pump(const Duration(milliseconds: 320));
    expect(
      _opacity(tester, ProfileEntranceStage.bio),
      greaterThan(_opacity(tester, ProfileEntranceStage.tabs)),
    );
    expect(
      _opacity(tester, ProfileEntranceStage.tabs),
      greaterThan(_opacity(tester, ProfileEntranceStage.content)),
    );
    await tester.pump(const Duration(milliseconds: 320));
    for (final stage in ProfileEntranceStage.values) {
      expect(_opacity(tester, stage), 1);
      expect(
        tester.getRect(find.text('Profile ${stage.name}')),
        original[stage],
      );
    }
    expect(tester.takeException(), isNull);
  });

  testWidgets('refreshing profile details does not replay the entrance', (
    tester,
  ) async {
    await tester.pumpWidget(_scene());
    await tester.pump(ProfileEntrance.duration);
    await tester.pumpWidget(_scene(label: 'Updated'));
    for (final stage in ProfileEntranceStage.values) {
      expect(_opacity(tester, stage), 1);
    }
    expect(find.text('Updated summary'), findsOneWidget);
  });

  testWidgets(
    'reduced motion shows all details immediately and finishes an active fade',
    (tester) async {
      await tester.pumpWidget(_scene(reducedMotion: true));
      expect(
        find.descendant(
          of: find.byType(ProfileEntrance),
          matching: find.byType(FadeTransition),
        ),
        findsNothing,
      );
      await tester.pumpWidget(_scene());
      for (final stage in ProfileEntranceStage.values) {
        expect(_opacity(tester, stage), 1);
      }
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpWidget(_scene());
      await tester.pump(const Duration(milliseconds: 30));
      expect(_opacity(tester, ProfileEntranceStage.content), 0);
      await tester.pumpWidget(_scene(reducedMotion: true));
      await tester.pumpWidget(_scene());
      for (final stage in ProfileEntranceStage.values) {
        expect(_opacity(tester, stage), 1);
      }
      expect(tester.takeException(), isNull);
    },
  );
}
