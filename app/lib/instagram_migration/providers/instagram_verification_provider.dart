import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_account_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'instagram_verification_provider.g.dart';

final instagramVerificationPollIntervalProvider = Provider<Duration>(
  (_) => const Duration(seconds: 2),
);

final instagramVerificationNowProvider = Provider<DateTime Function()>(
  (_) =>
      () => DateTime.now().toUtc(),
);

@immutable
final class InstagramVerificationViewState {
  const InstagramVerificationViewState({
    this.attempt,
    this.isBusy = false,
    this.hasError = false,
  });

  final InstagramVerificationAttempt? attempt;
  final bool isBusy;
  final bool hasError;

  InstagramVerificationViewState copyWith({
    InstagramVerificationAttempt? attempt,
    bool clearAttempt = false,
    bool? isBusy,
    bool? hasError,
  }) => InstagramVerificationViewState(
    attempt: clearAttempt ? null : attempt ?? this.attempt,
    isBusy: isBusy ?? this.isBusy,
    hasError: hasError ?? this.hasError,
  );

  @override
  String toString() => 'InstagramVerificationViewState([REDACTED])';
}

@riverpod
class InstagramVerification extends _$InstagramVerification {
  Timer? _pollTimer;
  Timer? _expiryTimer;
  bool _pollInFlight = false;

  @override
  InstagramVerificationViewState build(ActiveAccountLease lease) {
    ref.onDispose(_stopTimers);
    return const InstagramVerificationViewState();
  }

  Future<bool> create() async {
    _stopTimers();
    state = state.copyWith(isBusy: true, hasError: false, clearAttempt: true);
    try {
      final repository = await _repository();
      final attempt = await repository.createVerification();
      ensureInstagramOperationCurrent(ref, lease);
      state = InstagramVerificationViewState(attempt: attempt);
      _schedule(attempt);
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object {
      if (!_isCurrent) return false;
      state = state.copyWith(isBusy: false, hasError: true);
      return false;
    }
  }

  Future<bool> poll() async {
    final current = state.attempt;
    if (_pollInFlight || current == null || !_shouldPoll(current.state)) {
      return false;
    }
    _pollInFlight = true;
    try {
      final repository = await _repository();
      final status = await repository.getVerification(current.verificationId);
      ensureInstagramOperationCurrent(ref, lease);
      final merged = _mergeStatus(current, status);
      state = state.copyWith(attempt: merged, hasError: false);
      if (_isTerminal(merged.state)) {
        _stopTimers();
      } else if (!_shouldPoll(merged.state)) {
        _pollTimer?.cancel();
        _pollTimer = null;
      }
      return true;
    } on InstagramOperationDiscarded {
      _stopTimers();
      return false;
    } on Object {
      if (!_isCurrent) return false;
      state = state.copyWith(hasError: true);
      return false;
    } finally {
      _pollInFlight = false;
    }
  }

  Future<bool> cancel() async {
    final current = state.attempt;
    if (current == null) return false;
    state = state.copyWith(isBusy: true, hasError: false);
    try {
      final repository = await _repository();
      await repository.cancelVerification(current.verificationId);
      ensureInstagramOperationCurrent(ref, lease);
      _stopTimers();
      state = InstagramVerificationViewState(
        attempt: _withState(current, InstagramVerificationState.cancelled),
      );
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object {
      if (!_isCurrent) return false;
      state = state.copyWith(isBusy: false, hasError: true);
      return false;
    }
  }

  Future<bool> confirm({required bool discoverable}) async {
    final current = state.attempt;
    if (current == null ||
        current.state != InstagramVerificationState.pendingConfirmation) {
      return false;
    }
    state = state.copyWith(isBusy: true, hasError: false);
    try {
      final repository = await _repository();
      final confirmation = await repository.confirmVerification(
        current.verificationId,
        discoverable: discoverable,
      );
      ensureInstagramOperationCurrent(ref, lease);
      _stopTimers();
      state = InstagramVerificationViewState(
        attempt: _withState(current, confirmation.state),
      );
      ref.invalidate(instagramAccountProvider(lease));
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object {
      if (!_isCurrent) return false;
      state = state.copyWith(isBusy: false, hasError: true);
      return false;
    }
  }

  Future<InstagramMigrationRepository> _repository() async {
    final repository = await ref.read(
      instagramMigrationRepositoryProvider(lease).future,
    );
    ensureInstagramOperationCurrent(ref, lease);
    return repository;
  }

  void _schedule(InstagramVerificationAttempt attempt) {
    _stopTimers();
    if (_isTerminal(attempt.state)) return;
    final now = ref.read(instagramVerificationNowProvider)();
    final untilExpiry = attempt.expiresAt.difference(now);
    if (untilExpiry <= Duration.zero) {
      state = state.copyWith(
        attempt: _withState(attempt, InstagramVerificationState.expired),
      );
      return;
    }
    _expiryTimer = Timer(untilExpiry, () {
      if (!_isCurrent || !ref.mounted) return;
      final current = state.attempt;
      if (current == null) return;
      _stopTimers();
      state = state.copyWith(
        attempt: _withState(current, InstagramVerificationState.expired),
      );
    });
    if (_shouldPoll(attempt.state)) {
      _pollTimer = Timer.periodic(
        ref.read(instagramVerificationPollIntervalProvider),
        (_) => unawaited(poll()),
      );
    }
  }

  void _stopTimers() {
    _pollTimer?.cancel();
    _expiryTimer?.cancel();
    _pollTimer = null;
    _expiryTimer = null;
  }

  bool get _isCurrent {
    if (!ref.mounted) return false;
    try {
      ensureInstagramOperationCurrent(ref, lease);
      return true;
    } on InstagramOperationDiscarded {
      return false;
    }
  }
}

InstagramVerificationAttempt _mergeStatus(
  InstagramVerificationAttempt previous,
  InstagramVerificationAttempt status,
) => InstagramVerificationAttempt(
  verificationId: status.verificationId,
  state: status.state,
  expiresAt: status.expiresAt,
  challenge: previous.challenge,
  dmUrl: previous.dmUrl,
  candidateUsername: status.candidateUsername,
  retryCode: status.retryCode,
);

InstagramVerificationAttempt _withState(
  InstagramVerificationAttempt attempt,
  InstagramVerificationState next,
) => InstagramVerificationAttempt(
  verificationId: attempt.verificationId,
  state: next,
  expiresAt: attempt.expiresAt,
  challenge: attempt.challenge,
  dmUrl: attempt.dmUrl,
  candidateUsername: attempt.candidateUsername,
  retryCode: attempt.retryCode,
);

bool _isTerminal(InstagramVerificationState state) => switch (state) {
  InstagramVerificationState.pendingDm ||
  InstagramVerificationState.processing ||
  InstagramVerificationState.pendingConfirmation => false,
  _ => true,
};

bool _shouldPoll(InstagramVerificationState state) => switch (state) {
  InstagramVerificationState.pendingDm ||
  InstagramVerificationState.processing => true,
  _ => false,
};
