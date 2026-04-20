import 'package:craftsky_app/profile/pages/user_profile_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('UserProfilePage renders its title', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          home: UserProfilePage(handle: 'alice.bsky.social'),
        ),
      ),
    );
    expect(find.textContaining('alice'), findsWidgets);
  });
}
