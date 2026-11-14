import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/inline_pinch_zoom.dart';
import 'package:craftsky_app/feed/widgets/post_image_carousel.dart';
import 'package:craftsky_app/shared/widgets/root_overlay_scope.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
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
  testWidgets('swiping a mixed-ratio album preserves its frame height', (
    tester,
  ) async {
    await _pumpCarousel(
      tester,
      PostImageCarousel(
        images: [
          PostImage(
            cid: 'portrait',
            mime: 'image/jpeg',
            size: 1,
            alt: 'Portrait',
            aspectRatio: const PostImageAspectRatio(width: 4, height: 5),
          ),
          PostImage(
            cid: 'landscape',
            mime: 'image/jpeg',
            size: 1,
            alt: 'Landscape',
            aspectRatio: const PostImageAspectRatio(width: 10, height: 1),
          ),
        ],
      ),
    );
    final frame = find.byKey(const Key('post-image-frame'));
    final height = tester.getSize(frame).height;
    await tester.drag(find.byType(PageView), const Offset(-320, 0));
    await tester.pumpAndSettle();
    expect(find.text('2/2'), findsOneWidget);
    expect(tester.getSize(frame).height, height);
  });

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
        aspectRatio: const PostImageAspectRatio(width: 800, height: 2000),
      );

      expect(height, 700);
    });

    test('preserves very wide images without a minimum height', () {
      final height = computeBoundedImageHeight(
        availableWidth: 320,
        aspectRatio: const PostImageAspectRatio(width: 2000, height: 919),
      );

      expect(height, 320 * 919 / 2000);
    });
  });

  test('carousel max height grows with the viewport within safe bounds', () {
    expect(computeResponsiveCarouselMaxHeight(600), 390);
    expect(computeResponsiveCarouselMaxHeight(300), 195);
    expect(computeResponsiveCarouselMaxHeight(1024), 665.6);
    expect(computeResponsiveCarouselMaxHeight(1400), 700);
  });

  for (final ratio in [
    const PostImageAspectRatio(width: 4, height: 5),
    const PostImageAspectRatio(width: 10, height: 1),
    null,
  ]) {
    testWidgets('contains the whole image regardless of ratio ($ratio)', (
      tester,
    ) async {
      await _pumpCarousel(
        tester,
        PostImageCarousel(
          images: [
            PostImage(
              cid: 'whole',
              mime: 'image/jpeg',
              size: 1,
              alt: '',
              aspectRatio: ratio,
              fullsize: 'https://example.com/whole.jpg',
            ),
          ],
        ),
      );
      expect(
        tester.widget<CachedNetworkImage>(find.byType(CachedNetworkImage)).fit,
        BoxFit.contain,
      );
    });
  }

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
      665.6,
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

    final zoom = tester.widget<InlinePinchZoom>(find.byType(InlinePinchZoom));
    final zoomContext = tester.element(find.byType(InlinePinchZoom));
    expect(zoom.maxScale, 4);
    expect(zoom.barrierColor, isNull);
    expect(zoom.resetDuration, const Duration(milliseconds: 300));
    expect(
      Overlay.of(zoom.overlayContext!),
      same(Overlay.of(zoomContext, rootOverlay: true)),
    );
    expect(find.bySemanticsLabel('Blue shawl drying flat'), findsOneWidget);
    expect(find.byKey(const Key('post-image-count')), findsNothing);
    expect(find.byKey(const Key('post-image-dots')), findsNothing);
  });

  testWidgets('pinch lifts the image into the root overlay until release', (
    tester,
  ) async {
    await _pumpCarousel(
      tester,
      PostImageCarousel(
        images: [
          PostImage(
            cid: 'bafkoverlayimage',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Quilt detail in overlay',
          ),
        ],
      ),
    );

    final center = tester.getCenter(find.byType(InlinePinchZoom));
    final firstFinger = await tester.createGesture(pointer: 1);
    final secondFinger = await tester.createGesture(pointer: 2);
    await firstFinger.down(center - const Offset(20, 0));
    await secondFinger.down(center + const Offset(20, 0));
    await firstFinger.moveTo(center - const Offset(60, 0));
    await secondFinger.moveTo(center + const Offset(60, 0));
    await tester.pump();
    final scrim = Theme.of(
      tester.element(find.byType(InlinePinchZoom)),
    ).colorScheme.scrim.withValues(alpha: 0.12);

    expect(find.bySemanticsLabel('Quilt detail in overlay'), findsOneWidget);
    expect(
      find.descendant(
        of: find.byType(InlinePinchZoom),
        matching: find.byWidgetPredicate(
          (widget) => widget is Opacity && widget.opacity == 0,
        ),
      ),
      findsOneWidget,
    );
    expect(
      find.byWidgetPredicate(
        (widget) => widget is ModalBarrier && widget.color == scrim,
      ),
      findsOneWidget,
    );
    expect(
      tester.getSize(find.bySemanticsLabel('Quilt detail in overlay')),
      const Size(320, 320),
    );

    await firstFinger.up();
    await secondFinger.up();
    await tester.pump();
    await tester.pump(const Duration(seconds: 1));

    expect(find.bySemanticsLabel('Quilt detail in overlay'), findsOneWidget);
    expect(
      find.byWidgetPredicate(
        (widget) => widget is ModalBarrier && widget.color == scrim,
      ),
      findsNothing,
    );
  });

  testWidgets('pinching does not change the carousel page', (tester) async {
    await _pumpCarousel(
      tester,
      PostImageCarousel(images: _images('carousel')),
    );
    final center = tester.getCenter(find.byType(InlinePinchZoom));
    final first = await tester.startGesture(center - const Offset(20, 0));
    final second = await tester.startGesture(
      center + const Offset(20, 0),
      pointer: 4,
    );
    await first.moveBy(const Offset(-80, -10));
    await second.moveBy(const Offset(-80, 10));
    await tester.pump(const Duration(milliseconds: 300));
    expect(find.text('1/2'), findsOneWidget);

    await first.up();
    await second.up();
    await tester.pump();
    await tester.pump(const Duration(seconds: 1));
    await tester.drag(
      find.byKey(const Key('post-image-carousel')),
      const Offset(-500, 0),
    );
    await tester.pump(const Duration(milliseconds: 300));
    expect(find.text('2/2'), findsOneWidget);
  });

  testWidgets('pinch works after the first finger starts scrolling the feed', (
    tester,
  ) async {
    final controller = ScrollController();
    addTearDown(controller.dispose);
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          home: Scaffold(
            body: ListView(
              controller: controller,
              children: [
                const SizedBox(height: 100),
                Align(
                  alignment: Alignment.centerLeft,
                  child: SizedBox(
                    width: 320,
                    child: PostImageCarousel(images: _images('scrolling')),
                  ),
                ),
                const SizedBox(height: 1000),
              ],
            ),
          ),
        ),
      ),
    );

    final center = tester.getCenter(find.byType(InlinePinchZoom));
    final first = await tester.startGesture(center - const Offset(20, 0));
    await first.moveBy(const Offset(0, -30));
    await tester.pump();
    await first.moveBy(const Offset(0, -20));
    await tester.pump();
    expect(controller.offset, greaterThan(0));
    expect(
      tester
          .getRect(find.byType(InlinePinchZoom))
          .contains(
            center + const Offset(20, 0),
          ),
      isTrue,
    );

    final second = await tester.startGesture(
      center + const Offset(20, 0),
      pointer: 2,
    );
    await tester.pump();
    final scrim = Theme.of(
      tester.element(find.byType(InlinePinchZoom)),
    ).colorScheme.scrim.withValues(alpha: 0.12);
    expect(
      find.byWidgetPredicate(
        (widget) => widget is ModalBarrier && widget.color == scrim,
      ),
      findsOneWidget,
    );
    final heldOffset = controller.offset;
    await first.moveBy(const Offset(-35, -20));
    await second.moveBy(const Offset(35, 20));
    await tester.pump();
    expect(controller.offset, heldOffset);
    await first.moveBy(const Offset(0, -60));
    await second.moveBy(const Offset(0, -60));
    await tester.pump();
    expect(controller.offset, heldOffset);
    controller.jumpTo(heldOffset + 40);
    expect(controller.offset, heldOffset);
    await first.moveBy(const Offset(0, -60));
    await second.moveBy(const Offset(0, -60));
    await tester.pump();
    expect(controller.offset, heldOffset);
    expect(
      tester
          .widgetList<Transform>(find.byType(Transform))
          .any(
            (transform) => transform.transform.getMaxScaleOnAxis() > 1,
          ),
      isTrue,
    );
    expect(
      find.byWidgetPredicate(
        (widget) => widget is ModalBarrier && widget.color == scrim,
      ),
      findsOneWidget,
    );

    await first.up();
    await second.moveBy(const Offset(0, -40));
    await tester.pump();
    expect(controller.offset, heldOffset);
    await second.up();
    await tester.pump();
    await tester.pump(const Duration(seconds: 1));
    expect(
      find.byWidgetPredicate(
        (widget) => widget is ModalBarrier && widget.color == scrim,
      ),
      findsNothing,
    );
    await tester.drag(find.byType(ListView), const Offset(0, -80));
    await tester.pump();
    expect(controller.offset, greaterThan(heldOffset));
  });

  testWidgets('a second finger outside the image does not start zoom', (
    tester,
  ) async {
    await _pumpCarousel(
      tester,
      PostImageCarousel(images: _images('outside')),
    );
    final first = await tester.startGesture(
      tester.getCenter(find.byType(InlinePinchZoom)),
    );
    final second = await tester.startGesture(
      const Offset(500, 500),
      pointer: 3,
    );
    await tester.pump();

    final scrim = Theme.of(
      tester.element(find.byType(InlinePinchZoom)),
    ).colorScheme.scrim.withValues(alpha: 0.12);
    expect(
      find.byWidgetPredicate(
        (widget) => widget is ModalBarrier && widget.color == scrim,
      ),
      findsNothing,
    );
    await first.up();
    await second.up();
  });

  testWidgets('targets the scoped overlay above a nested navigator', (
    tester,
  ) async {
    late OverlayState rootOverlay;
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          home: Builder(
            builder: (rootContext) {
              rootOverlay = Overlay.of(rootContext);
              return RootOverlayScope(
                overlayContext: rootContext,
                child: Navigator(
                  onGenerateRoute: (_) => MaterialPageRoute<void>(
                    builder: (_) => Scaffold(
                      body: SizedBox(
                        width: 320,
                        child: PostImageCarousel(
                          images: [
                            PostImage(
                              cid: 'bafknestedoverlay',
                              mime: 'image/jpeg',
                              size: 10,
                              alt: 'Nested navigator image',
                            ),
                          ],
                        ),
                      ),
                    ),
                  ),
                ),
              );
            },
          ),
        ),
      ),
    );

    final zoom = tester.widget<InlinePinchZoom>(find.byType(InlinePinchZoom));
    final nestedContext = tester.element(find.byType(InlinePinchZoom));
    expect(Overlay.of(nestedContext), isNot(same(rootOverlay)));
    expect(Overlay.of(zoom.overlayContext!), same(rootOverlay));
  });

  testWidgets('uses the soft rounded outline from embedded previews', (
    tester,
  ) async {
    await _pumpCarousel(tester, PostImageCarousel(images: _images('outlined')));

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

  testWidgets('alt badge follows the current image and preserves image taps', (
    tester,
  ) async {
    int? tappedIndex;
    await _pumpCarousel(
      tester,
      PostImageCarousel(
        images: [
          for (final alt in ['Stitch detail', '', '   '])
            PostImage(cid: 'bafkimage', mime: 'image/jpeg', size: 10, alt: alt),
        ],
        onImageTap: (index, _) => tappedIndex = index,
      ),
    );

    final badge = find.byKey(const Key('post-image-alt-text-badge'));
    expect(badge, findsOneWidget);
    await tester.tapAt(tester.getCenter(badge));
    await tester.pump(const Duration(milliseconds: 350));
    expect(tappedIndex, 0);

    for (var index = 1; index < 3; index++) {
      await tester.drag(
        find.byKey(const Key('post-image-carousel')),
        const Offset(-500, 0),
      );
      await tester.pumpAndSettle();
      expect(badge, findsNothing);
    }
  });

  testWidgets('uses stable unique Hero tags for each image', (tester) async {
    final carousel = PostImageCarousel(images: _images('hero'));
    await _pumpCarousel(tester, carousel);

    final firstTag = tester.widget<Hero>(find.byType(Hero)).tag;

    await tester.drag(
      find.byKey(const Key('post-image-carousel')),
      const Offset(-500, 0),
    );
    await tester.pump(const Duration(milliseconds: 300));
    final secondTag = tester.widget<Hero>(find.byType(Hero)).tag;
    expect(secondTag, isNot(same(firstTag)));

    await tester.drag(
      find.byKey(const Key('post-image-carousel')),
      const Offset(500, 0),
    );
    await tester.pump(const Duration(milliseconds: 300));
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
    await tester.pump(const Duration(milliseconds: 300));
    expect(find.text('2/2'), findsOneWidget);

    await tester.drag(find.byType(ListView), const Offset(0, -900));
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.text('1/2'), findsOneWidget);
  });
}
