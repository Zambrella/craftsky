import 'package:craftsky_app/profile/pages/saved_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('SavedPage renders its title', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: SavedPage()),
      ),
    );
    expect(find.text('Saved'), findsWidgets);
  });
}
