import 'package:craftsky_app/auth/models/pending_auth.dart' as model;
import 'package:craftsky_app/auth/providers/pending_auth_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  late ProviderContainer container;

  setUp(() => container = ProviderContainer.test());

  test('starts null', () {
    expect(container.read(pendingAuthProvider), isNull);
  });

  test('start records handle + current time', () {
    final before = DateTime.now();
    container.read(pendingAuthProvider.notifier).start('alice.bsky.social');
    final pending = container.read(pendingAuthProvider);

    expect(pending, isA<model.PendingAuth>());
    expect(pending!.handle, 'alice.bsky.social');
    expect(pending.startedAt.isBefore(before), isFalse);
  });

  test('clear resets to null', () {
    container.read(pendingAuthProvider.notifier).start('a.bsky.social');
    container.read(pendingAuthProvider.notifier).clear();
    expect(container.read(pendingAuthProvider), isNull);
  });

  test('start overwrites any prior pending auth', () {
    container.read(pendingAuthProvider.notifier).start('a.bsky.social');
    container.read(pendingAuthProvider.notifier).start('b.bsky.social');
    expect(container.read(pendingAuthProvider)!.handle, 'b.bsky.social');
  });

  test('debugSet directly replaces state (for aging in other tests)', () {
    final aged = model.PendingAuth(
      handle: 'x.bsky.social',
      startedAt: DateTime.now().subtract(const Duration(minutes: 15)),
    );
    container.read(pendingAuthProvider.notifier).debugSet(aged);

    expect(container.read(pendingAuthProvider)!.startedAt, aged.startedAt);
  });
}
