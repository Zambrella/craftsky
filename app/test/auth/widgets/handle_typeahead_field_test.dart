import 'dart:async';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/auth/providers/handle_typeahead_provider.dart';
import 'package:craftsky_app/auth/services/handle_typeahead_service.dart';
import 'package:craftsky_app/auth/widgets/handle_typeahead_field.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/image_cache_fakes.dart';

class _Search extends HandleTypeaheadService {
  final queries = <String>[];
  final requests = <Completer<List<HandleSuggestion>>>[];
  final tokens = <CancelToken?>[];
  bool closed = false;

  @override
  void close() {
    closed = true;
    super.close();
  }

  @override
  Future<List<HandleSuggestion>> search(
    String query, {
    CancelToken? cancelToken,
  }) {
    queries.add(query);
    tokens.add(cancelToken);
    final request = Completer<List<HandleSuggestion>>();
    requests.add(request);
    return request.future;
  }
}

HandleSuggestion _actor(String handle, [String? name]) =>
    HandleSuggestion(handle: Handle.parse(handle), displayName: name);

void main() {
  late _Search search;
  late TextEditingController controller;
  late List<String> submissions;
  late FakeBaseCacheManager imageCache;

  setUp(() {
    search = _Search();
    controller = TextEditingController();
    submissions = [];
    imageCache = FakeBaseCacheManager();
  });

  tearDown(() {
    controller.dispose();
    search.close();
  });

  Future<void> mount(
    WidgetTester tester, {
    bool enabled = true,
    double topPadding = 0,
    bool centered = false,
  }) => tester.pumpWidget(
    ProviderScope(
      overrides: [
        handleTypeaheadServiceProvider.overrideWith((ref) {
          ref.onDispose(search.close);
          return search;
        }),
        profileImageCacheManagerProvider.overrideWith((ref) => imageCache),
      ],
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: LayoutBuilder(
            builder: (context, constraints) => SingleChildScrollView(
              child: ConstrainedBox(
                constraints: BoxConstraints(
                  minHeight: centered ? constraints.maxHeight : 0,
                ),
                child: Column(
                  mainAxisAlignment: centered
                      ? MainAxisAlignment.center
                      : MainAxisAlignment.start,
                  children: [
                    SizedBox(height: topPadding),
                    HandleTypeaheadField(
                      controller: controller,
                      enabled: enabled,
                      onSubmitted: submissions.add,
                    ),
                    TextButton(
                      onPressed: () {},
                      child: const Text('Continue'),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    ),
  );

  testWidgets('debounces, skips short queries, strips a leading @', (
    tester,
  ) async {
    await mount(tester);
    await tester.enterText(find.byType(TextField), 'a');
    await tester.pump(const Duration(milliseconds: 400));
    expect(search.queries, isEmpty);
    await tester.enterText(find.byType(TextField), 'al');
    await tester.pump(const Duration(milliseconds: 200));
    await tester.enterText(find.byType(TextField), ' @alice ');
    await tester.pump(const Duration(milliseconds: 299));
    expect(search.queries, isEmpty);
    await tester.pump(const Duration(milliseconds: 1));
    expect(search.queries, ['alice']);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('tap fills the handle without submitting or searching again', (
    tester,
  ) async {
    await mount(tester);
    await tester.enterText(find.byType(TextField), 'alice');
    await tester.pump(const Duration(milliseconds: 300));
    search.requests.single.complete([_actor('alice.test', 'Alice')]);
    await tester.pump();
    await tester.tap(find.text('@alice.test'));
    await tester.pump(const Duration(milliseconds: 400));
    expect(controller.text, 'alice.test');
    expect(find.byType(ListTile), findsNothing);
    expect(submissions, isEmpty);
    expect(search.queries, ['alice']);
    await tester.testTextInput.receiveAction(TextInputAction.done);
    expect(submissions, ['alice.test']);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('shows avatar beside suggestion with initial while loading', (
    tester,
  ) async {
    await mount(tester);
    await tester.enterText(find.byType(TextField), 'alice');
    await tester.pump(const Duration(milliseconds: 300));
    search.requests.single.complete([
      HandleSuggestion(
        handle: Handle.parse('alice.test'),
        displayName: 'Alice',
        avatarUrl: 'https://example.test/avatar.jpg',
      ),
      _actor('bob.test', 'Bob'),
    ]);
    await tester.pump();
    await tester.pump();
    final avatars = tester.widgetList<ProfileAvatar>(
      find.byType(ProfileAvatar),
    );
    expect(avatars.first.avatarUrl, 'https://example.test/avatar.jpg');
    expect(avatars.first.size, ProfileAvatarSize.small);
    expect(avatars.last.avatarUrl, isNull);
    expect(find.byType(CachedNetworkImage), findsOneWidget);
    expect(find.text('A'), findsOneWidget);
    expect(find.text('B'), findsOneWidget);
    await tester.tap(find.text('@alice.test'));
    await tester.pump();
    expect(controller.text, 'alice.test');
    expect(submissions, isEmpty);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('failed avatar preserves selectable suggestion', (tester) async {
    imageCache.nextStream = (_) => erroringStream();
    await mount(tester);
    await tester.enterText(find.byType(TextField), 'alice');
    await tester.pump(const Duration(milliseconds: 300));
    search.requests.single.complete([
      HandleSuggestion(
        handle: Handle.parse('alice.test'),
        displayName: 'Alice',
        avatarUrl: 'https://example.test/broken.jpg',
      ),
    ]);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    expect(find.text('A'), findsOneWidget);
    expect(find.text('@alice.test'), findsOneWidget);
    await tester.tap(find.text('@alice.test'));
    await tester.pump();
    expect(controller.text, 'alice.test');
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('clears stale options and ignores out-of-order responses', (
    tester,
  ) async {
    await mount(tester);
    await tester.enterText(find.byType(TextField), 'alice');
    await tester.pump(const Duration(milliseconds: 300));
    await tester.enterText(find.byType(TextField), 'bob');
    expect(search.tokens.first!.isCancelled, isTrue);
    await tester.pump(const Duration(milliseconds: 300));
    search.requests.last.complete([_actor('bob.test')]);
    await tester.pump();
    search.requests.first.complete([_actor('alice.test')]);
    await tester.pump();
    expect(find.text('@bob.test'), findsOneWidget);
    expect(find.text('@alice.test'), findsNothing);
    await tester.enterText(find.byType(TextField), 'b');
    await tester.pump();
    expect(find.byType(ListTile), findsNothing);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('failed search does not block manual submission', (tester) async {
    await mount(tester);
    await tester.enterText(find.byType(TextField), 'custom.example');
    await tester.pump(const Duration(milliseconds: 300));
    search.requests.single.completeError(Exception('offline'));
    await tester.pump();
    await tester.testTextInput.receiveAction(TextInputAction.done);
    expect(submissions, ['custom.example']);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('waits for IME composition to finish', (tester) async {
    await mount(tester);
    await tester.tap(find.byType(TextField));
    tester.testTextInput.updateEditingValue(
      const TextEditingValue(
        text: 'alice',
        composing: TextRange(start: 0, end: 5),
      ),
    );
    await tester.pump(const Duration(milliseconds: 400));
    expect(search.queries, isEmpty);
    tester.testTextInput.updateEditingValue(
      const TextEditingValue(text: 'alice'),
    );
    await tester.pump(const Duration(milliseconds: 300));
    expect(search.queries, ['alice']);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('keyboard navigation selects; Escape dismisses', (tester) async {
    await mount(tester);
    await tester.enterText(find.byType(TextField), 'alice');
    await tester.pump(const Duration(milliseconds: 300));
    search.requests.single.complete([
      _actor('alice.test'),
      _actor('alice.example'),
    ]);
    await tester.pump();
    await tester.sendKeyEvent(LogicalKeyboardKey.arrowDown);
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pump();
    expect(controller.text, 'alice.example');
    expect(submissions, isEmpty);
    await tester.enterText(find.byType(TextField), 'bob');
    await tester.pump(const Duration(milliseconds: 300));
    search.requests.last.complete([_actor('bob.test')]);
    await tester.pump();
    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pump();
    expect(find.byType(ListTile), findsNothing);
    expect(controller.text, 'bob');
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('disable and dispose cancel requests and discard results', (
    tester,
  ) async {
    await mount(tester);
    await tester.enterText(find.byType(TextField), 'alice');
    await tester.pump(const Duration(milliseconds: 300));
    await mount(tester, enabled: false);
    expect(search.tokens.single!.isCancelled, isTrue);
    search.requests.single.complete([_actor('alice.test')]);
    await tester.pump();
    expect(find.byType(ListTile), findsNothing);
    await mount(tester);
    await tester.enterText(find.byType(TextField), 'bob');
    await tester.pump(const Duration(milliseconds: 300));
    await tester.pumpWidget(const SizedBox());
    search.requests.last.complete([_actor('bob.test')]);
    await tester.pump();
    expect(tester.takeException(), isNull);
  });

  testWidgets('keeps auto-dispose service alive while a request is pending', (
    tester,
  ) async {
    await mount(tester);
    await tester.enterText(find.byType(TextField), 'alice');
    await tester.pump(const Duration(milliseconds: 300));
    await tester.pump(const Duration(seconds: 1));
    expect(search.closed, isFalse);
    search.requests.single.complete([_actor('alice.test')]);
    await tester.pump();
    await tester.pump();
    expect(find.text('@alice.test'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    await tester.pump();
    expect(search.closed, isTrue);
  });

  testWidgets('Tab dismisses suggestions and focuses the next form control', (
    tester,
  ) async {
    await mount(tester);
    await tester.enterText(find.byType(TextField), 'alice');
    await tester.pump(const Duration(milliseconds: 300));
    search.requests.single.complete([_actor('alice.test')]);
    await tester.pump();
    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pump();
    expect(find.byType(ListTile), findsNothing);
    expect(
      FocusManager.instance.primaryFocus!.context!
          .findAncestorWidgetOfExactType<TextButton>(),
      isNotNull,
    );
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('overlay leaves form layout unchanged and dismisses outside', (
    tester,
  ) async {
    await mount(tester);
    await tester.enterText(find.byType(TextField), 'alice');
    final fieldRect = tester.getRect(find.byType(BrandTextField));
    final buttonRect = tester.getRect(find.byType(TextButton));
    await tester.pump(const Duration(milliseconds: 300));
    search.requests.single.complete([_actor('alice.test')]);
    await tester.pump();
    expect(find.byType(CraftskyAnchoredSelectOverlay), findsOneWidget);
    expect(tester.getRect(find.byType(BrandTextField)), fieldRect);
    expect(tester.getRect(find.byType(TextButton)), buttonRect);
    await tester.tapAt(const Offset(10, 500));
    await tester.pump();
    expect(find.byType(CraftskyAnchoredSelectOverlay), findsNothing);
    expect(controller.text, 'alice');
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('overlay opens above the field to avoid the keyboard', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    tester.view.viewInsets = const FakeViewPadding(bottom: 330);
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.view.resetViewInsets);
    await mount(tester, topPadding: 300);
    await tester.enterText(find.byType(TextField), 'alice');
    await tester.pump(const Duration(milliseconds: 300));
    search.requests.single.complete([
      for (var i = 0; i < 5; i++) _actor('alice$i.test'),
    ]);
    await tester.pump();
    final menuRect = tester.getRect(find.byType(CraftskyOptionsPanel));
    final fieldRect = tester.getRect(find.byType(BrandTextField));
    expect(menuRect.bottom, lessThan(fieldRect.top));
    expect(menuRect.bottom, lessThanOrEqualTo(514));
    expect(menuRect.left, fieldRect.left);
    expect(menuRect.width, fieldRect.width);
    for (var i = 0; i < 4; i++) {
      await tester.sendKeyEvent(LogicalKeyboardKey.arrowDown);
      await tester.pumpAndSettle();
    }
    final rowRect = tester.getRect(find.byType(ListTile).last);
    expect(rowRect.top, greaterThanOrEqualTo(menuRect.top));
    expect(rowRect.bottom, lessThanOrEqualTo(menuRect.bottom));
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pump();
    expect(controller.text, 'alice4.test');
    expect(find.byType(CraftskyAnchoredSelectOverlay), findsNothing);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('overlay stays anchored when the form scrolls', (tester) async {
    await mount(tester, topPadding: 500);
    await tester.enterText(find.byType(TextField), 'alice');
    await tester.pump(const Duration(milliseconds: 300));
    search.requests.single.complete([_actor('alice.test')]);
    await tester.pump();
    final scrollable = tester.state<ScrollableState>(
      find.byType(Scrollable).first,
    );
    scrollable.position.jumpTo(30);
    await tester.pump();
    await tester.pump();
    final fieldRect = tester.getRect(find.byType(BrandTextField));
    final menuRect = tester.getRect(find.byType(CraftskyOptionsPanel));
    expect(menuRect.bottom, closeTo(fieldRect.top - 4, 0.01));
    expect(menuRect.left, fieldRect.left);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('open overlay repositions when the keyboard appears', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.view.resetViewInsets);
    await mount(tester, centered: true);
    await tester.enterText(find.byType(TextField), 'alice');
    await tester.pump(const Duration(milliseconds: 300));
    search.requests.single.complete([
      for (var i = 0; i < 5; i++) _actor('alice$i.test'),
    ]);
    await tester.pump();
    expect(
      tester.getRect(find.byType(CraftskyOptionsPanel)).top,
      greaterThan(tester.getRect(find.byType(BrandTextField)).bottom),
    );
    tester.view.viewInsets = const FakeViewPadding(bottom: 330);
    await tester.pumpAndSettle();
    final menuRect = tester.getRect(find.byType(CraftskyOptionsPanel));
    final fieldRect = tester.getRect(find.byType(BrandTextField));
    expect(
      menuRect.bottom == fieldRect.top - 4 ||
          menuRect.top == fieldRect.bottom + 4,
      isTrue,
    );
    expect(menuRect.bottom, lessThanOrEqualTo(514));
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('suggestions remain selectable on a narrow, short viewport', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 300);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await mount(tester);
    await tester.enterText(find.byType(TextField), 'alice');
    await tester.pump(const Duration(milliseconds: 300));
    search.requests.single.complete([
      for (var i = 0; i < 5; i++)
        _actor('alice$i.test', 'A long display name for Alice $i'),
    ]);
    await tester.pump();
    for (var i = 0; i < 4; i++) {
      await tester.sendKeyEvent(LogicalKeyboardKey.arrowDown);
      await tester.pumpAndSettle();
    }
    final lastRow = tester.getRect(find.byType(ListTile).last);
    expect(lastRow.top, greaterThanOrEqualTo(0));
    expect(lastRow.bottom, lessThanOrEqualTo(300));
    await tester.tap(find.text('@alice4.test'));
    await tester.pump();
    expect(controller.text, 'alice4.test');
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
  });
}
