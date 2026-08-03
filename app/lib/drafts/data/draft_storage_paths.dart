import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:path/path.dart' as p;

final class DraftPathException implements Exception {
  const DraftPathException();

  @override
  String toString() => 'DraftPathException(invalidComponent)';
}

/// Owns all path construction for one account's private draft namespace.
final class DraftStoragePaths {
  DraftStoragePaths({required String documentsRoot, required AccountKey owner})
    : accountRoot = p.join(
        p.normalize(documentsRoot),
        'CraftSky',
        'drafts',
        'v1',
        base64Url.encode(utf8.encode(owner.did.value)).replaceAll('=', ''),
      );

  final String accountRoot;

  String draftDirectory(String draftId) {
    _requireUuid(draftId);
    return p.join(accountRoot, draftId);
  }

  String manifestPath(String draftId) =>
      p.join(draftDirectory(draftId), 'manifest.json');

  String mediaDirectory(String draftId) =>
      p.join(draftDirectory(draftId), 'media');

  String mediaFilePath(String draftId, String storageFileName) {
    _requireLeaf(storageFileName);
    return p.join(mediaDirectory(draftId), storageFileName);
  }

  String pendingManifestPath(String draftId, String operationId) {
    _requireUuid(operationId);
    return p.join(
      draftDirectory(draftId),
      '.pending-$operationId-manifest.json',
    );
  }

  @override
  String toString() => 'DraftStoragePaths(<redacted>)';
}

final _uuidPattern = RegExp(
  r'^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$',
  caseSensitive: false,
);

void _requireUuid(String value) {
  if (!_uuidPattern.hasMatch(value)) throw const DraftPathException();
}

void _requireLeaf(String value) {
  if (value.isEmpty ||
      p.isAbsolute(value) ||
      p.basename(value) != value ||
      value.contains('/') ||
      value.contains(r'\') ||
      value.contains('..')) {
    throw const DraftPathException();
  }
}
