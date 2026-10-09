import 'dart:async';

import 'package:craftsky_app/service_status/data/announcement_dismissal_store.dart';
import 'package:craftsky_app/service_status/data/service_status_repository.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
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

final class ServiceStatusState {
  const ServiceStatusState({
    this.document,
    this.fetching = false,
    this.dismissalReady = false,
    this.dismissedRevision,
    this.dismissing = false,
  });
  final ServiceStatusDocument? document;
  final bool fetching;
  final bool dismissalReady;
  final String? dismissedRevision;
  final bool dismissing;

  ServiceStatusState copyWith({
    ServiceStatusDocument? document,
    bool? fetching,
    bool? dismissalReady,
    String? dismissedRevision,
    bool? dismissing,
  }) => ServiceStatusState(
    document: document ?? this.document,
    fetching: fetching ?? this.fetching,
    dismissalReady: dismissalReady ?? this.dismissalReady,
    dismissedRevision: dismissedRevision ?? this.dismissedRevision,
    dismissing: dismissing ?? this.dismissing,
  );

  @override
  String toString() =>
      'ServiceStatusState(mode: ${document?.mode.name ?? 'unknown'}, '
      'fetching: $fetching)';
}

class ServiceStatusController extends Notifier<ServiceStatusState> {
  static final _log = Logger('ServiceStatus');
  void _diagnostic(Object error, StackTrace stack, String operation) {
    final expected =
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

  @override
  ServiceStatusState build() {
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

  ServiceStatusDocument? get announcement =>
      state.dismissalReady &&
          state.document?.mode == ServiceStatusMode.announcement &&
          state.document?.revision != state.dismissedRevision
      ? state.document
      : null;

  Future<void> _loadDismissal() async {
    String? revision;
    try {
      revision = await _resolve(
        announcementDismissalStoreProvider,
      )?.readRevision();
    } on Object catch (error, stack) {
      _diagnostic(error, stack, 'dismissalRead');
    }
    if (!_disposed) {
      state = state.copyWith(dismissalReady: true, dismissedRevision: revision);
    }
  }

  Future<bool> dismissAnnouncement() async {
    final revision = announcement?.revision;
    if (_disposed || revision == null || state.dismissing) return false;
    state = state.copyWith(dismissing: true);
    try {
      final store = _resolve(announcementDismissalStoreProvider);
      if (store == null) {
        state = state.copyWith(dismissing: false);
        return false;
      }
      await store.writeRevision(revision);
      if (_disposed) return false;
      state = state.copyWith(dismissedRevision: revision, dismissing: false);
      return true;
    } on Object catch (error, stack) {
      _diagnostic(error, stack, 'dismissalWrite');
      if (!_disposed) state = state.copyWith(dismissing: false);
      return false;
    }
  }

  bool get maintenanceActive {
    if (state.document?.mode != ServiceStatusMode.maintenance ||
        _acceptedAt == null) {
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
    if (state.document?.mode == ServiceStatusMode.maintenance) {
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
    void finish() {
      if (_disposed || epoch != _epoch) return;
      _deadline?.cancel();
      _request = null;
      _pending = null;
      state = state.copyWith(fetching: false);
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
            state = state.copyWith(document: document, fetching: true);
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
            finish();
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
// ignore: one_member_abstracts
abstract interface class ServiceStatusClock {
  StatusAgeSample sample();
}

final class StatusAgeSample {
  const StatusAgeSample(this.elapsed, this.wall);
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
