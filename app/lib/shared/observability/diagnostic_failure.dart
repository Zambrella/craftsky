/// App-owned wrappers may preserve a cause without granting its free text
/// diagnostic provenance. Ownership is local to this operation occurrence.
abstract interface class DiagnosticFailureCause {
  Object? get diagnosticCause;
  StackTrace? get diagnosticStack;
}

/// A reviewed developer-written StateError explanation, never user input,
/// dependency prose or interpolated payloads. Keeps StateError catch semantics.
final class DiagnosticStateError extends StateError {
  DiagnosticStateError(super.message);
}
