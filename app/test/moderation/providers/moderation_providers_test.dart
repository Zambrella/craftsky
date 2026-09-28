import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/moderation/data/moderation_repository.dart';
import 'package:craftsky_app/moderation/models/account_moderation.dart';
import 'package:craftsky_app/moderation/providers/moderation_providers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
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
  const _FakeRepository({required this.strikes});

  final int strikes;

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
  }) async => const ModerationHistoryPage(items: []);

  @override
  Future<ModerationHistoryPage> getHistoryEntry(
    ModerationCaseReference reference,
  ) async => const ModerationHistoryPage(items: []);
}
