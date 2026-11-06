final class LocalVideoPreviewResource {
  new(this.uri, this._release);
  final Uri uri;
  final Future<void> Function() _release;
  Future<void>? _disposal;
  Future<void> dispose() => _disposal ??= _release();
}
