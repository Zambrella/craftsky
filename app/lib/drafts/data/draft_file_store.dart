import 'dart:io';
import 'dart:typed_data';

enum DraftFileStoreFailureReason { notFound, accessDenied, ioFailure }

final class DraftFileStoreException implements Exception {
  const DraftFileStoreException(this.reason);

  final DraftFileStoreFailureReason reason;

  @override
  String toString() => 'DraftFileStoreException(${reason.name})';
}

abstract interface class DraftFileStore {
  Future<void> ensureDirectory(String path);

  Future<bool> fileExists(String path);

  Future<bool> directoryExists(String path);

  Future<Uint8List> readBytes(String path);

  Future<void> writeBytesFlushed(String path, Uint8List bytes);

  Future<void> atomicReplace({
    required String sourcePath,
    required String targetPath,
  });

  Future<List<String>> listChildDirectories(String path);

  Future<List<String>> listChildFiles(String path);

  Future<void> deleteFile(String path);

  Future<void> deleteDirectory(String path);

  Future<void> moveDirectory({
    required String sourcePath,
    required String targetPath,
  });
}

final class IoDraftFileStore implements DraftFileStore {
  @override
  Future<void> ensureDirectory(String path) => _guard(() async {
    await Directory(path).create(recursive: true);
  });

  @override
  // Async filesystem access is intentional to keep draft I/O off the UI path.
  // ignore: avoid_slow_async_io
  Future<bool> fileExists(String path) => _guard(() => File(path).exists());

  @override
  Future<bool> directoryExists(String path) => _guard(
    // Async filesystem access is intentional to keep draft I/O off the UI path.
    // ignore: avoid_slow_async_io
    () => Directory(path).exists(),
  );

  @override
  Future<Uint8List> readBytes(String path) =>
      _guard(() async => File(path).readAsBytes());

  @override
  Future<void> writeBytesFlushed(String path, Uint8List bytes) =>
      _guard(() async {
        await File(path).writeAsBytes(bytes, flush: true);
      });

  @override
  Future<void> atomicReplace({
    required String sourcePath,
    required String targetPath,
  }) => _guard(() async {
    await File(sourcePath).rename(targetPath);
  });

  @override
  Future<List<String>> listChildDirectories(String path) => _guard(() async {
    final entries = await Directory(path).list(followLinks: false).toList();
    return [
      for (final entry in entries)
        if (entry is Directory) entry.path,
    ];
  });

  @override
  Future<List<String>> listChildFiles(String path) => _guard(() async {
    final entries = await Directory(path).list(followLinks: false).toList();
    return [
      for (final entry in entries)
        if (entry is File) entry.path,
    ];
  });

  @override
  Future<void> deleteFile(String path) => _guard(() async {
    final file = File(path);
    // Async filesystem access is intentional to keep draft I/O off the UI path.
    // ignore: avoid_slow_async_io
    if (await file.exists()) await file.delete();
  });

  @override
  Future<void> deleteDirectory(String path) => _guard(() async {
    final directory = Directory(path);
    // Async filesystem access is intentional to keep draft I/O off the UI path.
    // ignore: avoid_slow_async_io
    if (await directory.exists()) await directory.delete(recursive: true);
  });

  @override
  Future<void> moveDirectory({
    required String sourcePath,
    required String targetPath,
  }) => _guard(() async {
    await Directory(sourcePath).rename(targetPath);
  });
}

Future<T> _guard<T>(Future<T> Function() operation) async {
  try {
    return await operation();
  } on PathNotFoundException {
    throw const DraftFileStoreException(DraftFileStoreFailureReason.notFound);
  } on PathAccessException {
    throw const DraftFileStoreException(
      DraftFileStoreFailureReason.accessDenied,
    );
  } on FileSystemException {
    throw const DraftFileStoreException(DraftFileStoreFailureReason.ioFailure);
  }
}
