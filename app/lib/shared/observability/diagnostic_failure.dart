/// App-owned wrappers may preserve a cause without granting its free text
/// diagnostic provenance. Ownership is local to this operation occurrence.
abstract interface class DiagnosticFailureCause {
  Object? get diagnosticCause;
  StackTrace? get diagnosticStack;
}

/// A reviewed developer-written StateError explanation, never user input,
/// dependency prose or interpolated payloads. Keeps StateError catch semantics.
final class DiagnosticStateError extends StateError {
  new(super.message);
}

/// A static, reviewed format explanation with an optional original cause.
/// Never construct the message from input, mapper prose, or dependency output.
final class DiagnosticFormatException extends FormatException
    implements DiagnosticFailureCause {
  const new(super.message, {this.diagnosticCause, this.diagnosticStack});

  @override
  final Object? diagnosticCause;
  @override
  final StackTrace? diagnosticStack;
}
