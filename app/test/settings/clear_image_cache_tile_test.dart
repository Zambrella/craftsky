import 'package:craftsky_app/settings/widgets/clear_image_cache_tile.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/image_cache_fakes.dart';

void main() {
  group('ClearImageCacheTile', () {
    testWidgets('tap calls emptyCache on both managers', (tester) async {
      final profileFake = FakeBaseCacheManager();
      final feedFake = FakeBaseCacheManager();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            profileImageCacheManagerProvider.overrideWith((ref) => profileFake),
            feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
          ],
          child: const MaterialApp(
            home: Scaffold(body: ClearImageCacheTile()),
          ),
        ),
      );

      await tester.tap(find.byType(ClearImageCacheTile));
      await tester.pump(); // schedule the future
      await tester.pump(); // resolve

      expect(profileFake.emptyCacheCalls, 1);
      expect(feedFake.emptyCacheCalls, 1);
    });

    testWidgets('shows success snackbar when both caches clear', (
      tester,
    ) async {
      final profileFake = FakeBaseCacheManager();
      final feedFake = FakeBaseCacheManager();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            profileImageCacheManagerProvider.overrideWith((ref) => profileFake),
            feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
          ],
          child: const MaterialApp(
            home: Scaffold(body: ClearImageCacheTile()),
          ),
        ),
      );

      await tester.tap(find.byType(ClearImageCacheTile));
      await tester.pump();
      await tester.pump();

      expect(find.text('Image cache cleared'), findsOneWidget);
    });

    testWidgets('shows error snackbar when a cache fails to clear', (
      tester,
    ) async {
      final profileFake = FakeBaseCacheManager()
        ..throwOnEmptyCache = StateError('disk full');
      final feedFake = FakeBaseCacheManager();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            profileImageCacheManagerProvider.overrideWith((ref) => profileFake),
            feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
          ],
          child: const MaterialApp(
            home: Scaffold(body: ClearImageCacheTile()),
          ),
        ),
      );

      await tester.tap(find.byType(ClearImageCacheTile));
      await tester.pump();
      await tester.pump();

      expect(
        find.textContaining('Could not clear cache'),
        findsOneWidget,
      );
    });
  });
}
