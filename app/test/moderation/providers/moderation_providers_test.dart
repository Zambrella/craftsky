import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/moderation/data/moderation_repository.dart';
import 'package:craftsky_app/moderation/models/account_moderation.dart';
import 'package:craftsky_app/moderation/providers/moderation_providers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('history refresh discards pagination from its previous build', () async {
    final account = AccountKey('did:plc:first');
    final page = Completer<ModerationHistoryPage>();
    final paginationStarted = Completer<void>();
    var initialLoads = 0;
    final repository = _FakeRepository(
      strikes: 0,
      onHistory: (cursor) async {
        if (cursor != null) {
          paginationStarted.complete();
          return await page.future;
        }
        initialLoads++;
        return ModerationHistoryPage(
          items: const [],
          cursor: initialLoads == 1 ? 'original' : 'fresh',
        );
      },
    );
    final container = ProviderContainer.test(
      overrides: [
        accountModerationRepositoryProvider(account).overrideWith(
          (_) async => repository,
        ),
      ],
    );
    final provider = accountModerationHistoryProvider(account);
    container.listen(provider, (_, _) {});
    await container.read(provider.future);
    final notifier = container.read(provider.notifier);
    final pending = notifier.loadMore();
    await paginationStarted.future;
    await notifier.refresh();
    await container.read(provider.future);
    expect(container.read(provider).requireValue.cursor, 'fresh');
    page.complete(const ModerationHistoryPage(items: [], cursor: 'stale'));
    await pending;
    expect(container.read(provider).requireValue.cursor, 'fresh');
  });

  test('standing and history providers are isolated by account key', () async {
    final first = AccountKey('did:plc:first');
    final second = AccountKey('did:plc:second');
    final container = ProviderContainer(
      overrides: [
        accountModerationRepositoryProvider(
          first,
        ).overrideWith((_) async => const _FakeRepository(strikes: 1)),
        accountModerationRepositoryProvider(
          second,
        ).overrideWith((_) async => const _FakeRepository(strikes: 2)),
      ],
    );
    addTearDown(container.dispose);

    expect(
      (await container.read(
        accountStandingProvider(first).future,
      )).activeStrikeCount,
      1,
    );
    expect(
      (await container.read(
        accountStandingProvider(second).future,
      )).activeStrikeCount,
      2,
    );
  });

  test('appeal URI contains only public appeal data', () {
    final uri = moderationAppealUri(
      ModerationCaseReference.parse(
        'MOD-550e8400-e29b-41d4-a716-446655440000',
      ),
    );

    expect(uri.scheme, 'mailto');
    expect(uri.path, moderationAppealAddress);
    expect(uri.queryParameters['subject'], contains('MOD-550e8400'));
    expect(uri.queryParameters.keys, unorderedEquals(['subject', 'body']));
  });
}

final class _FakeRepository implements ModerationRepository {
  const new({required this.strikes, this.onHistory});

  final int strikes;
  final Future<ModerationHistoryPage> Function(String? cursor)? onHistory;

  @override
  Future<AccountStanding> getStanding() async => AccountStanding(
    activeStrikeCount: strikes,
    strikeThreshold: 3,
    thresholdSuspended: false,
    severeSuspended: false,
    suspended: false,
  );

  @override
  Future<ModerationHistoryPage> getHistory({
    String? cursor,
    int? limit,
  }) async =>
      await onHistory?.call(cursor) ?? const ModerationHistoryPage(items: []);

  @override
  Future<ModerationHistoryPage> getHistoryEntry(
    ModerationCaseReference reference,
  ) async => const ModerationHistoryPage(items: []);
}
