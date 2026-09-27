import 'dart:async';
import 'dart:math' as math;

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

typedef PdsMutationScheduler =
    void Function(Duration delay, void Function() callback);

enum PdsMutationStatus { inFlight, ambiguous, failed }

@immutable
final class PdsMutationScope {
  const PdsMutationScope({required this.lease, required this.identity});

  final AccountSessionLease lease;
  final String identity;

  @override
  bool operator ==(Object other) =>
      other is PdsMutationScope &&
      other.lease == lease &&
      other.identity == identity;

  @override
  int get hashCode => Object.hash(lease, identity);

  @override
  String toString() => 'PdsMutationScope(<redacted>)';
}

@immutable
final class PdsMutationToken {
  const PdsMutationToken._({
    required this.scope,
    required this.operationKey,
    required this.endpoint,
    required this.immutableBody,
    required this.sequence,
    required this._epoch,
  });

  final PdsMutationScope scope;
  final String operationKey;
  final String endpoint;
  final String immutableBody;
  final int sequence;
  final int _epoch;
}

@immutable
final class PdsMutationOperation {
  const PdsMutationOperation({
    required this.token,
    required this.status,
    this.retryAfterSeconds,
  });

  final PdsMutationToken token;
  final PdsMutationStatus status;
  final int? retryAfterSeconds;

  PdsMutationOperation copyWith({
    required PdsMutationStatus status,
    int? retryAfterSeconds,
  }) => PdsMutationOperation(
    token: token,
    status: status,
    retryAfterSeconds: retryAfterSeconds,
  );
}

final class PdsMutationOverlay {
  PdsMutationOverlay({
    required this.token,
    required this.optimisticValue,
    required this.acceptedAt,
    required this.graceAt,
    required this.expiresAt,
    required this._agrees,
    this.refresh,
  });

  final PdsMutationToken token;
  final Object? optimisticValue;
  final DateTime acceptedAt;
  final DateTime graceAt;
  final DateTime expiresAt;
  final bool Function(Object? authoritativeValue) _agrees;
  final void Function()? refresh;
  bool graceRefreshRequested = false;

  bool agreesWith(Object? authoritativeValue) => _agrees(authoritativeValue);
}

final class PdsRecordOperationController {
  PdsRecordOperationController({
    DateTime Function()? now,
    void Function(PdsMutationScope scope)? refreshScope,
    PdsMutationScheduler? schedule,
  }) : _now = now ?? DateTime.now,
       // Public constructor names intentionally omit private field prefixes.
       // ignore: prefer_initializing_formals
       _refreshScope = refreshScope,
       // Public constructor names intentionally omit private field prefixes.
       // ignore: prefer_initializing_formals
       _schedule = schedule;

  final DateTime Function() _now;
  final void Function(PdsMutationScope scope)? _refreshScope;
  final PdsMutationScheduler? _schedule;
  final _operations = <PdsMutationScope, PdsMutationOperation>{};
  final _overlays = <PdsMutationScope, PdsMutationOverlay>{};
  final _sequences = <PdsMutationScope, int>{};
  final _timers = <Timer>[];
  var _epoch = 0;

  PdsMutationToken begin({
    required PdsMutationScope scope,
    required String operationKey,
    required String endpoint,
    required String immutableBody,
  }) {
    if (_operations[scope]?.status == PdsMutationStatus.ambiguous) {
      throw StateError('An ambiguous mutation must be retried unchanged');
    }
    final sequence = (_sequences[scope] ?? 0) + 1;
    _sequences[scope] = sequence;
    final token = PdsMutationToken._(
      scope: scope,
      operationKey: operationKey,
      endpoint: endpoint,
      immutableBody: immutableBody,
      sequence: sequence,
      epoch: _epoch,
    );
    _operations[scope] = PdsMutationOperation(
      token: token,
      status: PdsMutationStatus.inFlight,
    );
    return token;
  }

  PdsMutationOperation? operationFor(PdsMutationScope scope) =>
      _operations[scope];

  PdsMutationToken? retryToken({
    required PdsMutationScope scope,
    required String endpoint,
    required String immutableBody,
  }) {
    final operation = _operations[scope];
    if (operation?.status != PdsMutationStatus.ambiguous ||
        operation!.token._epoch != _epoch ||
        operation.token.endpoint != endpoint ||
        operation.token.immutableBody != immutableBody) {
      return null;
    }
    return operation.token;
  }

  PdsMutationToken beginOrRetry({
    required PdsMutationScope scope,
    required String endpoint,
    required String immutableBody,
    required String Function() newOperationKey,
  }) {
    final retry = retryToken(
      scope: scope,
      endpoint: endpoint,
      immutableBody: immutableBody,
    );
    if (retry != null) return retry;
    return begin(
      scope: scope,
      operationKey: newOperationKey(),
      endpoint: endpoint,
      immutableBody: immutableBody,
    );
  }

  PdsMutationOverlay? overlayFor(PdsMutationScope scope) => _overlays[scope];

  Iterable<PdsMutationOverlay> get activeOverlays => _overlays.values;

  bool markAmbiguous(
    PdsMutationToken token, {
    required int retryAfterSeconds,
  }) {
    if (!_isCurrent(token)) return false;
    _operations[token.scope] = _operations[token.scope]!.copyWith(
      status: PdsMutationStatus.ambiguous,
      retryAfterSeconds: retryAfterSeconds.clamp(1, 5),
    );
    return true;
  }

  bool canRetry(
    PdsMutationToken token, {
    required String operationKey,
    required String endpoint,
    required String immutableBody,
  }) {
    final operation = _operations[token.scope];
    return _isCurrent(token) &&
        operation?.status == PdsMutationStatus.ambiguous &&
        operationKey == token.operationKey &&
        endpoint == token.endpoint &&
        immutableBody == token.immutableBody;
  }

  bool markFailed(PdsMutationToken token) {
    if (!_isCurrent(token)) return false;
    _operations[token.scope] = _operations[token.scope]!.copyWith(
      status: PdsMutationStatus.failed,
    );
    return true;
  }

  bool markAcceptedWithoutOverlay(PdsMutationToken token) {
    if (!_isCurrent(token)) return false;
    _operations.remove(token.scope);
    return true;
  }

  bool markAccepted(
    PdsMutationToken token, {
    required Object? optimisticValue,
    required bool Function(Object? authoritativeValue) agrees,
    PdsMutationScope? overlayScope,
    void Function()? refresh,
  }) {
    if (!_isCurrent(token)) return false;
    final acceptedAt = _now();
    _operations.remove(token.scope);
    final acceptedScope = overlayScope ?? token.scope;
    _overlays[acceptedScope] = PdsMutationOverlay(
      token: token,
      optimisticValue: optimisticValue,
      acceptedAt: acceptedAt,
      graceAt: acceptedAt.add(const Duration(seconds: 2)),
      expiresAt: acceptedAt.add(const Duration(seconds: 30)),
      agrees: agrees,
      refresh: refresh,
    );
    _refresh(_overlays[acceptedScope]!);
    _scheduleRefresh(const Duration(seconds: 2));
    _scheduleRefresh(const Duration(seconds: 30));
    return true;
  }

  bool reconcile(PdsMutationScope scope, Object? authoritativeValue) {
    final overlay = _overlays[scope];
    if (overlay == null || overlay.token._epoch != _epoch) return false;
    if (!overlay.agreesWith(authoritativeValue)) return false;
    _overlays.remove(scope);
    if (_overlays.isEmpty) _cancelTimers();
    return true;
  }

  void advanceTime() {
    final now = _now();
    for (final entry in _overlays.entries.toList(growable: false)) {
      final overlay = entry.value;
      if (!now.isBefore(overlay.expiresAt)) {
        _overlays.remove(entry.key);
        _refresh(overlay);
      } else if (!overlay.graceRefreshRequested &&
          !now.isBefore(overlay.graceAt)) {
        overlay.graceRefreshRequested = true;
        _refresh(overlay);
      }
    }
  }

  void reset() {
    _epoch++;
    _operations.clear();
    _overlays.clear();
    _sequences.clear();
    _cancelTimers();
  }

  bool _isCurrent(PdsMutationToken token) {
    final current = _operations[token.scope]?.token;
    return token._epoch == _epoch &&
        current?.sequence == token.sequence &&
        current?.operationKey == token.operationKey;
  }

  void _refresh(PdsMutationOverlay overlay) {
    final refresh = overlay.refresh;
    if (refresh != null) {
      refresh();
      return;
    }
    _refreshScope?.call(overlay.token.scope);
  }

  void _scheduleRefresh(Duration delay) {
    final schedule = _schedule;
    if (schedule != null) {
      schedule(delay, advanceTime);
      return;
    }
    late final Timer timer;
    timer = Timer(delay, () {
      _timers.remove(timer);
      advanceTime();
    });
    _timers.add(timer);
  }

  void _cancelTimers() {
    for (final timer in _timers) {
      timer.cancel();
    }
    _timers.clear();
  }
}

final pdsRecordOperationControllerProvider =
    Provider<PdsRecordOperationController>((ref) {
      final controller = PdsRecordOperationController();
      ref.onDispose(controller.reset);
      return controller;
    });

final pdsMutationDelayProvider = Provider<Future<void> Function(Duration)>(
  (ref) => Future<void>.delayed,
);

final pdsMutationNowProvider = Provider<DateTime Function()>(
  (ref) => DateTime.now,
);

final pdsMutationJitterProvider = Provider<int Function(int)>(
  (ref) => math.Random().nextInt,
);

@immutable
final class PdsMutationRetryPolicy {
  const PdsMutationRetryPolicy();

  static const _localBackoffSeconds = <int>[1, 2, 4, 5, 5, 5];
  static const _maximumElapsed = Duration(seconds: 30);

  Duration? nextDelay({
    required int retryIndex,
    required int? retryAfterSeconds,
    required Duration elapsed,
    required int Function(int maximumExclusive) jitterMillis,
  }) {
    if (retryIndex < 0 || retryIndex >= _localBackoffSeconds.length) {
      return null;
    }
    final localSeconds = _localBackoffSeconds[retryIndex];
    final seconds = retryAfterSeconds == null
        ? localSeconds
        : math.max(localSeconds, retryAfterSeconds.clamp(1, 5));
    final jitter = math.max(0, jitterMillis(251)).clamp(0, 250);
    final delay = Duration(seconds: seconds, milliseconds: jitter);
    if (elapsed + delay > _maximumElapsed) return null;
    return delay;
  }
}
