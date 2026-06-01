import 'package:craftsky_app/router/router.dart';
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

  testWidgets('AT-006 SearchPage can hold hashtag context', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: SearchPage(tag: 'SockKAL')),
      ),
    );
    expect(find.text('Search'), findsWidgets);
    expect(find.text('#SockKAL'), findsOneWidget);
  });

  test('AT-006 SearchRoute preserves tag query context', () {
    expect(const SearchRoute(tag: 'SockKAL').location, '/search?tag=SockKAL');
  });
}
