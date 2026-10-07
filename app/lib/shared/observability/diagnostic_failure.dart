/// App-owned wrappers may preserve a cause without granting its free text
/// diagnostic provenance. Ownership is local to this operation occurrence.
abstract interface class DiagnosticFailureCause {
  Object? get diagnosticCause;
  StackTrace? get diagnosticStack;
}
