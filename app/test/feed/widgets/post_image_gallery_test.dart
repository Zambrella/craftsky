import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_image_gallery.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:smooth_page_indicator/smooth_page_indicator.dart';

Future<void> _pump(
  WidgetTester tester,
  Widget child, {
  EdgeInsets viewPadding = EdgeInsets.zero,
}) {
  return tester.pumpWidget(
    ProviderScope(
      child: MaterialApp(
        builder: (context, routeChild) {
          final mediaQuery = MediaQuery.of(context);
          return MediaQuery(
            data: mediaQuery.copyWith(viewPadding: viewPadding),
            child: routeChild!,
          );
        },
        home: Scaffold(body: SizedBox(height: 500, child: child)),
      ),
    ),
  );
}

void main() {
  testWidgets('shows current alt text and updates when swiping pages', (
    tester,
  ) async {
    await _pump(
      tester,
      PostImageGallery(
        images: [
          PostImage(
            cid: 'bafkimage1',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Blue shawl laid flat',
            thumb: 'https://cdn.example.com/thumb1.jpg',
            fullsize: 'https://cdn.example.com/full1.jpg',
          ),
          PostImage(
            cid: 'bafkimage2',
            mime: 'image/png',
            size: 11,
            alt: 'Close-up stitch detail',
            thumb: 'https://cdn.example.com/thumb2.jpg',
            fullsize: 'https://cdn.example.com/full2.jpg',
          ),
        ],
      ),
    );

    expect(find.text('Blue shawl laid flat'), findsOneWidget);
    expect(find.text('1/2'), findsOneWidget);
    expect(find.byKey(const Key('post-image-gallery-count')), findsOneWidget);
    expect(find.byKey(const Key('post-image-gallery-dots')), findsOneWidget);
    expect(find.byType(SmoothPageIndicator), findsOneWidget);
    expect(find.bySemanticsLabel('Blue shawl laid flat'), findsWidgets);
    expect(find.byType(InteractiveViewer), findsWidgets);

    final pageView = tester.widget<PageView>(
      find.byKey(const Key('post-image-gallery-page-view')),
    );
    pageView.onPageChanged?.call(1);
    await tester.pump();

    expect(find.text('Close-up stitch detail'), findsOneWidget);
    expect(find.text('2/2'), findsOneWidget);
    expect(find.bySemanticsLabel('Close-up stitch detail'), findsWidgets);
  });

  testWidgets('gallery images are zoom-enabled via InteractiveViewer', (
    tester,
  ) async {
    await _pump(
      tester,
      PostImageGallery(
        images: [
          PostImage(
            cid: 'bafkimage1',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Blue shawl laid flat',
            thumb: 'https://cdn.example.com/thumb1.jpg',
            fullsize: 'https://cdn.example.com/full1.jpg',
          ),
        ],
      ),
    );

    final viewer = tester.widget<InteractiveViewer>(
      find.byType(InteractiveViewer).first,
    );
    expect(viewer.minScale, 1);
    expect(viewer.maxScale, greaterThan(1));
    expect(find.byKey(const Key('post-image-gallery-count')), findsNothing);
    expect(find.byKey(const Key('post-image-gallery-dots')), findsNothing);
  });

  testWidgets('gallery overlays account for media view padding', (
    tester,
  ) async {
    await _pump(
      tester,
      PostImageGallery(
        images: [
          PostImage(
            cid: 'bafkimage1',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Blue shawl laid flat',
            thumb: 'https://cdn.example.com/thumb1.jpg',
            fullsize: 'https://cdn.example.com/full1.jpg',
          ),
          PostImage(
            cid: 'bafkimage2',
            mime: 'image/png',
            size: 11,
            alt: 'Close-up stitch detail',
            thumb: 'https://cdn.example.com/thumb2.jpg',
            fullsize: 'https://cdn.example.com/full2.jpg',
          ),
        ],
      ),
      viewPadding: const EdgeInsets.only(
        left: 6,
        top: 24,
        right: 8,
        bottom: 34,
      ),
    );

    expect(
      tester.getTopLeft(find.byKey(const Key('post-image-gallery-count'))).dy,
      40,
    );
    final altPadding = tester.widget<Padding>(
      find.byKey(const Key('post-image-gallery-alt-text-padding')),
    );
    expect(
      altPadding.padding,
      const EdgeInsets.fromLTRB(18, 12, 20, 46),
    );
    expect(find.byType(SafeArea), findsNothing);
  });

  testWidgets('gallery page indicator has a contrast background', (
    tester,
  ) async {
    await _pump(
      tester,
      PostImageGallery(
        images: [
          PostImage(
            cid: 'bafkimage1',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Blue shawl laid flat',
          ),
          PostImage(
            cid: 'bafkimage2',
            mime: 'image/png',
            size: 11,
            alt: 'Close-up stitch detail',
          ),
        ],
      ),
    );

    final background = tester.widget<DecoratedBox>(
      find.byKey(const Key('post-image-gallery-dots')),
    );
    final decoration = background.decoration as BoxDecoration;
    expect(decoration.color, Colors.black.withValues(alpha: 0.58));

    final indicator = tester.widget<SmoothPageIndicator>(
      find.byType(SmoothPageIndicator),
    );
    expect(indicator.count, 2);
    expect(indicator.effect, isA<WormEffect>());
  });
}
