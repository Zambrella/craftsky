import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_image_carousel.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pinch_zoom/pinch_zoom.dart';
import 'package:smooth_page_indicator/smooth_page_indicator.dart';

Future<void> _pumpCarousel(
  WidgetTester tester,
  PostImageCarousel carousel, {
  double width = 320,
}) {
  return tester.pumpWidget(
    ProviderScope(
      child: MaterialApp(
        home: Scaffold(
          body: SizedBox(width: width, child: carousel),
        ),
      ),
    ),
  );
}

List<PostImage> _images(String prefix) {
  return [
    PostImage(
      cid: 'bafk${prefix}image1',
      mime: 'image/jpeg',
      size: 10,
      alt: '$prefix image one',
    ),
    PostImage(
      cid: 'bafk${prefix}image2',
      mime: 'image/jpeg',
      size: 11,
      alt: '$prefix image two',
    ),
  ];
}

void main() {
  group('computeBoundedImageHeight', () {
    test('uses stable fallback when aspect ratio is missing', () {
      final height = computeBoundedImageHeight(
        availableWidth: 320,
        aspectRatio: null,
      );

      expect(height, 320);
    });

    test('keeps 1:1 images within bounds', () {
      final height = computeBoundedImageHeight(
        availableWidth: 320,
        aspectRatio: const PostImageAspectRatio(width: 1, height: 1),
      );

      expect(height, 320);
    });

    test('clamps very tall images to max height', () {
      final height = computeBoundedImageHeight(
        availableWidth: 320,
        aspectRatio: const PostImageAspectRatio(width: 919, height: 2000),
      );

      expect(height, 420);
    });

    test('clamps very wide images to min height', () {
      final height = computeBoundedImageHeight(
        availableWidth: 320,
        aspectRatio: const PostImageAspectRatio(width: 2000, height: 919),
      );

      expect(height, 160);
    });
  });

  test('carousel max height grows with the viewport within safe bounds', () {
    expect(computeResponsiveCarouselMaxHeight(600), 420);
    expect(computeResponsiveCarouselMaxHeight(1024), 716.8);
    expect(computeResponsiveCarouselMaxHeight(1400), 720);
  });

  test('fitting portraits and landscape images remain edge-to-edge', () {
    expect(
      computePostImageFit(
        availableWidth: 320,
        aspectRatio: const PostImageAspectRatio(width: 4, height: 5),
        maxHeight: 420,
      ),
      BoxFit.cover,
    );
    expect(
      computePostImageFit(
        availableWidth: 800,
        aspectRatio: const PostImageAspectRatio(width: 16, height: 9),
        maxHeight: 420,
      ),
      BoxFit.cover,
    );
  });

  testWidgets('contains over-height portrait images on tablet surfaces', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(1200, 1024);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await _pumpCarousel(
      tester,
      PostImageCarousel(
        images: [
          PostImage(
            cid: 'bafktabletimage',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Portrait project photo',
            aspectRatio: const PostImageAspectRatio(width: 4, height: 5),
            fullsize: 'https://example.com/portrait.jpg',
          ),
        ],
      ),
      width: 800,
    );

    expect(
      tester.getSize(find.byKey(const Key('post-image-frame'))).height,
      716.8,
    );
    expect(
      tester.widget<CachedNetworkImage>(find.byType(CachedNetworkImage)).fit,
      BoxFit.contain,
    );
  });

  testWidgets('wraps post images in Instagram-style pinch zoom', (
    tester,
  ) async {
    await _pumpCarousel(
      tester,
      PostImageCarousel(
        images: [
          PostImage(
            cid: 'bafkimage1',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Blue shawl drying flat',
          ),
        ],
      ),
    );

    final zoom = tester.widget<PinchZoom>(find.byType(PinchZoom));
    expect(zoom.maxScale, 4);
    expect(zoom.zoomEnabled, isTrue);
    expect(find.bySemanticsLabel('Blue shawl drying flat'), findsOneWidget);
  });

  testWidgets('uses the soft rounded outline from embedded previews', (
    tester,
  ) async {
    await _pumpCarousel(
      tester,
      PostImageCarousel(images: _images('outlined')),
    );

    final context = tester.element(find.byType(PostImageCarousel));
    final theme = Theme.of(context);
    final radii = theme.extension<RadiusTheme>() ?? const RadiusTheme();
    final outline = tester.widget<DecoratedBox>(
      find.byKey(const Key('post-image-carousel-outline')),
    );
    final decoration = outline.decoration as BoxDecoration;

    expect(
      decoration.border,
      Border.all(color: theme.colorScheme.outlineVariant),
    );
    expect(decoration.borderRadius, BorderRadius.circular(radii.r2));

    final clip = tester.widget<ClipRRect>(
      find.ancestor(
        of: find.byKey(const Key('post-image-carousel')),
        matching: find.byType(ClipRRect),
      ),
    );
    expect(clip.borderRadius, BorderRadius.circular(radii.r2));
  });

  testWidgets('uses high-contrast worm page indicators', (tester) async {
    await _pumpCarousel(
      tester,
      PostImageCarousel(
        images: [
          PostImage(
            cid: 'bafkimage1',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Blue shawl drying flat',
          ),
          PostImage(
            cid: 'bafkimage2',
            mime: 'image/jpeg',
            size: 11,
            alt: 'Close-up stitch detail',
          ),
        ],
      ),
    );

    final background = tester.widget<DecoratedBox>(
      find.byKey(const Key('post-image-dots')),
    );
    final decoration = background.decoration as BoxDecoration;
    expect(decoration.color, Colors.black.withValues(alpha: 0.58));
    expect(find.byType(SmoothPageIndicator), findsOneWidget);

    final indicator = tester.widget<SmoothPageIndicator>(
      find.byType(SmoothPageIndicator),
    );
    expect(indicator.count, 2);
    expect(indicator.effect, isA<WormEffect>());
  });

  testWidgets('uses stable unique Hero tags for each image', (tester) async {
    final carousel = PostImageCarousel(images: _images('hero'));
    await _pumpCarousel(tester, carousel);

    final firstTag = tester.widget<Hero>(find.byType(Hero)).tag;

    await tester.drag(
      find.byKey(const Key('post-image-carousel')),
      const Offset(-500, 0),
    );
    await tester.pumpAndSettle();
    final secondTag = tester.widget<Hero>(find.byType(Hero)).tag;
    expect(secondTag, isNot(same(firstTag)));

    await tester.drag(
      find.byKey(const Key('post-image-carousel')),
      const Offset(500, 0),
    );
    await tester.pumpAndSettle();
    expect(tester.widget<Hero>(find.byType(Hero)).tag, same(firstTag));
  });

  testWidgets('newly visible carousels start on the first image', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(400, 600));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          home: Scaffold(
            body: ListView.builder(
              itemCount: 3,
              itemBuilder: (context, index) {
                return switch (index) {
                  0 => SizedBox(
                    width: 320,
                    child: PostImageCarousel(images: _images('first')),
                  ),
                  1 => const SizedBox(height: 500),
                  _ => SizedBox(
                    width: 320,
                    child: PostImageCarousel(images: _images('second')),
                  ),
                };
              },
            ),
          ),
        ),
      ),
    );

    await tester.drag(
      find.byKey(const Key('post-image-carousel')),
      const Offset(-500, 0),
    );
    await tester.pumpAndSettle();
    expect(find.text('2/2'), findsOneWidget);

    await tester.drag(find.byType(ListView), const Offset(0, -900));
    await tester.pumpAndSettle();

    expect(find.text('1/2'), findsOneWidget);
  });
}
