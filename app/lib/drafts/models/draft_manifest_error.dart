import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';

enum DraftManifestFailureReason { unsupportedVersion, corrupt, invalidMedia }

/// Content-free presentation with private, transient diagnostic causes.
final class DraftManifestException
    implements Exception, DiagnosticFailureCause {
  const new(this.reason, {this.diagnosticCause, this.diagnosticStack});

  final DraftManifestFailureReason reason;
  @override
  final Object? diagnosticCause;
  @override
  final StackTrace? diagnosticStack;

  @override
  String toString() => 'DraftManifestException(${reason.name})';
}
