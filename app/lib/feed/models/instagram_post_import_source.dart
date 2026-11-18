/// Private import identity sent only to AppView during publication.
final class InstagramPostImportSource {
  new({required this.sourceKey, required DateTime createdAt})
    : createdAt = createdAt.toUtc() {
    if (!_sourceKeyPattern.hasMatch(sourceKey) ||
        this.createdAt.isBefore(_earliestDate)) {
      throw const FormatException('Invalid Instagram post import source');
    }
  }

  factory fromMap(Map<String, Object?> map) {
    final sourceKey = map['sourceKey'];
    final createdAt = map['createdAt'];
    final parsedDate = createdAt is String
        ? DateTime.tryParse(createdAt)
        : null;
    if (sourceKey is! String || parsedDate == null || !parsedDate.isUtc) {
      throw const FormatException('Invalid Instagram post import source');
    }
    return InstagramPostImportSource(
      sourceKey: sourceKey,
      createdAt: parsedDate,
    );
  }

  static final _sourceKeyPattern = RegExp(r'^[0-9a-f]{32}$');
  static final _earliestDate = DateTime.utc(2010);

  final String sourceKey;
  final DateTime createdAt;

  Map<String, Object?> toMap() => {
    'sourceKey': sourceKey,
    'createdAt': createdAt.toIso8601String(),
  };

  @override
  String toString() => 'InstagramPostImportSource(<redacted>)';
}
