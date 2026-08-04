enum DraftManifestFailureReason { unsupportedVersion, corrupt, invalidMedia }

/// A content-free persistence error safe to surface to diagnostics.
final class DraftManifestException implements Exception {
  const DraftManifestException(this.reason);

  final DraftManifestFailureReason reason;

  @override
  String toString() => 'DraftManifestException(${reason.name})';
}
