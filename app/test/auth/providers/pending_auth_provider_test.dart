import 'package:craftsky_app/auth/models/pending_auth.dart' as model;
import 'package:craftsky_app/auth/providers/pending_auth_provider.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  late ProviderContainer container;

  setUp(() => container = ProviderContainer.test());

  test('starts null', () {
    expect(container.read(pendingAuthProvider), isNull);
  });

  test('startSignIn records handle + current time', () {
    final before = DateTime.now();
    container
        .read(pendingAuthProvider.notifier)
        .startSignIn('alice.bsky.social');
    final pending = container.read(pendingAuthProvider);

    expect(pending, isA<model.PendingAuth>());
    expect(pending!.purpose, model.PendingAuthPurpose.signIn);
    expect(pending.handle, 'alice.bsky.social');
    expect(pending.startedAt.isBefore(before), isFalse);
  });

  test('clear resets to null', () {
    container.read(pendingAuthProvider.notifier).startSignIn('a.bsky.social');
    container.read(pendingAuthProvider.notifier).clear();
    expect(container.read(pendingAuthProvider), isNull);
  });

  test('UT-009 pause/resume stays retryable and a second start is allowed', () {
    final notifier = container.read(pendingAuthProvider.notifier)
      ..startRegistration();
    final first = container.read(pendingAuthProvider);

    TestWidgetsFlutterBinding.ensureInitialized()
      ..handleAppLifecycleStateChanged(AppLifecycleState.paused)
      ..handleAppLifecycleStateChanged(AppLifecycleState.resumed);

    expect(container.read(pendingAuthProvider), same(first));
    expect(first!.purpose, model.PendingAuthPurpose.registration);
    expect(first.handle, isNull);

    notifier.startRegistration();
    final second = container.read(pendingAuthProvider);
    expect(second, isNot(same(first)));
    expect(second!.purpose, model.PendingAuthPurpose.registration);
    expect(second.handle, isNull);
  });

  test('debugSet directly replaces state (for aging in other tests)', () {
    final aged = model.PendingAuth.signIn(
      handle: 'x.bsky.social',
      startedAt: DateTime.now().subtract(const Duration(minutes: 15)),
    );
    container.read(pendingAuthProvider.notifier).debugSet(aged);

    expect(container.read(pendingAuthProvider)!.startedAt, aged.startedAt);
  });
}
