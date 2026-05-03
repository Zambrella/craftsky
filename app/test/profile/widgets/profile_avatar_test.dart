import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/image_cache_fakes.dart';

Widget _wrap(Widget child, {List<dynamic> overrides = const []}) {
  return ProviderScope(
    // ProviderScope.overrides is List<Override>; Override is @internal in
    // riverpod 3.x so we cast here to satisfy the type system without
    // naming the sealed class directly.
    overrides: List.from(overrides),
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      home: Scaffold(body: Center(child: child)),
    ),
  );
}

void main() {
  group('ProfileAvatar', () {
    testWidgets('renders the initial-letter fallback when avatarUrl is null', (
      tester,
    ) async {
      await tester.pumpWidget(_wrap(const ProfileAvatar(seed: 'Alice')));

      expect(find.text('A'), findsOneWidget);
      expect(find.byType(Image), findsNothing);
    });

    testWidgets('renders "?" when seed is empty and avatarUrl is null', (
      tester,
    ) async {
      await tester.pumpWidget(_wrap(const ProfileAvatar(seed: '')));

      expect(find.text('?'), findsOneWidget);
    });

    testWidgets(
      'mounts an Image with a CachedNetworkImageProvider pointing at the '
      'profile cache manager when avatarUrl is set',
      (tester) async {
        final fake = FakeBaseCacheManager();

        await tester.pumpWidget(
          _wrap(
            const ProfileAvatar(
              seed: 'Bob',
              avatarUrl: 'https://example.test/b.jpg',
            ),
            overrides: [
              profileImageCacheManagerProvider.overrideWith((ref) => fake),
            ],
          ),
        );
        await tester.pump();

        final image = tester.widget<Image>(find.byType(Image));
        final provider = image.image;
        expect(provider, isA<CachedNetworkImageProvider>());
        provider as CachedNetworkImageProvider;
        expect(provider.url, 'https://example.test/b.jpg');
        expect(provider.cacheManager, same(fake));
        expect(image.fit, BoxFit.cover);
      },
    );

    testWidgets('shows the initial-letter placeholder while loading', (
      tester,
    ) async {
      // Default fake: getFileStream returns an empty, never-closing stream,
      // so CachedNetworkImage sits on its placeholder.
      final fake = FakeBaseCacheManager();

      await tester.pumpWidget(
        _wrap(
          const ProfileAvatar(
            seed: 'Cara',
            avatarUrl: 'https://example.test/c.jpg',
          ),
          overrides: [
            profileImageCacheManagerProvider.overrideWith((ref) => fake),
          ],
        ),
      );
      await tester.pump();

      // Initial letter visible during load.
      expect(find.text('C'), findsOneWidget);
    });

    testWidgets('shows the initial-letter on cache error', (tester) async {
      final fake = FakeBaseCacheManager()..nextStream = (_) => erroringStream();

      await tester.pumpWidget(
        _wrap(
          const ProfileAvatar(
            seed: 'Dan',
            avatarUrl: 'https://example.test/d.jpg',
          ),
          overrides: [
            profileImageCacheManagerProvider.overrideWith((ref) => fake),
          ],
        ),
      );
      // Pump twice: once to mount, once to let the error propagate.
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('D'), findsOneWidget);
    });

    testWidgets('preserves chunky border + butter background', (tester) async {
      await tester.pumpWidget(_wrap(const ProfileAvatar(seed: 'E')));

      // Find the outermost Container (the one decorated with the circle
      // shape + border + shadow). It is the first Container ancestor
      // of the rendered Text 'E' that has a circular BoxDecoration.
      final containers = tester.widgetList<Container>(find.byType(Container));
      final circle = containers.firstWhere((c) {
        final d = c.decoration;
        return d is BoxDecoration && d.shape == BoxShape.circle;
      });
      final decoration = circle.decoration! as BoxDecoration;

      // Butter background present, ink border applied.
      final swatches = AppTheme.lightThemeData.extension<BrandSwatchTheme>()!;
      expect(decoration.color, swatches.butter);
      expect(decoration.border, isNotNull);
      expect(decoration.boxShadow, isNotEmpty);
    });
  });
}
