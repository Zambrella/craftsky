import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/profile/models/follower_growth.dart';
import 'package:craftsky_app/profile/providers/follower_growth_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_profile_repository.dart';

void main() {
  test('late growth response remains isolated to its account key', () async {
    final alice = AccountKey('did:plc:alice');
    final bob = AccountKey('did:plc:bob');
    final aliceResult = Completer<FollowerGrowth>();
    final bobResult = Completer<FollowerGrowth>();
    final repositories = {
      alice: FakeProfileRepository(
        onFetchFollowerGrowth: (_) => aliceResult.future,
      ),
      bob: FakeProfileRepository(
        onFetchFollowerGrowth: (_) => bobResult.future,
      ),
    };
    final container = ProviderContainer.test(
      overrides: [
        accountFollowerGrowthRepositoryProvider.overrideWith(
          (ref, account) async => repositories[account]!,
        ),
      ],
    );
    final aliceSubscription = container.listen(
      followerGrowthProvider(alice, FollowerGrowthPeriod.thirtyDays),
      (_, _) {},
    );
    final bobSubscription = container.listen(
      followerGrowthProvider(bob, FollowerGrowthPeriod.thirtyDays),
      (_, _) {},
    );
    addTearDown(aliceSubscription.close);
    addTearDown(bobSubscription.close);

    final aliceFuture = container.read(
      followerGrowthProvider(alice, FollowerGrowthPeriod.thirtyDays).future,
    );
    final bobFuture = container.read(
      followerGrowthProvider(bob, FollowerGrowthPeriod.thirtyDays).future,
    );
    bobResult.complete(_growth(7));
    await expectLater(bobFuture, completion(hasFollowerCount(7)));

    aliceResult.complete(_growth(41));
    await expectLater(aliceFuture, completion(hasFollowerCount(41)));

    expect(
      container
          .read(
            followerGrowthProvider(bob, FollowerGrowthPeriod.thirtyDays),
          )
          .requireValue
          .latestFollowerCount,
      7,
    );
  });

  test('account invalidation clears retained growth state', () async {
    final account = AccountKey('did:plc:alice');
    var calls = 0;
    final repository = FakeProfileRepository(
      onFetchFollowerGrowth: (_) async => _growth(++calls),
    );
    final container = ProviderContainer.test(
      overrides: [
        accountFollowerGrowthRepositoryProvider.overrideWith(
          (ref, account) async => repository,
        ),
      ],
    );
    final subscription = container.listen(
      followerGrowthProvider(account, FollowerGrowthPeriod.sevenDays),
      (_, _) {},
    );
    addTearDown(subscription.close);

    expect(
      (await container.read(
        followerGrowthProvider(
          account,
          FollowerGrowthPeriod.sevenDays,
        ).future,
      )).latestFollowerCount,
      1,
    );

    await container.read(accountStateInvalidatorProvider)();

    expect(
      (await container.read(
        followerGrowthProvider(
          account,
          FollowerGrowthPeriod.sevenDays,
        ).future,
      )).latestFollowerCount,
      2,
    );
  });
}

Matcher hasFollowerCount(int count) => isA<FollowerGrowth>().having(
  (growth) => growth.latestFollowerCount,
  'latestFollowerCount',
  count,
);

FollowerGrowth _growth(int count) => FollowerGrowth(
  period: FollowerGrowthPeriod.thirtyDays,
  rangeStart: DateTime.utc(2026, 8),
  rangeEnd: DateTime.utc(2026, 8),
  availableFrom: DateTime.utc(2026, 8),
  latestSnapshotDate: DateTime.utc(2026, 8),
  latestCapturedAt: DateTime.utc(2026, 8),
  latestFollowerCount: count,
  netChange: null,
  points: [
    FollowerGrowthPoint(date: DateTime.utc(2026, 8), count: count),
  ],
);
