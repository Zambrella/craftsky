import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('FeedPage renders its title', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: FeedPage()),
      ),
    );
    expect(find.text('Feed'), findsWidgets);
  });
}
