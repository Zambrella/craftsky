abstract interface class CaptionUriResource {
  Uri get uri;
  Future<void> dispose();
}

final class ManagedCaptionUriResource implements CaptionUriResource {
  new(this.uri, this._dispose);

  @override
  final Uri uri;
  final Future<void> Function() _dispose;

  @override
  Future<void> dispose() => _dispose();
}
