import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('activates locally across a hard state boundary', () async {
    var registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'token-alice',
          did: 'did:plc:alice',
          handle: 'alice.test',
        )
        .upsertAndActivate(
          token: 'token-bob',
          did: 'did:plc:bob',
          handle: 'bob.test',
        )
        .activate(
          AccountSessionLease(
            account: AccountKey('did:plc:alice'),
            sessionGeneration: 1,
          ),
        );
    final oldActive = registry.activeLease;
    final target = registry.leaseFor(AccountKey('did:plc:bob'))!;
    final operations = <String>[];
    final coordinator = AccountActivationCoordinator(
      readRegistry: () => registry,
      commitActivation: (lease) async {
        operations.add('commit');
        registry = registry.activate(lease);
      },
      invalidateAccountState: () async => operations.add('invalidate'),
      resetToHome: () async => operations.add('home'),
    );

    final result = await coordinator.activate(target);

    expect(result, AccountActivationResult.activated);
    expect(registry.activeDid, 'did:plc:bob');
    expect(registry.isCurrent(oldActive), isFalse);
    expect(operations, ['invalidate', 'commit', 'home']);
  });

  test('coalesces duplicate activation requests for the same lease', () async {
    var registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'token-alice',
          did: 'did:plc:alice',
          handle: 'alice.test',
        )
        .upsertAndActivate(
          token: 'token-bob',
          did: 'did:plc:bob',
          handle: 'bob.test',
        )
        .activate(
          AccountSessionLease(
            account: AccountKey('did:plc:alice'),
            sessionGeneration: 1,
          ),
        );
    final target = registry.leaseFor(AccountKey('did:plc:bob'))!;
    final releaseCommit = Completer<void>();
    var commits = 0;
    final coordinator = AccountActivationCoordinator(
      readRegistry: () => registry,
      commitActivation: (lease) async {
        commits++;
        await releaseCommit.future;
        registry = registry.activate(lease);
      },
      invalidateAccountState: () async {},
      resetToHome: () async {},
    );

    final first = coordinator.activate(target);
    final duplicate = coordinator.activate(target);
    await Future<void>.delayed(Duration.zero);
    expect(commits, 1);

    releaseCommit.complete();
    expect(await first, AccountActivationResult.activated);
    expect(await duplicate, AccountActivationResult.activated);
  });

  test(
    'UT-015 unsaved-work cancellation destroys the activation attempt',
    () async {
      var registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-bob',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'token-alice',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final target = registry.leaseFor(AccountKey('did:plc:bob'))!;
      var confirmations = 0;
      var commits = 0;
      final coordinator = AccountActivationCoordinator(
        readRegistry: () => registry,
        confirmLeave: (owner) async {
          expect(owner, registry.activeLease!.session);
          confirmations++;
          return false;
        },
        commitActivation: (lease) async {
          commits++;
          registry = registry.activate(lease);
        },
        invalidateAccountState: () async {},
        resetToHome: () async {},
      );

      expect(
        await coordinator.activate(target),
        AccountActivationResult.cancelled,
      );
      expect(confirmations, 1);
      expect(commits, 0);
      expect(registry.activeDid?.value, 'did:plc:alice');

      // A canceled request is not retained and cannot surprise-switch later.
      await Future<void>.delayed(Duration.zero);
      expect(commits, 0);
    },
  );

  test('UT-015 confirmation finishes before activation begins', () async {
    var registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'token-bob',
          did: 'did:plc:bob',
          handle: 'bob.test',
        )
        .upsertAndActivate(
          token: 'token-alice',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
    final target = registry.leaseFor(AccountKey('did:plc:bob'))!;
    final operations = <String>[];
    final coordinator = AccountActivationCoordinator(
      readRegistry: () => registry,
      confirmLeave: (_) async {
        operations.add('confirm-and-close');
        return true;
      },
      commitActivation: (lease) async {
        operations.add('commit');
        registry = registry.activate(lease);
      },
      invalidateAccountState: () async {},
      resetToHome: () async {},
    );

    expect(
      await coordinator.activate(target),
      AccountActivationResult.activated,
    );
    expect(operations, ['confirm-and-close', 'commit']);
  });
}
