import 'package:craftsky_app/search/pages/search_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('SearchPage renders its title', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: SearchPage()),
      ),
    );
    expect(find.text('Search'), findsWidgets);
  });
}
