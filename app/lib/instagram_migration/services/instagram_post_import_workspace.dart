import 'dart:io';

import 'package:flutter/services.dart';
import 'package:path/path.dart' as path;
import 'package:path_provider/path_provider.dart';

final instagramPostImportWorkspace = InstagramPostImportWorkspace(
  cacheDirectory: getTemporaryDirectory,
  legacyMediaDirectory: Directory.systemTemp,
);

Future<void> recoverInstagramPostImportWorkspace() async {
  if (!Platform.isAndroid && !Platform.isIOS) return;
  try {
    await instagramPostImportWorkspace.prepare();
  } on FileSystemException {
    // Cache cleanup must not interrupt startup. Picking can retry it later.
  } on MissingPlatformDirectoryException {
    // The platform cache is unavailable.
  } on PlatformException {
    // A native cache lookup can fail during startup.
  } on MissingPluginException {
    // The host may not provide a cache-directory plugin.
  }
}

/// Owns one app-private workspace. Initialization removes scratch files from
/// an interrupted process before any review in the current process can use it.
final class InstagramPostImportWorkspace {
  new({required this.cacheDirectory, required this.legacyMediaDirectory});

  final Future<Directory> Function() cacheDirectory;
  final Directory legacyMediaDirectory;
  Future<Directory>? _initialization;

  static final _legacyArchiveName = RegExp(
    r'^craftsky-instagram-posts-[a-zA-Z0-9]+$',
  );
  static final _legacyMediaName = RegExp(r'^craftsky-instagram-[a-zA-Z0-9]+$');

  Future<Directory> prepare() async {
    final initialization = _initialization ??= _prepare();
    try {
      return await initialization;
    } on Object {
      if (identical(_initialization, initialization)) _initialization = null;
      rethrow;
    }
  }

  Future<Directory> createReviewDirectory() async {
    final root = await prepare();
    await root.create(recursive: true);
    return await root.createTemp('review-');
  }

  Future<Directory> _prepare() async {
    final cache = await cacheDirectory();
    final root = Directory(
      path.join(cache.path, 'craftsky-instagram-post-import'),
    );
    await _delete(root);
    await _removeLegacyDirectories(cache, _legacyArchiveName);
    await _removeLegacyDirectories(legacyMediaDirectory, _legacyMediaName);
    return root;
  }

  Future<void> _removeLegacyDirectories(Directory parent, RegExp names) async {
    try {
      await for (final entry in parent.list(followLinks: false)) {
        if (entry is Directory && names.hasMatch(path.basename(entry.path))) {
          await _delete(entry);
        }
      }
    } on FileSystemException {
      // Cache directories can be evicted or unavailable.
    }
  }

  Future<void> _delete(Directory directory) async {
    try {
      await directory.delete(recursive: true);
    } on FileSystemException {
      // Another cache cleanup or the OS may have removed it already.
    }
  }
}
