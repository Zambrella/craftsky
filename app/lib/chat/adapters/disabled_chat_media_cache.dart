import 'dart:typed_data';

import 'package:cross_cache/cross_cache.dart';
import 'package:dio/dio.dart';

/// No superclass construction: the package's web cache opens IndexedDB in its
/// constructor. Chat media never uses that cache, filesystem or network client.
final class DisabledChatMediaCache implements CrossCache {
  const new();
  @override
  BaseOptions? get options => null;
  @override
  Future<Uint8List> downloadAndSave(
    String source, {
    Map<String, dynamic>? headers,
    ProgressCallback? onReceiveProgress,
  }) => Future.error(StateError('Messaging media cache is disabled'));
  @override
  Future<Uint8List> get(String key) =>
      Future.error(StateError('Messaging media cache is disabled'));
  @override
  Future<void> set(String key, Uint8List value) =>
      Future.error(StateError('Messaging media cache is disabled'));
  @override
  Future<bool> contains(String key) => Future.value(false);
  @override
  Future<void> delete(String key) => Future.value();
  @override
  Future<void> updateKey(String key, String newKey) => Future.value();
  @override
  void dispose() {}
}
