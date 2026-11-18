import 'dart:convert';

import 'package:craftsky_app/feed/models/instagram_post_import_source.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';

/// Exact AppView request kept with an account's local Instagram draft until
/// there is a definite publication outcome.
final class InstagramImportPublication {
  factory({
    required String operationKey,
    required Map<String, Object?> request,
  }) {
    final source = request['instagramImport'];
    final embed = request['embed'];
    if (!isCanonicalPdsMutationOperationKey(operationKey) ||
        source is! Map<String, Object?> ||
        request.containsKey('reply') ||
        (embed is Map<String, Object?> && embed.containsKey('quote'))) {
      throw const FormatException('Invalid pending Instagram publication');
    }
    return InstagramImportPublication._(
      operationKey,
      jsonEncode(request),
      InstagramPostImportSource.fromMap(source),
    );
  }

  const new _(this.operationKey, this.encodedRequest, this.source);

  factory fromMap(Map<String, Object?> map) {
    final key = map['operationKey'];
    final request = map['request'];
    if (key is! String || request is! Map<String, Object?>) {
      throw const FormatException('Invalid pending Instagram publication');
    }
    return InstagramImportPublication(operationKey: key, request: request);
  }

  final String operationKey;
  final String encodedRequest;
  final InstagramPostImportSource source;

  Map<String, Object?> get request =>
      jsonDecode(encodedRequest) as Map<String, Object?>;

  bool matchesSource(InstagramPostImportSource? expected) =>
      expected?.sourceKey == source.sourceKey &&
      expected?.createdAt == source.createdAt;

  Map<String, Object?> toMap() => {
    'operationKey': operationKey,
    'request': request,
  };

  @override
  String toString() => 'InstagramImportPublication(<redacted>)';
}
