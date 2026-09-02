import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/onboarding/data/onboarding_repository.dart';
import 'package:craftsky_app/onboarding/models/onboarding_completion.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_repository_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
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

final class _Repository implements OnboardingRepository {
  _Repository({required this.status, required this.completions});

  final OnboardingCompletion status;
  final List<Future<OnboardingCompletion> Function()> completions;
  int completeCalls = 0;

  @override
  Future<OnboardingCompletion> readStatus() async => status;

  @override
  Future<OnboardingCompletion> complete() {
    final result = completions[completeCalls.clamp(0, completions.length - 1)];
    completeCalls++;
    return result();
  }
}

void main() {
  test('loads server status for the exact account session lease', () async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final repository = _Repository(
      status: const OnboardingCompletion(completed: false),
      completions: [() async => const OnboardingCompletion(completed: true)],
    );
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        onboardingRepositoryProvider.overrideWith(
          (ref, lease) async => repository,
        ),
      ],
    );

    final lease = registry.activeLease!.session;
    expect(
      await container.read(onboardingStatusProvider(lease).future),
      const OnboardingCompletion(completed: false),
    );
  });

  test(
    'completes optimistically and retries transient failure silently',
    () async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final first = Completer<OnboardingCompletion>();
      final repository = _Repository(
        status: const OnboardingCompletion(completed: false),
        completions: [
          () => first.future,
          () async => OnboardingCompletion(
            completed: true,
            completedAt: DateTime.utc(2026, 8, 31),
          ),
        ],
      );
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          onboardingRepositoryProvider.overrideWith(
            (ref, lease) async => repository,
          ),
          onboardingCompletionRetryDelaysProvider.overrideWithValue(const [
            Duration.zero,
          ]),
        ],
      );
      final lease = registry.activeLease!.session;
      await container.read(onboardingStatusProvider(lease).future);

      final operation = container
          .read(onboardingStatusProvider(lease).notifier)
          .completeOptimistically();
      expect(
        container.read(onboardingStatusProvider(lease)).requireValue.completed,
        isTrue,
      );

      first.completeError(StateError('transient'));
      await operation;
      expect(repository.completeCalls, 2);
      expect(
        container
            .read(onboardingStatusProvider(lease))
            .requireValue
            .completedAt,
        DateTime.utc(2026, 8, 31),
      );
    },
  );

  test('retry stops when the owning session lease is replaced', () async {
    final storage = _Storage(
      SessionRegistry.empty().upsertAndActivate(
        token: 'old',
        did: 'did:plc:alice',
        handle: 'alice.test',
      ),
    );
    final repository = _Repository(
      status: const OnboardingCompletion(completed: false),
      completions: [() async => throw StateError('offline')],
    );
    final delay = Completer<void>();
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(storage),
        onboardingRepositoryProvider.overrideWith(
          (ref, lease) async => repository,
        ),
        onboardingCompletionDelayProvider.overrideWithValue(
          (_) => delay.future,
        ),
      ],
    );
    final oldLease = storage.value.activeLease!.session;
    await container.read(onboardingStatusProvider(oldLease).future);
    final operation = container
        .read(onboardingStatusProvider(oldLease).notifier)
        .completeOptimistically();
    await Future<void>.delayed(Duration.zero);

    await container.read(sessionRegistryProvider.future);
    await container
        .read(sessionRegistryProvider.notifier)
        .upsertAndActivate(
          token: 'new',
          did: AccountKey('did:plc:alice').did.value,
          handle: 'alice.test',
        );
    delay.complete();
    await operation;
    expect(repository.completeCalls, 1);
  });
}
