import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/account_type_controller.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-011 serializes updates and accepts the exact returned type',
    () async {
      final repository = _RecordingBusinessRepository();
      final reconciliation = <AccountType>[];
      final invalidations = <AccountType>[];
      final container = _container(
        repository: repository,
        reconcile: reconciliation.add,
        invalidate: invalidations.add,
      );
      addTearDown(container.dispose);
      final subscription = container.listen(
        accountTypeControllerProvider,
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(subscription.close);
      await container.read(accountTypeControllerProvider.future);

      final first = container
          .read(accountTypeControllerProvider.notifier)
          .setAccountType(AccountType.business);
      final duplicate = await container
          .read(accountTypeControllerProvider.notifier)
          .setAccountType(AccountType.business);

      expect(duplicate, isFalse);
      expect(repository.accountTypeUpdates, [AccountType.business]);
      expect(container.read(accountTypeControllerProvider).isLoading, isTrue);

      repository.complete(AccountType.regular);
      expect(await first, isTrue);
      expect(
        container.read(accountTypeControllerProvider).requireValue,
        AccountType.regular,
      );
      expect(reconciliation, [AccountType.regular]);
      expect(invalidations, [AccountType.regular]);
    },
  );

  test(
    'UT-011 failure retains the confirmed type and re-enables updates',
    () async {
      final repository = _RecordingBusinessRepository();
      final container = _container(repository: repository);
      addTearDown(container.dispose);
      final subscription = container.listen(
        accountTypeControllerProvider,
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(subscription.close);
      await container.read(accountTypeControllerProvider.future);

      final failed = container
          .read(accountTypeControllerProvider.notifier)
          .setAccountType(AccountType.business);
      repository.fail(StateError('request failed'));

      expect(await failed, isFalse);
      final failedState = container.read(accountTypeControllerProvider);
      expect(failedState.hasError, isFalse);
      expect(failedState.value, AccountType.regular);

      final retry = container
          .read(accountTypeControllerProvider.notifier)
          .setAccountType(AccountType.business);
      expect(repository.accountTypeUpdates, [
        AccountType.business,
        AccountType.business,
      ]);
      repository.complete(AccountType.business);
      expect(await retry, isTrue);
    },
  );

  test(
    'IT-004 regular reconciliation hides cached business projection',
    () async {
      final profile = Profile(
        did: 'did:plc:alice',
        handle: 'alice.test',
        crafts: const [],
        accountType: AccountType.business,
        business: BusinessProfile(
          cid: 'bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq',
        ),
      );
      final container = ProviderContainer(
        overrides: [
          activeAccountIdentityProvider.overrideWith(
            (_) async => ActiveAccountIdentity(
              lease: AccountSessionLease(
                account: AccountKey('did:plc:alice'),
                sessionGeneration: 1,
              ),
              profile: profile,
            ),
          ),
          profileRepositoryProvider.overrideWithValue(
            _ProfileRepository(profile),
          ),
        ],
      );
      addTearDown(container.dispose);
      final subscription = container.listen(
        userProfileProvider('alice.test'),
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(subscription.close);
      await container.read(userProfileProvider('alice.test').future);
      await container.read(activeAccountIdentityProvider.future);

      container.read(accountTypeProfileReconcilerProvider)(AccountType.regular);

      final reconciled = container
          .read(userProfileProvider('alice.test'))
          .requireValue;
      expect(reconciled.accountType, AccountType.regular);
      expect(reconciled.business, isNull);
    },
  );
}

ProviderContainer _container({
  required _RecordingBusinessRepository repository,
  void Function(AccountType)? reconcile,
  void Function(AccountType)? invalidate,
}) => ProviderContainer(
  overrides: [
    activeAccountIdentityProvider.overrideWith(
      (_) async => ActiveAccountIdentity(
        lease: AccountSessionLease(
          account: AccountKey('did:plc:alice'),
          sessionGeneration: 1,
        ),
        profile: Profile(
          did: 'did:plc:alice',
          handle: 'alice.test',
          crafts: const [],
          accountType: AccountType.regular,
        ),
      ),
    ),
    businessRepositoryProvider.overrideWithValue(repository),
    if (reconcile != null)
      accountTypeProfileReconcilerProvider.overrideWithValue(reconcile),
    if (invalidate != null)
      accountTypeStateInvalidatorProvider.overrideWithValue(invalidate),
  ],
);

final class _RecordingBusinessRepository extends Fake
    implements BusinessRepository {
  final accountTypeUpdates = <AccountType>[];
  late Completer<AccountType> _completion;

  @override
  Future<AccountType> updateAccountType(AccountType value) {
    accountTypeUpdates.add(value);
    _completion = Completer<AccountType>();
    return _completion.future;
  }

  void complete(AccountType value) => _completion.complete(value);

  void fail(Object error) => _completion.completeError(error);
}

final class _ProfileRepository extends Fake implements ProfileRepository {
  _ProfileRepository(this.profile);

  final Profile profile;

  @override
  Future<Profile> fetch(String handleOrDid) async => profile;
}
