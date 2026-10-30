import 'package:craftsky_app/theme/animated_chunky_button.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets('one button morphs width and colour with its right edge fixed', (
    tester,
  ) async {
    final compact = ValueNotifier(false);
    addTearDown(compact.dispose);
    var posts = 0;
    var dismissals = 0;
    await _pumpButton(
      tester,
      compact,
      onPost: () => posts++,
      onDone: () => dismissals++,
    );
    final full = tester.getRect(find.byType(ChunkyButton));
    final primary = _surface(tester);
    final originalState = tester.state(find.byType(ChunkyButton));
    final semantics = tester.ensureSemantics();

    compact.value = true;
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    final middle = tester.getRect(find.byType(ChunkyButton));
    final blended = _surface(tester);
    expect(middle.right, full.right);
    expect(middle.width, lessThan(full.width));
    expect(blended, isNot(primary));
    expect(find.byType(ChunkyButton), findsOneWidget);
    expect(tester.state(find.byType(ChunkyButton)), same(originalState));
    expect(find.text('Post'), findsOneWidget);
    final labelFades = tester.widgetList<FadeTransition>(
      find.ancestor(
        of: find.text('Done'),
        matching: find.byType(FadeTransition),
      ),
    );
    expect(
      labelFades.map((fade) => fade.opacity.value),
      contains(inExclusiveRange(0, 1)),
    );
    expect(tester.getSemantics(find.byType(ChunkyButton)).label, 'Done');
    await tester.tap(find.byType(ChunkyButton));
    expect(posts, 0);
    expect(dismissals, 1);
    await tester.pumpAndSettle();
    final small = tester.getRect(find.byType(ChunkyButton));
    expect(small.width, lessThan(middle.width));
    expect(small.right, full.right);
    expect(_surface(tester), isNot(blended));
    expect(find.text('Post'), findsNothing);

    compact.value = false;
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    final expanding = tester.getRect(find.byType(ChunkyButton));
    expect(expanding.width, greaterThan(small.width));
    expect(expanding.width, lessThan(full.width));
    // Reversing a partially completed transition starts from its current size.
    compact.value = true;
    await tester.pump();
    expect(tester.getRect(find.byType(ChunkyButton)), expanding);
    await tester.pumpAndSettle();
    expect(tester.getRect(find.byType(ChunkyButton)), small);
    semantics.dispose();
  });

  testWidgets('reduced motion snaps and compact width follows text scaling', (
    tester,
  ) async {
    final compact = ValueNotifier(false);
    addTearDown(compact.dispose);
    await _pumpButton(tester, compact, reduced: true, scale: 2);
    compact.value = true;
    await tester.pump();
    final button = tester.getRect(find.byType(ChunkyButton));
    await tester.pump(const Duration(milliseconds: 80));
    expect(tester.getRect(find.byType(ChunkyButton)), button);
    expect(find.text('Post'), findsNothing);
    expect(find.text('Done'), findsOneWidget);
    final text = tester.getRect(find.text('Done'));
    expect(text.width, lessThan(button.width));
    expect(tester.takeException(), isNull);
  });
}

Color? _surface(WidgetTester tester) => tester
    .widgetList<DecoratedBox>(
      find.descendant(
        of: find.byType(ChunkyButton),
        matching: find.byType(DecoratedBox),
      ),
    )
    .map((widget) => widget.decoration)
    .whereType<ShapeDecoration>()
    .singleWhere(
      (decoration) =>
          decoration.shape is StadiumBorder &&
          (decoration.shape as StadiumBorder).side.width > 0,
    )
    .color;

Future<void> _pumpButton(
  WidgetTester tester,
  ValueNotifier<bool> compact, {
  VoidCallback? onPost,
  VoidCallback? onDone,
  bool reduced = false,
  double scale = 1,
}) async {
  await tester.pumpWidget(
    MaterialApp(
      theme: AppTheme.lightThemeData,
      home: MediaQuery(
        data: MediaQueryData(
          disableAnimations: reduced,
          textScaler: TextScaler.linear(scale),
        ),
        child: Scaffold(
          body: Center(
            child: SizedBox(
              width: 360,
              child: ValueListenableBuilder(
                valueListenable: compact,
                builder: (context, compact, _) => AnimatedChunkyButton(
                  compact: compact,
                  label: 'Post',
                  compactLabel: 'Done',
                  actionKey: ValueKey(compact),
                  onPressed: compact ? (onDone ?? () {}) : (onPost ?? () {}),
                ),
              ),
            ),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}
