enum VideoOperation {
  authorization,
  limits,
  upload,
  polling,
  publication,
  playback,
}

enum VideoOperationOutcome { succeeded, rejected, canceled, retrying, failed }

final class VideoDiagnosticEvent {
  const VideoDiagnosticEvent({
    required this.operation,
    required this.outcome,
    this.byteCount,
    this.requestId,
  });

  final VideoOperation operation;
  final VideoOperationOutcome outcome;
  final int? byteCount;
  final String? requestId;

  String get byteBand => switch (byteCount) {
    null => 'unknown',
    < 1000000 => '<1mb',
    < 100000000 => '1-99mb',
    < 300000000 => '100-299mb',
    _ => '300mb+',
  };

  @override
  String toString() =>
      'VideoDiagnosticEvent(operation: ${operation.name}, '
      'outcome: ${outcome.name}, bytes: $byteBand, '
      'requestId: ${requestId ?? 'none'})';
}
