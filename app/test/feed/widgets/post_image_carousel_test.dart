import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_image_carousel.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pinch_zoom/pinch_zoom.dart';
import 'package:smooth_page_indicator/smooth_page_indicator.dart';

Future<void> _pumpCarousel(WidgetTester tester, PostImageCarousel carousel) {
  return tester.pumpWidget(
    ProviderScope(
      child: MaterialApp(
        home: Scaffold(
          body: SizedBox(width: 320, child: carousel),
        ),
      ),
    ),
  );
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
}
