import 'package:craftsky_app/shared/widgets/scroll_to_top_button.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('only exposes the back-to-top action while visible', (
    tester,
  ) async {
    var pressed = false;
    Widget harness({required bool visible}) => MaterialApp(
      theme: AppTheme.lightThemeData,
      home: Scaffold(
        body: ScrollToTopButton(
          visible: visible,
          tooltip: 'Back to top',
          onPressed: () => pressed = true,
        ),
      ),
    );

    await tester.pumpWidget(harness(visible: false));
    expect(find.byKey(const Key('scroll-to-top-button')), findsNothing);

    await tester.pumpWidget(harness(visible: true));
    await tester.pumpAndSettle();
    expect(find.byTooltip('Back to top'), findsOneWidget);
    final context = tester.element(
      find.byKey(const Key('scroll-to-top-button')),
    );
    final button = tester.widget<FloatingActionButton>(
      find.byKey(const Key('scroll-to-top-button')),
    );
    expect(
      button.backgroundColor,
      Theme.of(context).colorScheme.surfaceContainerHigh,
    );
    expect(
      button.foregroundColor,
      Theme.of(context).colorScheme.onSurfaceVariant,
    );

    await tester.tap(find.byKey(const Key('scroll-to-top-button')));
    expect(pressed, isTrue);
  });
}
