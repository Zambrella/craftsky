import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_flow_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../profile/fakes/fake_profile_repository.dart';

final class _Storage implements SessionRegistryStorage {
  _Storage(this.registry);
  SessionRegistry registry;
  @override
  Future<SessionRegistry> read() async => registry;
  @override
  Future<void> write(SessionRegistry registry) async =>
      this.registry = registry;
}

void main() {
  test(
    'shows an editable onboarding draft before profile prefill returns',
    () async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final profile = Completer<Profile>();
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          accountProfileRepositoryProvider.overrideWith(
            (ref, lease) async => FakeProfileRepository(
              onFetchMe: () => profile.future,
            ),
          ),
        ],
      );

      final provider = onboardingFlowProvider(registry.activeLease!);
      final subscription = container.listen(provider, (_, _) {});
      addTearDown(subscription.close);
      final state = await container
          .read(provider.future)
          .timeout(
            const Duration(milliseconds: 100),
          );

      expect(state.baseline.handle.value, 'alice.test');
      expect(state.identity.displayName, isEmpty);
      expect(state.identity.bio, isEmpty);
    },
  );

  test('UT-003 elapsed prefill bound includes an in-flight request', () async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    var reads = 0;
    final repository = FakeProfileRepository(
      onFetchMe: () {
        reads++;
        if (reads > 1) return Completer<Profile>().future;
        return Future.value(
          Profile(
            did: 'did:plc:alice',
            handle: 'alice.test',
            crafts: const [],
          ),
        );
      },
    );
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        accountProfileRepositoryProvider.overrideWith(
          (ref, lease) async => repository,
        ),
        onboardingPrefillRetryDelaysProvider.overrideWithValue(
          const [Duration.zero],
        ),
        onboardingPrefillDeadlineProvider.overrideWithValue(
          const Duration(milliseconds: 20),
        ),
      ],
    );

    final provider = onboardingFlowProvider(registry.activeLease!);
    final subscription = container.listen(provider, (_, _) {});
    addTearDown(subscription.close);
    final stopwatch = Stopwatch()..start();
    await container.read(provider.future);
    await Future<void>.delayed(const Duration(milliseconds: 100));
    final state = container.read(provider).requireValue;

    expect(stopwatch.elapsed, lessThan(const Duration(seconds: 1)));
    expect(state.identity.displayName, isEmpty);
    expect(reads, 2);
  });

  test('retries empty identity once per flow until populated', () async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    var reads = 0;
    final repository = FakeProfileRepository(
      onFetchMe: () async {
        reads++;
        return Profile(
          did: 'did:plc:alice',
          handle: 'alice.test',
          crafts: const [],
          displayName: reads == 1 ? null : 'Alice',
        );
      },
    );
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        accountProfileRepositoryProvider.overrideWith(
          (ref, lease) async => repository,
        ),
        onboardingPrefillRetryDelaysProvider.overrideWithValue(
          const [Duration.zero],
        ),
      ],
    );

    final provider = onboardingFlowProvider(registry.activeLease!);
    final populated = Completer<OnboardingFlowState>();
    final subscription = container.listen(provider, (_, next) {
      final value = next.value;
      if (value?.identity.displayName == 'Alice' && !populated.isCompleted) {
        populated.complete(value);
      }
    });
    addTearDown(subscription.close);
    await container.read(provider.future);
    final state = await populated.future.timeout(const Duration(seconds: 1));
    expect(state.identity.displayName, 'Alice');
    expect(reads, 2);
    container
        .read(onboardingFlowProvider(registry.activeLease!).notifier)
        .next();
    container
        .read(onboardingFlowProvider(registry.activeLease!).notifier)
        .previous();
    expect(reads, 2);
  });

  test('background prefill does not overwrite an onboarding edit', () async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final profile = Completer<Profile>();
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        accountProfileRepositoryProvider.overrideWith(
          (ref, lease) async => FakeProfileRepository(
            onFetchMe: () => profile.future,
          ),
        ),
      ],
    );
    final provider = onboardingFlowProvider(registry.activeLease!);
    addTearDown(container.listen(provider, (_, _) {}).close);
    await container.read(provider.future);
    container
        .read(provider.notifier)
        .updateIdentity(displayName: 'My new name');

    profile.complete(
      Profile(
        did: 'did:plc:alice',
        handle: 'alice.test',
        displayName: 'Delayed name',
        crafts: const [],
      ),
    );
    await Future<void>.delayed(Duration.zero);
    await Future<void>.delayed(Duration.zero);

    expect(
      container.read(provider).requireValue.identity.displayName,
      'My new name',
    );
  });

  test('a profile read error leaves the editable draft available', () async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        accountProfileRepositoryProvider.overrideWith(
          (ref, lease) async => FakeProfileRepository(
            onFetchMe: () async => throw StateError('offline'),
          ),
        ),
      ],
    );

    final provider = onboardingFlowProvider(registry.activeLease!);
    addTearDown(container.listen(provider, (_, _) {}).close);
    final state = await container.read(provider.future);
    await Future<void>.delayed(Duration.zero);

    expect(state.baseline.handle.value, 'alice.test');
    expect(container.read(provider).hasError, isFalse);
  });
}
