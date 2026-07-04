import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/widgets/clear_image_cache_tile.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/image_cache_fakes.dart';
import '../fakes/recording_messenger.dart';

typedef _PumpResult = ({
  FakeBaseCacheManager profile,
  FakeBaseCacheManager feed,
  RecordingMessenger messenger,
});

Future<_PumpResult> _pump(
  WidgetTester tester, {
  Object? throwOnEmptyCache,
}) async {
  final profileFake = FakeBaseCacheManager();
  final feedFake = FakeBaseCacheManager();
  if (throwOnEmptyCache != null) {
    profileFake.throwOnEmptyCache = throwOnEmptyCache;
  }
  final messenger = RecordingMessenger();

  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        profileImageCacheManagerProvider.overrideWith((ref) => profileFake),
        feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
      ],
      child: MessengerScope(
        messenger: messenger,
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(body: ClearImageCacheTile()),
        ),
      ),
    ),
  );

  return (profile: profileFake, feed: feedFake, messenger: messenger);
}

void main() {
  group('ClearImageCacheTile', () {
    testWidgets('tap calls emptyCache on both managers', (tester) async {
      final h = await _pump(tester);

      await tester.tap(find.byType(ClearImageCacheTile));
      await tester.pump();
      await tester.pump();

      expect(h.profile.emptyCacheCalls, 1);
      expect(h.feed.emptyCacheCalls, 1);
    });

    testWidgets('shows info message when both caches clear', (tester) async {
      final h = await _pump(tester);

      await tester.tap(find.byType(ClearImageCacheTile));
      await tester.pump();
      await tester.pump();

      expect(h.messenger.calls.length, 1);
      expect(h.messenger.calls.first.$1, 'info');
      expect(h.messenger.calls.first.$2, 'Image cache cleared');
    });

    testWidgets('shows error message when a cache fails to clear', (
      tester,
    ) async {
      final h = await _pump(tester, throwOnEmptyCache: StateError('disk full'));

      await tester.tap(find.byType(ClearImageCacheTile));
      await tester.pump();
      await tester.pump();

      expect(h.messenger.calls.length, 1);
      expect(h.messenger.calls.first.$1, 'error');
      expect(h.messenger.calls.first.$2, "That didn't work. Please try again.");
      expect(h.messenger.calls.first.$2, isNot(contains('disk full')));
    });
  });
}
