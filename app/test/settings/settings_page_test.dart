import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:craftsky_app/settings/widgets/clear_image_cache_tile.dart';
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets(
    'SettingsPage renders title, ClearImageCacheTile, and SignOutTile',
    (tester) async {
      await tester.pumpWidget(
        const ProviderScope(child: MaterialApp(home: SettingsPage())),
      );
      expect(find.text('Settings'), findsWidgets);
      expect(find.byType(ClearImageCacheTile), findsOneWidget);
      expect(find.byType(SignOutTile), findsOneWidget);
    },
  );
}
