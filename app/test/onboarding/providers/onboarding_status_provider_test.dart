import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
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
  new(this.value);
  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _Repository implements OnboardingRepository {
  new({required this.status, required this.completions});

  final OnboardingCompletion status;
  final List<Future<OnboardingCompletion> Function()> completions;
  int completeCalls = 0;

  @override
  Future<OnboardingCompletion> readStatus() async => status;

  @override
  Future<OnboardingCompletion> complete({
    required bool meetsMinimumAge,
    required String policyVersion,
  }) {
    final result = completions[completeCalls.clamp(0, completions.length - 1)];
    completeCalls++;
    return result();
  }
}

void main() {
  test(
    'explicit provider invalidation stops a pending completion retry',
    () async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final repository = _Repository(
        status: const OnboardingCompletion(completed: false),
        completions: [() async => throw StateError('offline')],
      );
      final enteredDelay = Completer<void>();
      final delay = Completer<void>();
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          onboardingRepositoryProvider.overrideWith(
            (ref, lease) async => repository,
          ),
          onboardingCompletionDelayProvider.overrideWithValue((_) {
            enteredDelay.complete();
            return delay.future;
          }),
        ],
      );
      final provider = onboardingStatusProvider(registry.activeLease!.session);
      await container.read(provider.future);
      final pending = container
          .read(provider.notifier)
          .completeOptimistically(meetsMinimumAge: true);
      await enteredDelay.future;
      container.invalidate(provider);
      await container.pump();
      delay.complete();
      await expectLater(pending, completes);
      expect(repository.completeCalls, 1);
    },
  );

  test(
    'a replaced account lease is cancellation rather than an invariant failure',
    () async {
      final original = SessionRegistry.empty().upsertAndActivate(
        token: 'token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final replacement = original.upsertAndActivate(
        token: 'replacement',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(replacement),
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      await expectLater(
        container.read(
          onboardingStatusProvider(original.activeLease!.session).future,
        ),
        throwsA(isA<AccountOperationCanceled>()),
      );
    },
  );

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
          .completeOptimistically(meetsMinimumAge: true);
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
        .completeOptimistically(meetsMinimumAge: true);
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

  test('retry exhaustion restores incomplete server state', () async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final repository = _Repository(
      status: const OnboardingCompletion(completed: false),
      completions: [() async => throw StateError('offline')],
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
        onboardingCompletionRetryDelaysProvider.overrideWithValue(const []),
      ],
    );
    final lease = registry.activeLease!.session;
    await container.read(onboardingStatusProvider(lease).future);

    await container
        .read(onboardingStatusProvider(lease).notifier)
        .completeOptimistically(meetsMinimumAge: true);

    expect(
      container.read(onboardingStatusProvider(lease)).requireValue.completed,
      isFalse,
    );
  });
}
