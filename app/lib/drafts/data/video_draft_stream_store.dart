import 'dart:io';

import 'package:crypto/crypto.dart';

final class VideoDraftStreamResult {
  const VideoDraftStreamResult({
    required this.byteLength,
    required this.sha256,
  });

  final int byteLength;
  final String sha256;
}

final class VideoDraftStreamException implements Exception {
  const VideoDraftStreamException();
}

Future<VideoDraftStreamResult> writeVideoDraftStream({
  required Stream<List<int>> source,
  required String targetPath,
  required int maximumBytes,
  void Function(int bytesWritten)? onChunkWritten,
}) async {
  if (maximumBytes < 0) throw const VideoDraftStreamException();
  final target = File(targetPath);
  RandomAccessFile? output;
  final digestOutput = _DigestOutput();
  final digestInput = sha256.startChunkedConversion(digestOutput);
  var byteLength = 0;
  var digestClosed = false;
  var completed = false;
  try {
    output = await target.open(mode: FileMode.writeOnly);
    await for (final chunk in source) {
      if (chunk.isEmpty) continue;
      if (chunk.length > maximumBytes - byteLength) {
        throw const VideoDraftStreamException();
      }
      await output.writeFrom(chunk);
      digestInput.add(chunk);
      byteLength += chunk.length;
      onChunkWritten?.call(byteLength);
    }
    digestInput.close();
    digestClosed = true;
    await output.flush();
    await output.close();
    output = null;
    completed = true;
    return VideoDraftStreamResult(
      byteLength: byteLength,
      sha256: digestOutput.value.toString(),
    );
  } on VideoDraftStreamException {
    rethrow;
  } on Object {
    throw const VideoDraftStreamException();
  } finally {
    await output?.close();
    if (!digestClosed) digestInput.close();
    if (!completed) {
      try {
        // Async filesystem access keeps draft cleanup off the UI path.
        // ignore: avoid_slow_async_io
        if (await target.exists()) await target.delete();
      } on Object {
        // A repository-level reconciliation pass can retry failed cleanup.
      }
    }
  }
}

final class _DigestOutput implements Sink<Digest> {
  Digest? _value;

  Digest get value => _value ?? (throw StateError('Digest is not complete'));
  @override
  void add(Digest data) => _value = data;

  @override
  void close() {}
}
