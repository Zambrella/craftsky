import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('renders its icon, title, and subtitle', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: CraftskyEmptyState(
            icon: Icons.inbox_outlined,
            title: 'Nothing here',
            subtitle: 'New items will appear here.',
          ),
        ),
      ),
    );

    expect(find.byIcon(Icons.inbox_outlined), findsOneWidget);
    expect(find.text('Nothing here'), findsOneWidget);
    expect(find.text('New items will appear here.'), findsOneWidget);
    expect(find.byType(TextButton), findsNothing);
  });

  testWidgets('runs the optional text action', (tester) async {
    var pressed = false;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: CraftskyEmptyState(
            icon: Icons.folder_outlined,
            title: 'No folders',
            subtitle: 'Create one to organise your posts.',
            actionLabel: 'Create folder',
            onAction: () => pressed = true,
          ),
        ),
      ),
    );

    await tester.tap(find.widgetWithText(TextButton, 'Create folder'));

    expect(pressed, isTrue);
  });
}
