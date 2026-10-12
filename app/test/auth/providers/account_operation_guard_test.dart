import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final Provider<ActiveAccountOperation> _operationProvider = Provider(
  ActiveAccountOperation.capture,
);

void main() {
  test('V-01 acquisition cannot bind a replacement session', () async {
    final device = Completer<String>();
    final storage = _Storage(
      SessionRegistry.empty().upsertAndActivate(
        token: 'original-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      ),
    );
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(storage),
        deviceIdProvider.overrideWith((ref) => device.future),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final operation = container.read(_operationProvider);
    final client = accountDioProvider(AccountKey('did:plc:alice'));
    final subscription = container.listen(client, (_, _) {});
    addTearDown(subscription.close);
    final pending = operation.step(() => container.read(client.future));
    final rejected = expectLater(
      pending,
      throwsA(isA<AccountOperationDiscarded>()),
    );

    await container
        .read(sessionRegistryProvider.notifier)
        .upsertAndActivate(
          token: 'replacement-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
    device.complete('installation-id');

    await rejected;
    expect(operation.isCurrent, isFalse);
  });

  test('V-02 switch back cannot revive the original operation', () async {
    final initial = SessionRegistry.empty()
        .upsertAndActivate(token: 'bob', did: 'did:plc:bob', handle: 'bob.test')
        .upsertAndActivate(
          token: 'alice',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(initial),
        ),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final operation = container.read(_operationProvider);
    final registry = container.read(sessionRegistryProvider.notifier);
    await registry.activate(initial.leaseFor(AccountKey('did:plc:bob'))!);
    await registry.activate(initial.leaseFor(AccountKey('did:plc:alice'))!);
    var dispatched = false;
    var published = false;
    await expectLater(
      operation.step(() async => dispatched = true),
      throwsA(isA<AccountOperationDiscarded>()),
    );
    expect(operation.publish(() => published = true), isFalse);
    expect(dispatched, isFalse);
    expect(published, isFalse);
  });

  test('V-03 a switch after a guarded step still blocks publication', () async {
    var current = true;
    final operation = ActiveAccountOperation.withGuard(
      ownership: null,
      isCurrent: () => current,
    );
    final result = await operation.step(() async => 'completed').then((value) {
      current = false;
      return value;
    });
    var published = false;
    expect(result, 'completed');
    expect(operation.publish(() => published = true), isFalse);
    expect(published, isFalse);
  });

  test('V-03 disposed owners cannot publish even without a session', () async {
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(SessionRegistry.empty()),
        ),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final operation = container.read(_operationProvider);
    expect(operation.ownership, isNull);
    var published = 0;
    expect(operation.publish(() => published++), isTrue);
    container
      ..invalidate(_operationProvider)
      ..read(_operationProvider);
    expect(operation.publish(() => published++), isFalse);
    expect(published, 1);
  });

  test(
    'V-06 a late step failure retains its original cause and stack',
    () async {
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(SessionRegistry.empty()),
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final operation = container.read(_operationProvider);
      final failed = Completer<void>();
      final cause = StateError('fixture failure');
      final stack = StackTrace.fromString('initiating dependency frame');
      final pending = operation.step(() => failed.future);
      final inspected = pending.then<void>(
        (_) => fail('expected failure'),
        onError: (Object error, StackTrace actualStack) {
          expect(error, same(cause));
          expect(actualStack.toString(), stack.toString());
        },
      );
      container
        ..invalidate(_operationProvider)
        ..read(_operationProvider);
      failed.completeError(cause, stack);
      await inspected;
    },
  );
}

final class _Storage implements SessionRegistryStorage {
  _Storage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}
