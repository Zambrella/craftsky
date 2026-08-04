import 'dart:typed_data';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/scheduled_posts/data/scheduled_post_repository.dart';
import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_post_repository_provider.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_posts_provider.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/recording_messenger.dart';

void main() {
  testWidgets('AT-005 explains full capacity and unlocks after refresh', (
    tester,
  ) async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final account = registry.activeLease!.session.account;
    final repository = _CapacityRepository([
      _summary('one'),
      _summary('two', status: ScheduledPostStatus.retrying),
      _summary('three', status: ScheduledPostStatus.needsAttention),
    ]);
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(registry),
        ),
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
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
            home: const PostComposerSheet(composerId: 'capacity'),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField).first, 'A valid post');
    await tester.pumpAndSettle();

    const capacityWarning =
        "You can't schedule another post because you already have "
        '3 scheduled posts.';
    expect(
      find.text(capacityWarning),
      findsOneWidget,
    );
    expect(find.text('Manage scheduled posts'), findsOneWidget);
    expect(_button(tester, 'Post').onPressed, isNotNull);

    await tester.tap(find.text('When'));
    await tester.pumpAndSettle();
    final laterTile = find.ancestor(
      of: find.text('Schedule for later'),
      matching: find.byType(ListTile),
    );
    expect(tester.widget<ListTile>(laterTile).enabled, isFalse);
    await tester.tap(find.text('Now').last);
    await tester.pumpAndSettle();

    repository.items.removeLast();
    await container.read(scheduledPostsProvider(account).notifier).refresh();
    await tester.pumpAndSettle();

    expect(find.textContaining('of 3 scheduled'), findsNothing);
    expect(find.textContaining("can't schedule another post"), findsNothing);
    expect(find.text('Manage scheduled posts'), findsNothing);

    await _selectLater(tester);
    expect(_button(tester, 'Schedule').onPressed, isNotNull);
  });
}

ChunkyButton _button(WidgetTester tester, String label) =>
    tester.widget<ChunkyButton>(find.widgetWithText(ChunkyButton, label));

Future<void> _selectLater(WidgetTester tester) async {
  await tester.tap(find.text('When'));
  await tester.pumpAndSettle();
  await tester.tap(find.text('Schedule for later'));
  await tester.pumpAndSettle();
  await tester.tap(find.text('OK'));
  await tester.pumpAndSettle();
  await tester.tap(find.text('OK'));
  await tester.pumpAndSettle();
}

ScheduledPostSummary _summary(
  String id, {
  ScheduledPostStatus status = ScheduledPostStatus.scheduled,
}) => ScheduledPostSummary(
  id: id,
  kind: ScheduledPostKind.standard,
  status: status,
  text: id,
  scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 2, 12)),
);

final class _CapacityRepository implements ScheduledPostRepository {
  _CapacityRepository(this.items);

  final List<ScheduledPostSummary> items;

  @override
  Future<List<ScheduledPostSummary>> list() async => List.of(items);

  @override
  Future<ScheduledPostDetail> create({
    required String operationId,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) => throw UnimplementedError();

  @override
  Future<void> delete(String id) => throw UnimplementedError();

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
    CancelToken? cancelToken,
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
