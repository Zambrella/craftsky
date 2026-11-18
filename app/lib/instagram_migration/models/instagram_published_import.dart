import 'package:at_primitives/at_uri.dart' show ensureValidAtUri;
import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

/// Private lookup result for an Instagram source the account already published.
final class InstagramPublishedImport {
  const new({
    required this.sourceKey,
    required this.uri,
    required this.isProject,
  });

  factory fromMap(Map<String, dynamic> map) {
    final sourceKey = map['sourceKey'];
    final uri = map['uri'];
    final kind = map['kind'];
    if (sourceKey is! String ||
        !isSourceKey(sourceKey) ||
        uri is! String ||
        !uri.startsWith('at://did:') ||
        uri.contains('#') ||
        (kind != 'standard' && kind != 'project')) {
      throw const FormatException('invalid_instagram_published_import');
    }
    ensureValidAtUri(uri);
    final postUri = AtUri.parse(uri);
    if (parseCraftskyPostUri(postUri) == null) {
      throw const FormatException('invalid_instagram_import_post_uri');
    }
    return InstagramPublishedImport(
      sourceKey: sourceKey,
      uri: postUri,
      isProject: kind == 'project',
    );
  }

  static final _sourceKeyPattern = RegExp(r'^[0-9a-f]{32}$');
  static bool isSourceKey(String value) =>
      value.length == 32 && _sourceKeyPattern.hasMatch(value);

  final String sourceKey;
  final AtUri uri;
  final bool isProject;

  @override
  String toString() => 'InstagramPublishedImport([REDACTED])';
}
