import 'dart:async';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:craftsky_app/auth/models/session_registry.dart' as registry;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/yarn_refresh_scroll_view.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:rive/rive.dart' as rive;

class _Storage implements SessionRegistryStorage {
  @override
  Future<registry.SessionRegistry> read() async =>
      registry.SessionRegistry.empty();
  @override
  Future<void> write(registry.SessionRegistry value) async {}
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late rive.File file;
  setUpAll(() async {
    await rive.RiveNative.init();
    file = (await rive.File.asset(
      'assets/animations/yarn_refresh.riv',
      // These tests cover gestures and animation state in the headless runner.
      // The device's Rive renderer still needs iOS release-build verification.
      riveFactory: rive.Factory.flutter,
    ))!;
  });
  tearDownAll(() => file.dispose());

  test('default loader works without a native render context', () async {
    // Exercise the real provider: the gesture tests below intentionally supply
    // their own Flutter-rendered file and would miss a native factory crash.
    expect(rive.Factory.rive.isSupported, isFalse);
    final container = ProviderContainer.test(retry: (_, _) => null);
    final loaded = await container.read(yarnRefreshFileProvider.future);

    final artboard = loaded.artboardAt(0)!;
    addTearDown(artboard.dispose);
    expect(artboard.riveFactory, same(rive.Factory.flutter));
  });

  Future<ProviderContainer> setup({bool broken = false}) async {
    final container = ProviderContainer(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(_Storage()),
        yarnRefreshFileProvider.overrideWith((ref) async {
          if (broken) throw const FormatException('Test unavailable yarn');
          return file;
        }),
      ],
    );
    await container
        .read(yarnRefreshFileProvider.future)
        .catchError((Object _) => file);
    addTearDown(container.dispose);
    return container;
  }

  Widget scene(
    ProviderContainer container, {
    required Future<void> Function() refresh,
    bool reduced = false,
    bool allowed = true,
    bool showSuccess = true,
  }) => UncontrolledProviderScope(
    container: container,
    child: MaterialApp(
      localizationsDelegates: appLocalizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: MediaQuery(
        data: MediaQueryData(disableAnimations: reduced),
        child: RepaintBoundary(
          key: const Key('yarn-capture'),
          child: Scaffold(
            body: YarnRefreshScrollView(
              onRefresh: refresh,
              showSuccess: showSuccess,
              notificationPredicate: (notification) =>
                  allowed && defaultScrollNotificationPredicate(notification),
              headerSliverCount: 1,
              slivers: const [
                SliverAppBar(title: Text('Feed'), pinned: true),
                SliverToBoxAdapter(
                  child: SizedBox(
                    height: 900,
                    child: Text('First post', key: Key('first-post')),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    ),
  );
  YarnRefreshGraphicStateForTesting graphic(WidgetTester tester) =>
      tester.state(find.byKey(const Key('yarn-refresh-graphic')))
          as YarnRefreshGraphicStateForTesting;
  Future<void> clean(WidgetTester tester, ProviderContainer container) async {
    await tester.pumpWidget(const SizedBox());
    container.dispose();
    await tester.pump();
  }

  testWidgets(
    'strand follows the real pull, reverses, and cancels below threshold',
    (tester) async {
      final container = await setup();
      var calls = 0;
      await tester.pumpWidget(
        scene(
          container,
          refresh: () async {
            calls++;
          },
        ),
      );
      await tester.pumpAndSettle();
      final gesture = await tester.startGesture(const Offset(300, 180));
      await gesture.moveBy(const Offset(0, 100));
      await tester.pump();
      final length = graphic(tester).strandLength;
      expect(graphic(tester).ballScale, 1);
      final headerY = tester.getTopLeft(find.text('Feed')).dy;
      expect(length, greaterThan(0));
      await gesture.moveBy(const Offset(0, 55));
      await tester.pump();
      expect(graphic(tester).strandLength, greaterThan(length));
      expect(graphic(tester).ballScale, 1);
      expect(tester.getTopLeft(find.text('Feed')).dy, headerY);
      await gesture.moveBy(const Offset(0, -70));
      await tester.pump();
      expect(graphic(tester).strandLength, lessThan(length));
      await gesture.up();
      await tester.pumpAndSettle();
      expect(calls, 0);
      await clean(tester, container);
    },
  );

  testWidgets(
    'refresh waits for release and the real future, then shows success',
    (tester) async {
      final container = await setup();
      final pending = Completer<void>();
      var calls = 0;
      await tester.pumpWidget(
        scene(
          container,
          refresh: () {
            calls++;
            return pending.future;
          },
        ),
      );
      await tester.pumpAndSettle();
      final gesture = await tester.startGesture(const Offset(300, 180));
      await gesture.moveBy(const Offset(0, 380));
      await tester.pump();
      await tester.pump();
      expect(calls, 0);
      await gesture.up();
      await tester.pump();
      expect(calls, 1);
      await tester.pump(const Duration(milliseconds: 600));
      expect(graphic(tester).showsSuccess, isFalse);
      final rotation = graphic(tester).rotation;
      await tester.pump(const Duration(milliseconds: 150));
      expect(graphic(tester).rotation, isNot(rotation));
      pending.complete();
      await tester.pump();
      await tester.pump();
      expect(graphic(tester).successOpacity, 0);
      await tester.pump(const Duration(milliseconds: 60));
      expect(graphic(tester).successOpacity, closeTo(0.5, 0.02));
      expect(graphic(tester).opacity, closeTo(0.5, 0.02));
      await tester.pump(const Duration(milliseconds: 60));
      expect(graphic(tester).showsSuccess, isTrue);
      expect(graphic(tester).opacity, 0);
      await tester.pump(const Duration(milliseconds: 250));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('yarn-refresh-graphic')), findsNothing);
      await clean(tester, container);
    },
  );

  testWidgets('reduced motion holds the yarn still during refresh', (
    tester,
  ) async {
    final container = await setup();
    final pending = Completer<void>();
    await tester.pumpWidget(
      scene(container, reduced: true, refresh: () => pending.future),
    );
    await tester.pumpAndSettle();
    await tester.drag(find.byType(CustomScrollView), const Offset(0, 400));
    await tester.pump(const Duration(milliseconds: 700));
    expect(graphic(tester).rotation, 0);
    await tester.pump(const Duration(milliseconds: 400));
    expect(graphic(tester).rotation, 0);
    pending.complete();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 250));
    await tester.pumpAndSettle();
    await clean(tester, container);
  });

  testWidgets('guarded drags and cancelled pointers never refresh', (
    tester,
  ) async {
    final container = await setup();
    var calls = 0;
    await tester.pumpWidget(
      scene(
        container,
        allowed: false,
        refresh: () async {
          calls++;
        },
      ),
    );
    await tester.pumpAndSettle();
    await tester.drag(find.byType(CustomScrollView), const Offset(0, 400));
    await tester.pumpAndSettle();
    expect(calls, 0);
    await tester.pumpWidget(
      scene(
        container,
        refresh: () async {
          calls++;
        },
      ),
    );
    await tester.pumpAndSettle();
    final gesture = await tester.startGesture(const Offset(300, 180));
    await gesture.moveBy(const Offset(0, 400));
    await tester.pump();
    await gesture.cancel();
    await tester.pumpAndSettle();
    expect(calls, 0);
    await clean(tester, container);
  });

  testWidgets('request failure and route disposal do not show success', (
    tester,
  ) async {
    final container = await setup();
    final pending = Completer<void>();
    await tester.pumpWidget(scene(container, refresh: () => pending.future));
    await tester.pumpAndSettle();
    await tester.drag(find.byType(CustomScrollView), const Offset(0, 400));
    await tester.pump(const Duration(milliseconds: 500));
    pending.completeError(StateError('Test refresh failure'));
    await tester.pump();
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('yarn-refresh-graphic')), findsNothing);
    expect(tester.takeException(), isNull);
    final later = Completer<void>();
    await tester.pumpWidget(scene(container, refresh: () => later.future));
    await tester.pumpAndSettle();
    await tester.drag(find.byType(CustomScrollView), const Offset(0, 400));
    await tester.pump();
    await tester.pumpWidget(const SizedBox());
    later.complete();
    await tester.pump();
    container.dispose();
    await tester.pump();
    expect(tester.takeException(), isNull);
  });
  testWidgets('an armed pull can still be reversed and cancelled', (
    tester,
  ) async {
    final container = await setup();
    var calls = 0;
    await tester.pumpWidget(
      scene(
        container,
        refresh: () async {
          calls++;
        },
      ),
    );
    await tester.pumpAndSettle();
    final gesture = await tester.startGesture(const Offset(300, 180));
    for (var i = 0; i < 20; i++) {
      await gesture.moveBy(const Offset(0, 20));
      await tester.pump();
    }
    expect(graphic(tester).strandLength, greaterThan(90));
    for (var i = 0; i < 18; i++) {
      await gesture.moveBy(const Offset(0, -20));
      await tester.pump();
    }
    await gesture.up();
    await tester.pumpAndSettle();
    expect(calls, 0);
    await clean(tester, container);
  });

  testWidgets('pinching cancels an armed pull without abandoning a future', (
    tester,
  ) async {
    final container = await setup();
    var calls = 0;
    await tester.pumpWidget(
      scene(
        container,
        refresh: () async {
          calls++;
        },
      ),
    );
    await tester.pumpAndSettle();
    final first = await tester.startGesture(const Offset(300, 180), pointer: 1);
    await first.moveBy(const Offset(0, 400));
    await tester.pump();
    await tester.pump();
    final second = await tester.startGesture(
      const Offset(380, 300),
      pointer: 2,
    );
    await second.up();
    await first.up();
    await tester.pumpAndSettle();
    expect(calls, 0);
    await clean(tester, container);
  });

  testWidgets('switching accounts discards a pull before releasing it', (
    tester,
  ) async {
    final container = await setup();
    var calls = 0;
    await tester.pumpWidget(
      scene(
        container,
        refresh: () async {
          calls++;
        },
      ),
    );
    await tester.pumpAndSettle();
    final gesture = await tester.startGesture(const Offset(300, 180));
    await gesture.moveBy(const Offset(0, 400));
    await tester.pump();
    await tester.pump();
    await container
        .read(sessionRegistryProvider.notifier)
        .upsertAndActivate(
          token: 'local-test',
          did: 'did:plc:other',
          handle: 'other.craftsky.social',
        );
    await tester.pump();
    await gesture.up();
    await tester.pumpAndSettle();
    expect(calls, 0);
    expect(find.byKey(const Key('yarn-refresh-graphic')), findsNothing);
    await clean(tester, container);
  });
  testWidgets('empty lists refresh even when the Rive asset is unavailable', (
    tester,
  ) async {
    final container = await setup(broken: true);
    final pending = Completer<void>();
    var calls = 0;
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: YarnRefreshScrollView.list(
              onRefresh: () {
                calls++;
                return pending.future;
              },
              children: const [],
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.drag(find.byType(CustomScrollView), const Offset(0, 400));
    await tester.pump(const Duration(milliseconds: 500));
    expect(calls, 1);
    expect(find.byType(rive.RiveWidget), findsNothing);
    expect(find.byType(StitchProgressIndicator), findsOneWidget);
    pending.complete();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 250));
    await tester.pumpAndSettle();
    await clean(tester, container);
  });
  testWidgets('large pulls paint the strand all the way to the feed edge', (
    tester,
  ) async {
    final container = await setup();
    final pending = Completer<void>();
    await tester.pumpWidget(scene(container, refresh: () => pending.future));
    await tester.pumpAndSettle();
    final gesture = await tester.startGesture(const Offset(300, 180));
    await gesture.moveBy(const Offset(0, 400));
    await tester.pump();
    await tester.pump();
    final graphicBounds = tester.getRect(find.byType(rive.RiveWidget));
    expect(graphicBounds.height, greaterThan(240));
    final ink = Theme.of(
      tester.element(find.byType(rive.RiveWidget)),
    ).colorScheme.primary;
    final boundary = tester.renderObject<RenderRepaintBoundary>(
      find.byKey(const Key('yarn-capture')),
    );
    await tester.runAsync(() async {
      final image = await boundary.toImage();
      final bytes = (await image.toByteData())!;
      final x = graphicBounds.center.dx.round();
      final y = (graphicBounds.bottom - 4).floor();
      final offset = (y * image.width + x) * 4;
      expect(bytes.getUint8(offset), closeTo((ink.r * 255).round(), 3));
      expect(bytes.getUint8(offset + 1), closeTo((ink.g * 255).round(), 3));
      expect(bytes.getUint8(offset + 2), closeTo((ink.b * 255).round(), 3));
      if (const bool.fromEnvironment('CAPTURE_YARN_REFRESH')) {
        final png = (await image.toByteData(format: ui.ImageByteFormat.png))!;
        await File('/tmp/craftsky-yarn-refresh.png')
            .writeAsBytes(png.buffer.asUint8List());
      }
      image.dispose();
    });
    await gesture.up();
    await tester.pump();
    pending.complete();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 250));
    await tester.pumpAndSettle();
    await clean(tester, container);
  });
  testWidgets('retained-error refresh callbacks can omit the success check', (
    tester,
  ) async {
    final container = await setup();
    final pending = Completer<void>();
    await tester.pumpWidget(
      scene(container, showSuccess: false, refresh: () => pending.future),
    );
    await tester.pumpAndSettle();
    await tester.drag(find.byType(CustomScrollView), const Offset(0, 400));
    await tester.pump(const Duration(milliseconds: 500));
    pending.complete();
    await tester.pump();
    if (find.byKey(const Key('yarn-refresh-graphic')).evaluate().isNotEmpty) {
      expect(graphic(tester).showsSuccess, isFalse);
    }
    await tester.pumpAndSettle();
    await clean(tester, container);
  });
  testWidgets(
    'initial drag fades the yarn in and reversal fades it out without resizing',
    (tester) async {
      final container = await setup();
      var calls = 0;
      await tester.pumpWidget(
        scene(
          container,
          refresh: () async {
            calls++;
          },
        ),
      );
      await tester.pumpAndSettle();
      final gesture = await tester.startGesture(const Offset(300, 180));
      await gesture.moveBy(const Offset(0, 24));
      await tester.pump();
      final initial = graphic(tester).opacity;
      expect(initial, greaterThan(0));
      expect(initial, lessThan(1));
      expect(graphic(tester).ballScale, 1);
      await gesture.moveBy(const Offset(0, 45));
      await tester.pump();
      final stronger = graphic(tester).opacity;
      expect(stronger, greaterThan(initial));
      expect(graphic(tester).ballScale, 1);
      await gesture.moveBy(const Offset(0, -25));
      await tester.pump();
      expect(graphic(tester).opacity, lessThan(stronger));
      expect(graphic(tester).ballScale, 1);
      await gesture.up();
      await tester.pumpAndSettle();
      expect(calls, 0);
      await clean(tester, container);
    },
  );
}
