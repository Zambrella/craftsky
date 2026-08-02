import 'dart:typed_data';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/scheduled_posts/data/scheduled_post_repository.dart';
import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/pages/scheduled_posts_page.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_post_repository_provider.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/recording_messenger.dart';

void main() {
  testWidgets('AT-008 deletes unpublished items and locks Publishing rows', (
    tester,
  ) async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final account = registry.activeLease!.session.account;
    final repository = _DeleteRepository([
      _summary('editable', ScheduledPostStatus.scheduled),
      _summary('locked', ScheduledPostStatus.publishing),
    ]);
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(registry),
        ),
        accountScheduledPostRepositoryProvider(
          account,
        ).overrideWith((ref) async => repository),
      ],
    );
    addTearDown(container.dispose);
    await container.read(sessionRegistryProvider.future);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ScheduledPostsPage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('editable preview'), findsOneWidget);
    expect(find.text('locked preview'), findsOneWidget);
    expect(find.byTooltip('Delete scheduled post'), findsOneWidget);
    expect(find.byTooltip('Edit scheduled post'), findsOneWidget);
    expect(
      find.text('Editing is unavailable while publishing'),
      findsOneWidget,
    );

    await tester.tap(find.byTooltip('Delete scheduled post'));
    await tester.pumpAndSettle();
    expect(find.text('Delete scheduled post?'), findsOneWidget);
    await tester.tap(find.widgetWithText(ChunkyButton, 'Delete'));
    await tester.pumpAndSettle();

    expect(repository.deletedIDs, ['editable']);
    expect(find.text('editable preview'), findsNothing);
    expect(find.text('locked preview'), findsOneWidget);
    expect(find.byTooltip('Delete scheduled post'), findsNothing);

    final listCallsBeforeRefresh = repository.listCalls;
    repository.items[0] = _summary(
      'locked',
      ScheduledPostStatus.needsAttention,
    );
    await tester.drag(find.byType(ListView), const Offset(0, 400));
    await tester.pumpAndSettle();

    expect(repository.listCalls, greaterThan(listCallsBeforeRefresh));
    expect(find.text('Needs attention'), findsOneWidget);
    expect(find.byTooltip('Delete scheduled post'), findsOneWidget);
    expect(find.byTooltip('Edit scheduled post'), findsOneWidget);
  });
}

ScheduledPostSummary _summary(String id, ScheduledPostStatus status) =>
    ScheduledPostSummary(
      id: id,
      kind: ScheduledPostKind.standard,
      status: status,
      text: '$id preview',
      scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 2, 12)),
    );

final class _DeleteRepository implements ScheduledPostRepository {
  _DeleteRepository(this.items);

  final List<ScheduledPostSummary> items;
  final deletedIDs = <String>[];
  int listCalls = 0;

  @override
  Future<List<ScheduledPostSummary>> list() async {
    listCalls += 1;
    return List.of(items);
  }

  @override
  Future<void> delete(String id) async {
    deletedIDs.add(id);
    items.removeWhere((item) => item.id == id);
  }

  @override
  Future<ScheduledPostDetail> create({
    required String operationId,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) => throw UnimplementedError();

  @override
  Future<ScheduledPostDetail> get(String id) => throw UnimplementedError();

  @override
  Future<Uint8List> mediaBytes(String id) => throw UnimplementedError();

  @override
  Future<void> publishNow({
    required String id,
    required Map<String, dynamic> payload,
  }) => throw UnimplementedError();

  @override
  Future<void> stageMedia({
    required String id,
    required List<int> bytes,
    required String mimeType,
  }) => throw UnimplementedError();

  @override
  Future<ScheduledPostDetail> update({
    required String id,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) => throw UnimplementedError();
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.registry);

  SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async {
    this.registry = registry;
  }
}
