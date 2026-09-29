import 'dart:async';

import 'package:craftsky_app/account_eligibility/data/account_eligibility_repository.dart';
import 'package:craftsky_app/account_eligibility/models/account_eligibility_status.dart';
import 'package:craftsky_app/account_eligibility/providers/account_eligibility_provider.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final class _Storage implements SessionRegistryStorage {
  _Storage(this.value);
  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _Repository implements AccountEligibilityRepository {
  _Repository(this.result);
  final Future<AccountEligibilityStatus> result;

  @override
  Future<AccountEligibilityStatus> readStatus() => result;
}

void main() {
  test('IT-018 discards a late eligibility response from account A', () async {
    var registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'token-a',
          did: 'did:plc:alice',
          handle: 'alice.test',
        )
        .upsertAndActivate(
          token: 'token-b',
          did: 'did:plc:bob',
          handle: 'bob.test',
        );
    final aliceSession = registry.leaseFor(AccountKey('did:plc:alice'))!;
    final bobSession = registry.leaseFor(AccountKey('did:plc:bob'))!;
    registry = registry.activate(aliceSession);
    final late = Completer<AccountEligibilityStatus>();
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        accountEligibilityRepositoryProvider.overrideWith(
          (ref, lease) async => _Repository(late.future),
        ),
      ],
    );
    final provider = accountEligibilityProvider(registry.activeLease!);
    final subscription = container.listen(provider, (_, _) {});
    addTearDown(subscription.close);

    await container.read(sessionRegistryProvider.future);
    await container.read(sessionRegistryProvider.notifier).activate(bobSession);
    late.complete(
      const AccountEligibilityStatus(
        state: AccountEligibilityState.restricted,
        appealable: true,
      ),
    );

    await expectLater(container.read(provider.future), throwsStateError);
  });
}
