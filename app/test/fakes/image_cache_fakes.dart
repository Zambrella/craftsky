import 'dart:async';

import 'package:flutter_cache_manager/flutter_cache_manager.dart';
import 'package:flutter_test/flutter_test.dart';

/// Recording fake `BaseCacheManager` for tests. Any method not explicitly
/// overridden throws `UnimplementedError` (via [Fake]) so unintended
/// usages are loud, not silent.
///
/// Override `nextStream` per-test to control what `getFileStream` emits;
/// override `throwOnEmptyCache` to make `emptyCache` fail.
class FakeBaseCacheManager extends Fake implements BaseCacheManager {
  int emptyCacheCalls = 0;
  Object? throwOnEmptyCache;

  /// Stream returned by [getFileStream]. Default is an empty stream that
  /// stays open forever — `CachedNetworkImage` will sit on its
  /// placeholder.
  Stream<FileResponse> Function(String url)? nextStream;

  @override
  Future<void> emptyCache() async {
    emptyCacheCalls++;
    final err = throwOnEmptyCache;
    if (err != null) {
      Error.throwWithStackTrace(err, StackTrace.current);
    }
  }

  @override
  Stream<FileResponse> getFileStream(
    String url, {
    String? key,
    Map<String, String>? headers,
    bool? withProgress,
  }) {
    final builder = nextStream;
    if (builder != null) {
      return builder(url);
    }
    // Default: a stream that emits nothing and never closes.
    return StreamController<FileResponse>().stream;
  }
}

/// Convenience builder: a stream that immediately errors. Use as
/// `fake.nextStream = (_) => erroringStream();` to drive `errorWidget`.
Stream<FileResponse> erroringStream([Object error = 'fake-cache-error']) {
  final controller = StreamController<FileResponse>()..addError(error);
  unawaited(controller.close());
  return controller.stream;
}
