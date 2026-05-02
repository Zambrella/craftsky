import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/profile/widgets/profile_banner.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/image_cache_fakes.dart';

Widget _wrap(Widget child, {List<dynamic> overrides = const []}) {
  return ProviderScope(
    overrides: List.from(overrides),
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      home: Scaffold(body: child),
    ),
  );
}

void main() {
  group('ProfileBanner', () {
    testWidgets('renders no CachedNetworkImage when bannerUrl is null', (
      tester,
    ) async {
      await tester.pumpWidget(
        _wrap(const ProfileBanner(color: Color(0xFFCC8866))),
      );

      expect(find.byType(CachedNetworkImage), findsNothing);
    });

    testWidgets('mounts CachedNetworkImage with the profile cache manager '
        'when bannerUrl is set', (tester) async {
      final fake = FakeBaseCacheManager();

      await tester.pumpWidget(
        _wrap(
          const ProfileBanner(
            color: Color(0xFFCC8866),
            bannerUrl: 'https://example.test/banner.jpg',
          ),
          overrides: [
            profileImageCacheManagerProvider.overrideWith((ref) => fake),
          ],
        ),
      );
      await tester.pump();

      final image = tester.widget<CachedNetworkImage>(
        find.byType(CachedNetworkImage),
      );
      expect(image.imageUrl, 'https://example.test/banner.jpg');
      expect(image.cacheManager, same(fake));
      expect(image.fit, BoxFit.cover);
    });

    testWidgets('respects the height parameter', (tester) async {
      await tester.pumpWidget(
        _wrap(const ProfileBanner(color: Color(0xFFCC8866), height: 200)),
      );

      // Find the outermost Container that carries the swatch colour and
      // explicit height — that's the banner.
      final containers = tester.widgetList<Container>(find.byType(Container));
      final banner = containers.firstWhere(
        (c) => c.constraints?.maxHeight == 200,
      );
      expect(banner.color, const Color(0xFFCC8866));
    });
  });
}
