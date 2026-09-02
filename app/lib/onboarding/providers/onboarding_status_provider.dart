import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/onboarding/models/onboarding_completion.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'onboarding_status_provider.g.dart';

typedef OnboardingCompletionDelay = Future<void> Function(Duration duration);

final onboardingCompletionRetryDelaysProvider = Provider<List<Duration>>(
  (_) => const [
    Duration(milliseconds: 250),
    Duration(seconds: 1),
    Duration(seconds: 2),
  ],
);

final onboardingCompletionDelayProvider = Provider<OnboardingCompletionDelay>(
  (_) => Future<void>.delayed,
);

final _log = Logger('OnboardingStatus');

@riverpod
class OnboardingStatus extends _$OnboardingStatus {
  @override
  Future<OnboardingCompletion> build(AccountSessionLease lease) async {
    final registry = await ref.watch(sessionRegistryProvider.future);
    if (registry.leaseFor(lease.account) != lease) {
      throw StateError('Account session unavailable');
    }
    final repository = await ref.watch(
      onboardingRepositoryProvider(lease).future,
    );
    return repository.readStatus();
  }

  Future<void> completeOptimistically() async {
    if (state.value?.completed ?? false) return;
    final keepAlive = ref.keepAlive();
    state = const AsyncData(OnboardingCompletion(completed: true));
    try {
      final repository = await ref.read(
        onboardingRepositoryProvider(lease).future,
      );
      final delays = ref.read(onboardingCompletionRetryDelaysProvider);
      for (var attempt = 0; ; attempt++) {
        if (!_ownsLease()) return;
        try {
          final completed = await repository.complete();
          if (ref.mounted && _ownsLease()) state = AsyncData(completed);
          return;
        } on Object {
          if (attempt >= delays.length || !_ownsLease()) {
            _log.warning('Onboarding completion retry exhausted');
            return;
          }
          await ref.read(onboardingCompletionDelayProvider)(delays[attempt]);
        }
      }
    } finally {
      keepAlive.close();
    }
  }

  bool _ownsLease() =>
      ref.read(sessionRegistryProvider).value?.leaseFor(lease.account) == lease;
}
