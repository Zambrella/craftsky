/// Reload a previously loaded range atomically. Failed continuations never
/// publish a shortened list; opaque, repeated cursors cannot loop forever.
Future<({List<T> items, C? cursor})> reloadLoadedRange<T, C>({
  required int minimumItems,
  required Future<({List<T> items, C? cursor})> Function(C? cursor) fetch,
  required Object Function(T item) key,
}) async {
  final items = <T>[];
  final seen = <Object>{};
  final visited = <C>{};
  C? cursor;
  do {
    final page = await fetch(cursor);
    for (final item in page.items) {
      if (seen.add(key(item))) items.add(item);
    }
    cursor = page.cursor;
  } while (items.length < minimumItems &&
      cursor != null &&
      visited.add(cursor));
  return (items: items, cursor: cursor);
}
