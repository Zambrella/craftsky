import 'dart:async';

import 'package:craftsky_app/service_status/data/announcement_dismissal_store.dart';
import 'package:craftsky_app/service_status/data/service_status_repository.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/status_runtime_provider.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:logging/logging.dart';

final serviceStatusReporterProvider = Provider<ErrorReporter>(
  (ref) => const NoopErrorReporter(),
);

final serviceStatusRepositoryProvider = Provider<ServiceStatusRepository>(
  (ref) {
    final repository = createServiceStatusRepository(
      validateStatusEndpoint(
        const String.fromEnvironment(
          'CRAFTSKY_STATUS_URL',
          defaultValue: 'https://status.craftsky.social/app.json',
        ),
        production: !kDebugMode,
      ),
    );
    ref.onDispose(repository.close);
    return repository;
  },
);
final announcementDismissalStoreProvider = Provider<AnnouncementDismissalStore>(
  (ref) => PreferencesAnnouncementDismissalStore(),
);
final serviceStatusControllerProvider =
    NotifierProvider<ServiceStatusController, ServiceStatusState>(
      ServiceStatusController.new,
    );

enum StatusPresentation { none, maintenance, updateRequired, message }

final class ServiceStatusState {
  const new({
    this.document,
    this.fetching = false,
    this.dismissalReady = false,
    this.dismissedRevision,
    this.dismissing = false,
    this.updatePolicyTrusted = false,
    this.activating = false,
    this.actionFailed = false,
  });
  final ServiceStatusDocument? document;
  final bool fetching;
  final bool dismissalReady;
  final String? dismissedRevision;
  final bool dismissing;
  final bool updatePolicyTrusted;
  final bool activating;
  final bool actionFailed;

  ServiceStatusState copyWith({
    ServiceStatusDocument? document,
    bool? fetching,
    bool? dismissalReady,
    String? dismissedRevision,
    bool? dismissing,
    bool? updatePolicyTrusted,
    bool? activating,
    bool? actionFailed,
    bool clearDocument = false,
    bool clearDismissedRevision = false,
  }) => ServiceStatusState(
    document: clearDocument ? null : document ?? this.document,
    fetching: fetching ?? this.fetching,
    dismissalReady: dismissalReady ?? this.dismissalReady,
    dismissedRevision: clearDismissedRevision
        ? null
        : dismissedRevision ?? this.dismissedRevision,
    dismissing: dismissing ?? this.dismissing,
    updatePolicyTrusted: updatePolicyTrusted ?? this.updatePolicyTrusted,
    activating: activating ?? this.activating,
    actionFailed: actionFailed ?? this.actionFailed,
  );

  @override
  String toString() =>
      'ServiceStatusState(mode: ${document?.mode.name ?? 'unknown'}, '
      'fetching: $fetching)';
}

class ServiceStatusController extends Notifier<ServiceStatusState> {
  static final _log = Logger('ServiceStatus');
  void _diagnostic(
    Object error,
    StackTrace stack,
    String operation, {
    bool expectedContentFailure = false,
  }) {
    final expected =
        expectedContentFailure ||
        error is FormatException ||
        error is TimeoutException ||
        error is StatusNetworkFailure ||
        isExpectedStatusPlatformFailure(error) ||
        (error is DioException &&
            (error.type != DioExceptionType.unknown ||
                (error.error != null &&
                    isExpectedStatusPlatformFailure(error.error!))));
    final context = ReportContext(
      feature: 'ServiceStatus',
      operation: operation,
      classification: expected ? 'status.unavailable' : 'status.unexpected',
      outcome: expected
          ? DiagnosticOutcome.expected
          : DiagnosticOutcome.automatic,
    );
    try {
      _log.log(
        expected ? Level.FINE : Level.WARNING,
        DiagnosticMessage('Status operation failed', context: context),
        error,
        stack,
      );
      if (!expected) {
        final reporter = ref.read(serviceStatusReporterProvider);
        unawaited(
          GuardedErrorReporter(
            reporter,
          ).captureException(error, context: context, stackTrace: stack),
        );
      }
    } on Object {
      /* Diagnostics never change feature behavior. */
    }
  }

  void reportImageFailure(
    Object error,
    StackTrace stack, {
    bool expectedContentFailure = false,
  }) => _diagnostic(
    error,
    stack,
    'image',
    expectedContentFailure: expectedContentFailure,
  );

  void reportActionFailure(Object error, StackTrace stack) =>
      _diagnostic(error, stack, 'action');

  void _cancelRequest() {
    try {
      _request?.cancel();
    } on Object catch (error, stack) {
      _diagnostic(error, stack, 'cancel');
    }
  }

  T? _resolve<T>(Provider<T> provider) {
    try {
      return ref.read(provider);
    } on Object {
      // Riverpod's observer owns provider construction failures. Do not
      // capture its rethrown ProviderException as another occurrence.
      return null;
    }
  }

  Timer? _expiry;
  StatusAgeSample? _acceptedAt;
  Timer? _poll;
  Timer? _deadline;
  StatusRequest? _request;
  Completer<void>? _pending;
  var _foreground = true;
  var _started = false;
  var _disposed = false;
  var _epoch = 0;
  var _dismissalGeneration = 0;
  var _dismissalOrder = 0;
  var _persistedDismissalOrder = 0;
  final _activatingRevisions = <String>{};
  final _failedActionRevisions = <String>{};
  final _sessionDismissed = <String>{};
  final _dismissingRevisions = <String>{};
  Future<void> _writeTail = Future.value();

  @override
  ServiceStatusState build() {
    ref.listen(statusRuntimeVersionProvider, (_, _) {
      if (!_disposed) state = state.copyWith();
    });
    ref.listen(statusPlatformProvider, (_, _) {
      if (!_disposed) state = state.copyWith();
    });
    ref.onDispose(() {
      _disposed = true;
      _epoch++;
      _expiry?.cancel();
      _poll?.cancel();
      _deadline?.cancel();
      _cancelRequest();
      _pending?.complete();
      _pending = null;
    });
    unawaited(Future.microtask(_loadDismissal));
    return const ServiceStatusState();
  }

  StatusPresentation get presentation {
    if (maintenanceActive) return StatusPresentation.maintenance;
    final version = ref.read(statusRuntimeVersionProvider).value;
    final platform = ref.read(statusPlatformProvider);
    if (state.updatePolicyTrusted &&
        requiresStatusUpdate(
          state.document?.minimumVersions,
          platform,
          version,
        )) {
      return StatusPresentation.updateRequired;
    }
    final message = state.document?.message;
    if (platform != null &&
        state.dismissalReady &&
        statusMessageEligible(message, version) &&
        message!.revision != state.dismissedRevision &&
        !_sessionDismissed.contains(message.revision)) {
      return StatusPresentation.message;
    }
    return StatusPresentation.none;
  }

  bool get blockingActive =>
      presentation == StatusPresentation.maintenance ||
      presentation == StatusPresentation.updateRequired;

  StatusMessage? get announcement => presentation == StatusPresentation.message
      ? state.document?.message
      : null;

  Future<void> _loadDismissal() async {
    final generation = _dismissalGeneration;
    String? revision;
    try {
      revision = await _resolve(
        announcementDismissalStoreProvider,
      )?.readRevision();
    } on Object catch (error, stack) {
      _diagnostic(error, stack, 'dismissalRead');
    }
    if (!_disposed) {
      state = state.copyWith(
        dismissalReady: true,
        dismissedRevision: generation == _dismissalGeneration
            ? revision
            : state.dismissedRevision,
      );
    }
  }

  Future<bool> _persistRevision(String revision, int order) {
    final operation = _writeTail.then((_) async {
      // Destination futures can finish out of order. Check inside the write
      // queue, after any newer dismissal ahead of us has finished saving.
      if (order < _persistedDismissalOrder) return false;
      try {
        final store = _resolve(announcementDismissalStoreProvider);
        if (store == null) return false;
        await store.writeRevision(revision);
        _persistedDismissalOrder = order;
        if (!_disposed) {
          _dismissalGeneration++;
          state = state.copyWith(dismissedRevision: revision);
        }
        return true;
      } on Object catch (error, stack) {
        _diagnostic(error, stack, 'dismissalWrite');
        return false;
      }
    });
    _writeTail = operation.then((_) {});
    return operation;
  }

  Future<bool> dismissAnnouncement() async {
    final revision = announcement?.revision;
    if (_disposed || revision == null || !_dismissingRevisions.add(revision)) {
      return false;
    }
    final order = ++_dismissalOrder;
    state = state.copyWith(dismissing: true);
    final saved = await _persistRevision(revision, order);
    _dismissingRevisions.remove(revision);
    if (_disposed) return false;
    state = state.copyWith(
      dismissing: _dismissingRevisions.contains(
        state.document?.message?.revision,
      ),
    );
    return saved;
  }

  void _updateActionState() {
    if (_disposed) return;
    final revision = state.document?.message?.revision;
    state = state.copyWith(
      activating: _activatingRevisions.contains(revision),
      actionFailed: _failedActionRevisions.contains(revision),
    );
  }

  Future<bool> activateMessage(
    String revision,
    Future<bool> Function() invoke,
  ) async {
    if (_disposed ||
        announcement?.revision != revision ||
        !_activatingRevisions.add(revision)) {
      return false;
    }
    final order = ++_dismissalOrder;
    _failedActionRevisions.remove(revision);
    _updateActionState();
    var activated = false;
    try {
      // Start the destination directly from the gesture, before persistence.
      activated = await invoke();
    } on Object catch (error, stack) {
      _diagnostic(error, stack, 'action');
    }
    if (_disposed) return false;
    if (!activated) {
      _failedActionRevisions.add(revision);
      _activatingRevisions.remove(revision);
      _updateActionState();
      return false;
    }
    _sessionDismissed.add(revision);
    _updateActionState();
    final saved = await _persistRevision(revision, order);
    if (_disposed) return true;
    if (saved) {
      _sessionDismissed.remove(revision);
    }
    _activatingRevisions.remove(revision);
    _updateActionState();
    return true;
  }

  bool get maintenanceActive {
    if (state.document?.maintenance == null || _acceptedAt == null) {
      return false;
    }
    final now = ref.read(serviceStatusClockProvider).sample();
    final wallAge = now.wall.difference(_acceptedAt!.wall);
    final elapsed = now.elapsed - _acceptedAt!.elapsed;
    if (wallAge.isNegative || elapsed.isNegative) {
      _acceptedAt = null;
      return false;
    }
    final age = elapsed > wallAge ? elapsed : wallAge;
    if (age >= const Duration(minutes: 5)) {
      _acceptedAt = null;
      return false;
    }
    return true;
  }

  void _scheduleExpiry() {
    _expiry?.cancel();
    if (state.document?.maintenance != null) {
      _expiry = Timer(const Duration(minutes: 5), () {
        if (!_disposed) {
          _acceptedAt = null;
          state = state.copyWith();
        }
      });
    }
  }

  void start() {
    if (_started || _disposed) return;
    _started = true;
    _schedulePolling();
    unawaited(refresh());
  }

  void _schedulePolling() {
    _poll?.cancel();
    if (_foreground && _started) {
      _poll = Timer.periodic(const Duration(seconds: 60), (_) {
        unawaited(refresh());
      });
    }
  }

  void setForeground({required bool foreground}) {
    if (_disposed || _foreground == foreground) return;
    _foreground = foreground;
    _schedulePolling();
    if (foreground && _started) unawaited(refresh());
  }

  Future<void> refresh() {
    if (_disposed) return Future.value();
    if (_pending != null) return _pending!.future;
    final completion = Completer<void>();
    _pending = completion;
    final epoch = ++_epoch;
    state = state.copyWith(fetching: true);
    void finish({bool accepted = false}) {
      if (_disposed || epoch != _epoch) return;
      _deadline?.cancel();
      _request = null;
      _pending = null;
      state = state.copyWith(fetching: false, updatePolicyTrusted: accepted);
      completion.complete();
    }

    _deadline = Timer(const Duration(seconds: 3), () {
      _cancelRequest();
      finish();
      _epoch++;
    });
    try {
      final repository = _resolve(serviceStatusRepositoryProvider);
      if (repository == null) {
        finish();
        return completion.future;
      }
      _request = repository.start();
      unawaited(
        _request!.result.then(
          (document) {
            if (_disposed || epoch != _epoch) return;
            _acceptedAt = ref.read(serviceStatusClockProvider).sample();
            state = state.copyWith(
              document: document,
              fetching: true,
              dismissing: _dismissingRevisions.contains(
                document.message?.revision,
              ),
              activating: _activatingRevisions.contains(
                document.message?.revision,
              ),
              actionFailed: _failedActionRevisions.contains(
                document.message?.revision,
              ),
            );
            _scheduleExpiry();
            try {
              _log.fine(
                DiagnosticMessage(
                  'Status accepted',
                  context: ReportContext(
                    feature: 'ServiceStatus',
                    operation: 'fetch',
                    classification: 'status.accepted',
                    safeDiagnostics: {'mode': document.mode.name},
                  ),
                ),
              );
            } on Object {
              /* Diagnostics must not change acceptance. */
            }
            finish(accepted: true);
          },
          onError: (Object error, StackTrace stack) {
            if (_disposed || epoch != _epoch) return;
            _diagnostic(error, stack, 'fetch');
            finish();
          },
        ),
      );
    } on Object catch (error, stack) {
      _diagnostic(error, stack, 'fetch');
      finish();
    }
    return completion.future;
  }
}

// Injectable clock boundary for deterministic freshness tests.
abstract interface class ServiceStatusClock {
  StatusAgeSample sample();
}

final class StatusAgeSample {
  const new(this.elapsed, this.wall);
  final Duration elapsed;
  final DateTime wall;
}

final class SystemServiceStatusClock implements ServiceStatusClock {
  final _stopwatch = Stopwatch()..start();
  @override
  StatusAgeSample sample() =>
      StatusAgeSample(_stopwatch.elapsed, DateTime.now().toUtc());
}

final serviceStatusClockProvider = Provider<ServiceStatusClock>(
  (ref) => SystemServiceStatusClock(),
);
