import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/scheduled_posts/data/scheduled_post_repository.dart';
import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/pages/scheduled_posts_page.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_post_repository_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-006 management rows expose edit/delete and publishing lock', (
    tester,
  ) async {
    final longProjectText = List.filled(140, 'x').join();
    final items = [
      _item(
        'editable',
        ScheduledPostStatus.scheduled,
        mediaIds: const ['private-media-id'],
        kind: ScheduledPostKind.project,
        text: longProjectText,
        projectTitle: 'Cardigan',
      ),
      _item('locked', ScheduledPostStatus.publishing),
      _item(
        'attention',
        ScheduledPostStatus.needsAttention,
        needsAttentionExpiresAt: DateTime.utc(2026, 9),
      ),
    ];
    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: ScheduledPostsPageContent(
            items: items,
            onRefresh: () async {},
            onEdit: (_) async {},
            onDelete: (_) async {},
            thumbnailBuilder: (mediaId) => Text('thumbnail:$mediaId'),
          ),
        ),
      ),
    );

    expect(find.text('Scheduled posts'), findsOneWidget);
    expect(find.text('Scheduled'), findsOneWidget);
    expect(find.text('Publishing'), findsOneWidget);
    expect(find.byTooltip('Edit scheduled post'), findsNWidgets(2));
    expect(find.byTooltip('Delete scheduled post'), findsNWidgets(2));
    expect(find.text('thumbnail:private-media-id'), findsOneWidget);
    expect(find.text('Cardigan'), findsOneWidget);
    expect(find.text('${List.filled(119, 'x').join()}…'), findsOneWidget);
    expect(find.textContaining(RegExp(r'UTC[+-]\d{2}:\d{2}')), findsWidgets);
    expect(find.textContaining('Deleted on'), findsOneWidget);
    expect(
      find.text('Editing is unavailable while publishing'),
      findsOneWidget,
    );
  });

  testWidgets(
    'IT-021 delayed detail cannot open after the active account changes',
    (tester) async {
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'alice-token',
            did: 'did:plc:alice',
            handle: 'alice.test',
          )
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final aliceRegistry = registry.activate(registry.leaseFor(alice)!);
      final detail = Completer<ScheduledPostDetail>();
      final aliceRepository = _PageRepository(
        items: [_item('alice-private', ScheduledPostStatus.scheduled)],
        detail: detail,
      );
      final bobRepository = _PageRepository(items: const []);
      final observer = _PushObserver();
      final container = ProviderContainer(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(aliceRegistry),
          ),
          accountScheduledPostRepositoryProvider(
            alice,
          ).overrideWith((ref) async => aliceRepository),
          accountScheduledPostRepositoryProvider(
            bob,
          ).overrideWith((ref) async => bobRepository),
        ],
      );
      addTearDown(container.dispose);
      await container.read(sessionRegistryProvider.future);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            navigatorObservers: [observer],
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ScheduledPostsPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byTooltip('Edit scheduled post'));
      await tester.pump();
      await aliceRepository.getStarted.future;
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(aliceRegistry.leaseFor(bob)!);
      await tester.pump();
      detail.complete(
        ScheduledPostDetail(
          id: 'alice-private',
          operationId: 'alice-operation',
          status: ScheduledPostStatus.scheduled,
          scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 2, 12)),
          payload: const {
            'kind': 'standard',
            'text': 'Alice private scheduled detail',
          },
        ),
      );
      await tester.pumpAndSettle();

      expect(observer.pushes, 1);
      expect(find.byType(PostComposerSheet), findsNothing);
      expect(find.text('Alice private scheduled detail'), findsNothing);
    },
  );
}

final class _PageRepository implements ScheduledPostRepository {
  _PageRepository({required this.items, this.detail});

  final List<ScheduledPostSummary> items;
  final Completer<ScheduledPostDetail>? detail;
  final getStarted = Completer<void>();

  @override
  Future<List<ScheduledPostSummary>> list() async => items;

  @override
  Future<ScheduledPostDetail> get(String id) {
    if (!getStarted.isCompleted) getStarted.complete();
    return detail!.future;
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _PushObserver extends NavigatorObserver {
  int pushes = 0;

  @override
  void didPush(Route<dynamic> route, Route<dynamic>? previousRoute) {
    pushes++;
    super.didPush(route, previousRoute);
  }
}

ScheduledPostSummary _item(
  String id,
  ScheduledPostStatus status, {
  List<String> mediaIds = const [],
  DateTime? needsAttentionExpiresAt,
  ScheduledPostKind kind = ScheduledPostKind.standard,
  String? text,
  String? projectTitle,
}) => ScheduledPostSummary(
  id: id,
  kind: kind,
  status: status,
  text: text ?? '$id preview',
  projectTitle: projectTitle,
  scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 2, 12)),
  mediaIds: mediaIds,
  needsAttentionExpiresAt: needsAttentionExpiresAt,
);
