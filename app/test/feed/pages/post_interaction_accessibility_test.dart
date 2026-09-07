import 'dart:async';
import 'dart:ui' show SemanticsAction;

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/pages/post_interaction_accounts_page.dart';
import 'package:craftsky_app/feed/pages/post_quotes_page.dart';
import 'package:craftsky_app/feed/providers/post_interaction_lists_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_interaction_summary.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/widgets/profile_account_list_tile.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/widgets/auto_paginated_list_view.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/auth_session_fakes.dart';
import '../fakes/fake_post_repository.dart';

const _did = 'did:plc:alice';
const _rkey = 'root';
const _constraints = <({Size size, double scale})>[
  (size: Size(390, 844), scale: 1),
  (size: Size(390, 844), scale: 2),
  (size: Size(1200, 800), scale: 1),
  (size: Size(1200, 800), scale: 2),
];

void main() {
  testWidgets(
    'AT-007 summary exposes exactly one actionable destination per count',
    (tester) async {
      final semantics = tester.ensureSemantics();
      await _pumpWidget(
        tester,
        PostInteractionSummary(
          post: _post(likeCount: 12, repostCount: 3, quoteCount: 2),
          onLikes: () {},
          onReposts: () {},
          onQuotes: () {},
        ),
      );

      for (final entry in [
        (label: '12 Likes', destination: 'Likes'),
        (label: '3 Reposts', destination: 'Reposts'),
        (label: '2 Quotes', destination: 'Quotes'),
      ]) {
        final finder = find.bySemanticsLabel(entry.label);
        expect(finder, findsOneWidget);
        final data = tester.getSemantics(finder).getSemanticsData();
        expect(data.label, entry.label);
        expect(data.hint, 'Open ${entry.destination} for this post');
        expect(data.flagsCollection.isButton, isTrue);
        expect(data.hasAction(SemanticsAction.tap), isTrue);
      }
      semantics.dispose();
    },
  );

  testWidgets(
    'AT-007 View likes is focusable, labelled, and first in a separate group',
    (tester) async {
      final semantics = tester.ensureSemantics();
      await _setConstraint(tester, _constraints.first);
      final root = PostRef(
        uri: 'at://did:plc:root/social.craftsky.feed.post/root',
        cid: 'bafyroot',
      );
      await _pumpWidget(
        tester,
        PostCard(
          post: _post(
            likeCount: 4,
            reply: PostReply(root: root, parent: root),
          ),
          onViewLikes: () {},
          onDelete: () {},
        ),
      );

      final menu = tester.widget<CraftskyContextMenuButton>(
        find.byType(CraftskyContextMenuButton),
      );
      expect(menu.groups, hasLength(2));
      expect(menu.groups.first.items, hasLength(1));
      expect(menu.groups.first.items.single.text, 'View likes');
      expect(
        menu.groups.first.items.single.style,
        CraftskyContextMenuItemStyle.normal,
      );

      await tester.tap(find.byTooltip('More actions'));
      await tester.pumpAndSettle();

      final action = find.bySemanticsLabel('View likes');
      expect(action, findsOneWidget);
      final data = tester.getSemantics(action).getSemanticsData();
      expect(data.label, 'View likes');
      expect(data.flagsCollection.isButton, isTrue);
      await tester.sendKeyEvent(LogicalKeyboardKey.tab);
      await tester.pump();
      expect(
        Focus.maybeOf(
          tester.element(find.text('View likes')),
          createDependency: false,
        )?.hasFocus,
        isTrue,
      );
      expect(
        find.descendant(
          of: find.byType(BottomSheet),
          matching: find.byType(CraftskyDivider),
        ),
        findsOneWidget,
      );
      expect(
        tester.getTopLeft(find.text('View likes')).dy,
        lessThan(tester.getTopLeft(find.text('Delete post')).dy),
      );
      semantics.dispose();
    },
  );

  testWidgets('AT-007 account row announces identity and profile action', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await _pumpWidget(
      tester,
      ProfileAccountListTile(account: _account()),
    );

    final row = find.byType(ProfileAccountListTile);
    final data = tester.getSemantics(row).getSemanticsData();
    expect(
      data.label,
      'A deliberately long display name that remains readable, '
      '@long-name.craftsky.social',
    );
    expect(data.hint, 'Visit profile');
    expect(data.flagsCollection.isButton, isTrue);
    expect(data.hasAction(SemanticsAction.tap), isTrue);
    semantics.dispose();
  });

  testWidgets(
    'AT-007 loading, error, empty, and retry states are understandable',
    (tester) async {
      final semantics = tester.ensureSemantics();
      final pending = Completer<ProfileAccountPage>();
      await _pumpAccountPage(
        tester,
        FakePostRepository(
          onListLikes: (did, rkey, {cursor, limit}) => pending.future,
        ),
      );
      await tester.pump();
      expect(find.bySemanticsLabel('Loading'), findsOneWidget);

      pending.completeError(const ApiNetworkError('offline'));
      await tester.pumpAndSettle();
      var status = tester
          .getSemantics(
            find.bySemanticsLabel("This didn't load. Please try again."),
          )
          .getSemanticsData();
      expect(status.flagsCollection.isLiveRegion, isTrue);
      final retry = tester
          .getSemantics(find.widgetWithText(TextButton, 'Retry'))
          .getSemanticsData();
      expect(retry.flagsCollection.isButton, isTrue);
      expect(retry.hasAction(SemanticsAction.tap), isTrue);

      await _pumpAccountPage(
        tester,
        FakePostRepository(
          onListLikes: (did, rkey, {cursor, limit}) async =>
              const ProfileAccountPage(items: [], totalCount: 0),
        ),
      );
      await tester.pumpAndSettle();
      status = tester
          .getSemantics(find.bySemanticsLabel('No likes yet.'))
          .getSemanticsData();
      expect(status.flagsCollection.isLiveRegion, isTrue);

      await _pumpQuotesPage(
        tester,
        FakePostRepository(
          onListQuotes: (did, rkey, {cursor, limit}) async =>
              throw const ApiNetworkError('offline'),
        ),
      );
      await tester.pumpAndSettle();
      status = tester
          .getSemantics(
            find.bySemanticsLabel("This didn't load. Please try again."),
          )
          .getSemanticsData();
      expect(status.flagsCollection.isLiveRegion, isTrue);
      expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);

      await _pumpQuotesPage(
        tester,
        FakePostRepository(
          onListQuotes: (did, rkey, {cursor, limit}) async =>
              const PostPage(items: []),
        ),
      );
      await tester.pumpAndSettle();
      status = tester
          .getSemantics(find.bySemanticsLabel('No quotes yet.'))
          .getSemanticsData();
      expect(status.flagsCollection.isLiveRegion, isTrue);
      semantics.dispose();
    },
  );

  for (final constraint in _constraints) {
    final label =
        '${constraint.size.width.toInt()}x${constraint.size.height.toInt()} '
        'at ${constraint.scale.toStringAsFixed(1)}';
    testWidgets(
      'AT-007 account full and pagination states do not overflow at $label',
      (tester) async {
        await _setConstraint(tester, constraint);
        await _pumpWidget(
          tester,
          PostInteractionSummary(
            post: _post(likeCount: 12345, repostCount: 6789, quoteCount: 2345),
            onLikes: () {},
            onReposts: () {},
            onQuotes: () {},
          ),
          scale: constraint.scale,
        );
        _expectNoFlutterError(tester);

        await _pumpAccountPage(
          tester,
          FakePostRepository(
            onListLikes: (did, rkey, {cursor, limit}) async =>
                ProfileAccountPage(items: [_account()], totalCount: 1),
          ),
          scale: constraint.scale,
        );
        await tester.pumpAndSettle();
        _expectNoFlutterError(tester);

        final continuation = Completer<ProfileAccountPage>();
        await _pumpAccountPage(
          tester,
          FakePostRepository(
            onListLikes: (did, rkey, {cursor, limit}) => cursor == null
                ? Future.value(
                    ProfileAccountPage(
                      items: [_account()],
                      totalCount: 2,
                      cursor: 'next',
                    ),
                  )
                : continuation.future,
          ),
          scale: constraint.scale,
        );
        await tester.pumpAndSettle();
        _notifyNearEnd(tester);
        await tester.pump();
        await _expectPaginationLoading(tester);
        _expectNoFlutterError(tester);

        continuation.completeError(const ApiNetworkError('offline'));
        await tester.pumpAndSettle();
        expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);
        _expectNoFlutterError(tester);
      },
    );

    testWidgets(
      'AT-007 quote full and pagination states do not overflow at $label',
      (tester) async {
        await _setConstraint(tester, constraint);
        await _pumpQuotesPage(
          tester,
          FakePostRepository(
            onListQuotes: (did, rkey, {cursor, limit}) async =>
                PostPage(items: [_post(longContent: true)]),
          ),
          scale: constraint.scale,
        );
        await tester.pumpAndSettle();
        _expectNoFlutterError(tester);

        final continuation = Completer<PostPage>();
        await _pumpQuotesPage(
          tester,
          FakePostRepository(
            onListQuotes: (did, rkey, {cursor, limit}) => cursor == null
                ? Future.value(
                    PostPage(
                      items: [_post(longContent: true)],
                      cursor: 'next',
                    ),
                  )
                : continuation.future,
          ),
          scale: constraint.scale,
        );
        await tester.pumpAndSettle();
        _notifyNearEnd(tester);
        await tester.pump();
        await _expectPaginationLoading(tester);
        _expectNoFlutterError(tester);

        continuation.completeError(const ApiNetworkError('offline'));
        await tester.pumpAndSettle();
        expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);
        _expectNoFlutterError(tester);
      },
    );
  }
}

Future<void> _setConstraint(
  WidgetTester tester,
  ({Size size, double scale}) constraint,
) async {
  await tester.binding.setSurfaceSize(constraint.size);
  addTearDown(() => tester.binding.setSurfaceSize(null));
}

Future<void> _pumpWidget(
  WidgetTester tester,
  Widget child, {
  double scale = 1,
}) => tester.pumpWidget(
  ProviderScope(
    overrides: List.from(_overrides(FakePostRepository())),
    child: _app(
      scale: scale,
      home: Scaffold(body: child),
    ),
  ),
);

Future<void> _pumpAccountPage(
  WidgetTester tester,
  FakePostRepository repository, {
  double scale = 1,
}) => tester.pumpWidget(
  ProviderScope(
    key: UniqueKey(),
    overrides: List.from(_overrides(repository)),
    retry: (_, _) => null,
    child: _app(
      scale: scale,
      home: PostInteractionAccountsPage(
        did: Did.parse(_did),
        rkey: RecordKey.parse(_rkey),
        kind: PostInteractionAccountKind.likes,
      ),
    ),
  ),
);

Future<void> _pumpQuotesPage(
  WidgetTester tester,
  FakePostRepository repository, {
  double scale = 1,
}) => tester.pumpWidget(
  ProviderScope(
    key: UniqueKey(),
    overrides: List.from(_overrides(repository)),
    retry: (_, _) => null,
    child: _app(
      scale: scale,
      home: PostQuotesPage(
        did: Did.parse(_did),
        rkey: RecordKey.parse(_rkey),
      ),
    ),
  ),
);

MaterialApp _app({required Widget home, double scale = 1}) => MaterialApp(
  theme: AppTheme.lightThemeData,
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  builder: (context, child) => MediaQuery(
    data: MediaQuery.of(
      context,
    ).copyWith(textScaler: TextScaler.linear(scale)),
    child: child!,
  ),
  home: home,
);

List<dynamic> _overrides(FakePostRepository repository) => [
  postRepositoryProvider.overrideWithValue(repository),
  authSessionProvider.overrideWith(
    () => SignedInAuthSession(did: 'did:plc:viewer'),
  ),
  activeLanguagePreferencesProvider.overrideWith(
    (ref) => const LanguagePreferences(
      primaryLanguage: 'en',
      contentLanguages: ['en'],
    ),
  ),
  secureSessionRegistryStorageProvider.overrideWithValue(
    _MemoryRegistryStorage(),
  ),
];

ProfileAccountSummary _account() => ProfileAccountSummary(
  did: 'did:plc:long-name',
  handle: 'long-name.craftsky.social',
  displayName: 'A deliberately long display name that remains readable',
  isCraftskyProfile: true,
);

Post _post({
  int likeCount = 0,
  int repostCount = 0,
  int quoteCount = 0,
  PostReply? reply,
  bool longContent = false,
}) => Post(
  uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
  cid: 'bafyroot',
  rkey: 'root',
  text: longContent
      ? 'A long quote post description that wraps over several lines without '
            'clipping, overlapping, or hiding any interaction controls.'
      : 'A post',
  tags: const [],
  createdAt: DateTime.utc(2026, 9, 6),
  indexedAt: DateTime.utc(2026, 9, 6),
  author: PostAuthor(
    did: 'did:plc:alice',
    handle: 'an-unusually-long-handle.craftsky.social',
    displayName: 'A deliberately long post author display name',
  ),
  likeCount: likeCount,
  repostCount: repostCount,
  quoteCount: quoteCount,
  replyCount: longContent ? 0 : 12345,
  viewerHasLiked: false,
  viewerHasReposted: false,
  viewerHasSaved: false,
  reply: reply,
);

void _notifyNearEnd(WidgetTester tester) {
  final context = tester.element(
    find.descendant(
      of: find.byType(AutoPaginatedListView),
      matching: find.byType(ListView),
    ),
  );
  ScrollUpdateNotification(
    metrics: FixedScrollMetrics(
      minScrollExtent: 0,
      maxScrollExtent: 100,
      pixels: 100,
      viewportDimension: 600,
      axisDirection: AxisDirection.down,
      devicePixelRatio: 1,
    ),
    context: context,
    scrollDelta: 1,
  ).dispatch(context);
}

void _expectNoFlutterError(WidgetTester tester) {
  expect(tester.takeException(), isNull);
}

Future<void> _expectPaginationLoading(WidgetTester tester) async {
  if (find.byType(StitchProgressIndicator).evaluate().isEmpty) {
    await tester.drag(find.byType(ListView), const Offset(0, -600));
    await tester.pump();
  }
  expect(find.byType(StitchProgressIndicator), findsOneWidget);
}

final class _MemoryRegistryStorage implements SessionRegistryStorage {
  _MemoryRegistryStorage()
    : registry = SessionRegistry.empty().upsertAndActivate(
        token: 'token-viewer',
        did: 'did:plc:viewer',
        handle: 'viewer.craftsky.social',
      );

  SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry value) async => registry = value;
}
