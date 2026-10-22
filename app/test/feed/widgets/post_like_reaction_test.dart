import 'package:craftsky_app/feed/widgets/post_like_reaction.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:rive/rive.dart' as rive;

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late rive.File file;
  setUpAll(() async {
    await rive.RiveNative.init();
    file = (await rive.File.asset(
      'assets/animations/like.riv',
      riveFactory: rive.Factory.flutter,
    ))!;
  });
  tearDownAll(() => file.dispose());

  Future<ProviderContainer> containerForTest() async {
    final container = ProviderContainer(
      overrides: [
        likeAnimationFileProvider.overrideWith((ref) async => file),
      ],
    );
    addTearDown(container.dispose);
    await container.read(likeAnimationFileProvider.future);
    return container;
  }

  Widget scene(
    ProviderContainer container, {
    bool liked = false,
    bool cancelled = false,
    bool reduced = false,
    String identity = 'first-post',
    VoidCallback? onLike,
    VoidCallback? onBodyTap,
  }) => UncontrolledProviderScope(
    container: container,
    child: MaterialApp(
      home: MediaQuery(
        data: MediaQueryData(disableAnimations: reduced),
        child: Scaffold(
          body: Center(
            child: SizedBox(
              width: 360,
              height: 300,
              child: PostLikeReaction(
                key: ValueKey(identity),
                liked: liked,
                cancelled: cancelled,
                onLike: onLike,
                builder: (context, react) => SizedBox.expand(
                  child: Column(
                    children: [
                      TextButton(onPressed: react, child: const Text('Like')),
                      Expanded(
                        child: Center(
                          child: TextButton(
                            onPressed: onBodyTap,
                            child: const Text('Post content'),
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    ),
  );

  testWidgets('plays the real asset centrally without intercepting taps', (
    tester,
  ) async {
    final container = await containerForTest();
    var likes = 0;
    var bodyTaps = 0;
    await tester.pumpWidget(
      scene(container, onLike: () => likes++, onBodyTap: () => bodyTaps++),
    );
    expect(find.byKey(const Key('post-like-reaction')), findsNothing);
    final initialSize = tester.getSize(find.byType(PostLikeReaction));
    await tester.tap(find.text('Like'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    expect(likes, 1);
    expect(find.byType(rive.RiveWidget), findsOneWidget);
    final animation = find.byType(rive.RiveWidget);
    expect(
      tester.getCenter(animation),
      tester.getCenter(find.byType(PostLikeReaction)),
    );
    expect(tester.getSize(find.byType(PostLikeReaction)), initialSize);
    expect(
      tester.widget<rive.RiveWidget>(animation).controller.artboard.name,
      'Like',
    );
    await tester.tap(find.text('Post content'));
    await tester.pump();
    expect(bodyTaps, 1);
    await tester.pump(const Duration(milliseconds: 1000));
    expect(find.byKey(const Key('post-like-reaction')), findsNothing);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets(
    'centres feedback in the visible post and follows scrolling without replay',
    (tester) async {
      final container = await containerForTest();
      final scroll = ScrollController();
      addTearDown(scroll.dispose);
      const viewportKey = Key('feed-viewport');
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            home: Scaffold(
              body: Center(
                child: SizedBox(
                  key: viewportKey,
                  width: 360,
                  height: 300,
                  child: SingleChildScrollView(
                    controller: scroll,
                    child: Column(
                      children: [
                        const SizedBox(height: 200),
                        RepaintBoundary(
                          child: SizedBox(
                            height: 800,
                            child: PostLikeReaction(
                              liked: false,
                              onLike: () {},
                              builder: (context, react) => Align(
                                alignment: Alignment.topCenter,
                                child: TextButton(
                                  onPressed: react,
                                  child: const Text('Like'),
                                ),
                              ),
                            ),
                          ),
                        ),
                        const SizedBox(height: 600),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Like'));
      await tester.pump();
      final animation = find.byType(rive.RiveWidget);
      final controller = tester.widget<rive.RiveWidget>(animation).controller;
      final viewport = tester.getRect(find.byKey(viewportKey));

      void expectVisibleCentre() {
        final post = tester.getRect(find.byType(PostLikeReaction));
        final visible = post.intersect(viewport);
        final heart = tester.getRect(animation);
        expect(heart.center, visible.center);
        expect(heart.left, greaterThanOrEqualTo(visible.left));
        expect(heart.top, greaterThanOrEqualTo(visible.top));
        expect(heart.right, lessThanOrEqualTo(visible.right));
        expect(heart.bottom, lessThanOrEqualTo(visible.bottom));
        expect(
          tester.widget<rive.RiveWidget>(animation).controller,
          controller,
        );
      }

      // Only the top 100 pixels of this tall post are initially visible.
      expectVisibleCentre();
      scroll.jumpTo(250);
      await tester.pump();
      expectVisibleCentre();
      scroll.jumpTo(900);
      await tester.pump();
      expectVisibleCentre();

      // The card is still mounted but fully outside the viewport.
      scroll.jumpTo(1050);
      await tester.pump();
      expect(find.byType(PostLikeReaction), findsOneWidget);
      expect(animation.hitTestable(), findsNothing);
      scroll.jumpTo(250);
      await tester.pump();
      expectVisibleCentre();
      await tester.pump(const Duration(milliseconds: 1500));
      expect(find.byKey(const Key('post-like-reaction')), findsNothing);
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets(
    'rapid unlike, replay, rollback and recycled cards clear feedback',
    (
      tester,
    ) async {
      final container = await containerForTest();
      await tester.pumpWidget(scene(container, onLike: () {}));
      await tester.tap(find.text('Like'));
      await tester.pump();
      final first = tester
          .widget<rive.RiveWidget>(find.byType(rive.RiveWidget))
          .controller;

      await tester.pumpWidget(scene(container, liked: true, onLike: () {}));
      await tester.tap(find.text('Like'));
      await tester.pump();
      expect(find.byType(rive.RiveWidget), findsNothing);
      await tester.pumpWidget(scene(container, onLike: () {}));
      await tester.tap(find.text('Like'));
      await tester.pump();
      final replay = tester
          .widget<rive.RiveWidget>(find.byType(rive.RiveWidget))
          .controller;
      expect(identical(first, replay), isFalse);
      await tester.pumpWidget(scene(container, cancelled: true, onLike: () {}));
      expect(find.byKey(const Key('post-like-reaction')), findsNothing);

      await tester.pumpWidget(scene(container, onLike: () {}));
      await tester.tap(find.text('Like'));
      await tester.pump();
      await tester.pumpWidget(
        scene(container, identity: 'another-post-or-session', onLike: () {}),
      );
      expect(find.byKey(const Key('post-like-reaction')), findsNothing);
      await tester.pump(const Duration(seconds: 2));
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets('reduced motion holds a static stitched heart briefly', (
    tester,
  ) async {
    final container = await containerForTest();
    await tester.pumpWidget(scene(container, reduced: true, onLike: () {}));
    await tester.tap(find.text('Like'));
    await tester.pump();
    final animation = find.byType(rive.RiveWidget);
    final controller = tester.widget<rive.RiveWidget>(animation).controller;
    expect(controller.active, isFalse);
    expect(TickerMode.valuesOf(tester.element(animation)).enabled, isFalse);
    await tester.pump(const Duration(milliseconds: 450));
    expect(find.byKey(const Key('post-like-reaction')), findsNothing);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('asset failure still allows liking and never shows a spinner', (
    tester,
  ) async {
    final container = ProviderContainer(
      retry: (count, error) => null,
      overrides: [
        likeAnimationFileProvider.overrideWith(
          (ref) async => throw const FormatException('Unavailable test asset'),
        ),
      ],
    );
    addTearDown(container.dispose);
    await container
        .read(likeAnimationFileProvider.future)
        .catchError((Object _) => file);
    var likes = 0;
    await tester.pumpWidget(scene(container, onLike: () => likes++));
    await tester.tap(find.text('Like'));
    await tester.pump();
    expect(likes, 1);
    expect(find.byKey(const Key('post-like-reaction')), findsOneWidget);
    expect(find.byType(rive.RiveWidget), findsNothing);
    expect(find.byType(CircularProgressIndicator), findsNothing);
    await tester.pump(const Duration(milliseconds: 1500));
    expect(find.byKey(const Key('post-like-reaction')), findsNothing);
    await tester.pumpWidget(const SizedBox());
  });
}
