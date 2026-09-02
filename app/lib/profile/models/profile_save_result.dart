import 'package:craftsky_app/shared/api/api_exception.dart';

enum ProfileSavePortion { ordinary, business }

enum ProfileSaveOutcomeStatus { skipped, succeeded, failed }

enum ProfileSaveFailureKind { general, conflict }

class PerRecordSaveOutcome<T> {
  const PerRecordSaveOutcome.skipped()
    : status = ProfileSaveOutcomeStatus.skipped,
      value = null,
      error = null,
      stackTrace = null,
      failureKind = null;

  const PerRecordSaveOutcome.success(T this.value)
    : status = ProfileSaveOutcomeStatus.succeeded,
      error = null,
      stackTrace = null,
      failureKind = null;

  factory PerRecordSaveOutcome.failure(Object error, [StackTrace? stackTrace]) {
    return PerRecordSaveOutcome._failure(
      error,
      stackTrace,
      error is ApiBadRequest && error.code == 'pds_record_conflict'
          ? ProfileSaveFailureKind.conflict
          : ProfileSaveFailureKind.general,
    );
  }

  const PerRecordSaveOutcome._failure(
    this.error,
    this.stackTrace,
    this.failureKind,
  ) : status = ProfileSaveOutcomeStatus.failed,
      value = null;

  final ProfileSaveOutcomeStatus status;
  final T? value;
  final Object? error;
  final StackTrace? stackTrace;
  final ProfileSaveFailureKind? failureKind;

  bool get wasRequested => status != ProfileSaveOutcomeStatus.skipped;
  bool get succeeded => status == ProfileSaveOutcomeStatus.succeeded;
  bool get failed => status == ProfileSaveOutcomeStatus.failed;
}

class CombinedProfileSaveResult {
  const CombinedProfileSaveResult({
    required this.ordinary,
    required this.business,
  });

  final PerRecordSaveOutcome<Object?> ordinary;
  final PerRecordSaveOutcome<Object?> business;

  bool get isFullSuccess =>
      (ordinary.wasRequested || business.wasRequested) &&
      !ordinary.failed &&
      !business.failed;

  bool get isPartialSuccess =>
      (ordinary.succeeded || business.succeeded) &&
      (ordinary.failed || business.failed);

  bool get retryOrdinary => ordinary.failed;
  bool get retryBusiness => business.failed;

  Set<ProfileSavePortion> get failedPortions => {
    if (ordinary.failed) ProfileSavePortion.ordinary,
    if (business.failed) ProfileSavePortion.business,
  };
}
