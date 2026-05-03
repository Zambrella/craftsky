import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Widget pumpHarness(Widget child) {
    return MaterialApp(
      theme: AppTheme.lightThemeData,
      home: Scaffold(body: Center(child: child)),
    );
  }

  group('CraftskyDialog', () {
    testWidgets('renders title, body, and actions', (tester) async {
      await tester.pumpWidget(
        pumpHarness(
          const CraftskyDialog(
            title: 'A title',
            body: Text('A body'),
            actions: [Text('Action one'), Text('Action two')],
          ),
        ),
      );

      expect(find.text('A title'), findsOneWidget);
      expect(find.text('A body'), findsOneWidget);
      expect(find.text('Action one'), findsOneWidget);
      expect(find.text('Action two'), findsOneWidget);
    });
  });
}
