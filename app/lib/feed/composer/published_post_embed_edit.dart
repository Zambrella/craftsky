/// Editor state for supported published media. Untouched references stay on
/// the server and never need to be uploaded again.
final class PublishedPostEmbedEdit {
  new(Map<String, Object?>? original)
    : _original = original == null ? null : Map.unmodifiable(original),
      _external = Map<String, Object?>.from(
        original?['external'] as Map? ?? const {},
      ),
      videoAlt = original?['alt'] as String? ?? '';

  final Map<String, Object?>? _original;
  String videoAlt;
  bool _removed = false;
  final Map<String, Object?> _external;

  bool get hasExternal =>
      !_removed && _original?[r'$type'] == 'app.bsky.embed.external';
  String get externalUri => _external['uri'] as String? ?? '';
  String get externalTitle => _external['title'] as String? ?? '';
  String get externalDescription => _external['description'] as String? ?? '';
  bool get hasExternalThumbnail => _external['thumb'] != null;

  bool get hasRetainedVideo =>
      !_removed && _original?[r'$type'] == 'app.bsky.embed.video';

  void remove() {
    if (_original?[r'$type'] != 'app.bsky.embed.video' && !hasExternal) {
      throw const FormatException('Unsupported embed change');
    }
    _removed = true;
  }

  Map<String, Object?> get changes {
    if (_removed) return {'embed': null};
    if (hasRetainedVideo && videoAlt != (_original?['alt'] ?? '')) {
      return {
        'embed': {'alt': videoAlt},
      };
    }
    return const {};
  }

  @override
  String toString() => 'PublishedPostEmbedEdit(<redacted>)';
}
