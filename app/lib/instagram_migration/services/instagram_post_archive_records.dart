part of 'instagram_post_archive.dart';

final _metadataPath = RegExp(
  r'^(?:([^./\\][^/\\]*)/)?your_instagram_activity/media/'
  r'(posts|reels|videos)(_\d+)?\.json$',
);
final _mediaPath = RegExp(
  r'^media/(?:(?:posts|reels|videos)/(?:\d{6}/)?|other/)[^/?#]+$',
);
final _drivePath = RegExp('^[a-zA-Z]:');
final _instagramLaunch = DateTime.utc(2010, 10, 6);
final _garbledCaption = RegExp('(?:Ã.|Â.|â.|ð.|�)');

bool _isSafePath(String path) =>
    !_drivePath.hasMatch(path) &&
    !path.contains(r'\') &&
    !path.contains('\u0000') &&
    path
        .split('/')
        .every(
          (part) => part.isNotEmpty && part != '.' && part != '..',
        );

({String wrapper, bool detailed})? _matchMetadataPath(String path) {
  final match = _metadataPath.firstMatch(path);
  if (match == null || match.end != path.length) return null;
  final wrapper = match.group(1);
  return (
    wrapper: wrapper == null ? '' : '$wrapper/',
    detailed: match.group(2) == 'posts' && match.group(3) == null,
  );
}

typedef _SourceMedia = ({String path, InstagramArchiveMediaKind kind});
typedef _SourcePost = ({
  num timestamp,
  String caption,
  List<_SourceMedia> media,
});
typedef _Record = Map<String, Object?>;

({List<_SourcePost> posts, int skipped, int recognized}) _parseRecords(
  Object? input, {
  required bool detailed,
}) {
  var value = input;
  if (!detailed && value is _Record) {
    if (value['ig_reels_media'] is List<Object?>) {
      value = value['ig_reels_media'];
    } else if (value['ig_videos'] is List<Object?>) {
      value = value['ig_videos'];
    }
  }
  final records = switch (value) {
    final List<Object?> list => list.whereType<_Record>(),
    final _Record record => [record],
    _ => const <_Record>[],
  };
  final keys = detailed
      ? const ['fbid', 'label_values', 'media', 'timestamp']
      : const ['creation_timestamp', 'media', 'title'];
  final posts = <_SourcePost>[];
  var skipped = 0;
  var recognized = 0;
  for (final record in records) {
    if (!keys.any(record.containsKey)) continue;
    if (++recognized > _maxPosts) {
      throw const InstagramPostArchiveException(
        InstagramPostArchiveError.archiveTooLarge,
      );
    }
    final post = detailed ? _detailedRecord(record) : _legacyRecord(record);
    if (post == null) {
      skipped++;
    } else {
      posts.add(post);
    }
  }
  return (posts: posts, skipped: skipped, recognized: recognized);
}

_SourcePost? _legacyRecord(_Record record) {
  final items = record['media'];
  if (items is! List<Object?>) return null;
  return _post(
    items,
    _milliseconds(record['creation_timestamp']),
    (titles) => _firstCaption([record['title'], ...titles]),
  );
}

_SourcePost? _detailedRecord(_Record record) {
  final labels = record['label_values'];
  if (labels is! List<Object?>) return null;
  final direct = _withLabels(labels).toList();
  final nested = <_Record>[];
  var nestedGroups = false;
  for (final container in labels.whereType<_Record>()) {
    if (container.containsKey('label') && container['label'] != 'Metadata') {
      continue;
    }
    final candidates = container['dict'];
    if (candidates is! List<Object?>) continue;
    for (final candidate in candidates.whereType<_Record>()) {
      final entries = candidate['dict'];
      if (entries is! List<Object?>) continue;
      final group = _withLabels(entries).toList();
      if (group.isEmpty) continue;
      nestedGroups = true;
      nested.addAll(group);
    }
  }
  if (!nestedGroups &&
      !direct.any(
        (entry) => const {
          'Caption',
          'Creation time',
          'Draft',
          'Media',
          'Published',
        }.contains(entry['label']),
      )) {
    return null;
  }
  final allLabels = [...direct, ...nested];
  final draft = _label(allLabels, 'Draft');
  final published = _label(allLabels, 'Published');
  if ((draft != null && _parseBoolean(draft['value']) != false) ||
      (published != null && _parseBoolean(published['value']) != true)) {
    return null;
  }
  final nestedMedia = nested
      .where((entry) => entry['label'] == 'Media')
      .toList();
  final mediaLabels = nestedMedia.isEmpty
      ? direct.where((entry) => entry['label'] == 'Media')
      : nestedMedia.reversed;
  final labelledTimestamps = allLabels
      .where((entry) => entry['label'] == 'Creation time')
      .map((entry) => _milliseconds(entry['timestamp_value']))
      .nonNulls;
  final firstLabelled = _minimum(labelledTimestamps);
  return _post(
    [
      for (final label in mediaLabels)
        if (label['media'] case final List<Object?> items) ...items,
    ],
    firstLabelled ?? _milliseconds(record['timestamp']),
    (_) => _firstCaption([_label(direct, 'Caption')?['value']]),
  );
}

Iterable<_Record> _withLabels(List<Object?> values) =>
    values.whereType<_Record>().where((entry) => entry['label'] is String);

_Record? _label(List<_Record> entries, String label) {
  for (final entry in entries) {
    if (entry['label'] == label) return entry;
  }
  return null;
}

bool? _parseBoolean(Object? value) {
  if (value is bool) return value;
  if (value is! String) return null;
  return switch (value.trim().toLowerCase()) {
    'true' => true,
    'false' => false,
    _ => null,
  };
}

_SourcePost? _post(
  List<Object?> items,
  num? fallbackTime,
  String Function(List<String>) caption,
) {
  final media = <_SourceMedia>[];
  final times = <num>[];
  final titles = <String>[];
  for (final item in items) {
    if (item is! _Record) return null;
    final uri = item['uri'];
    if (uri is! String || !_isSafePath(uri)) return null;
    final match = _mediaPath.firstMatch(uri);
    if (match == null || match.end != uri.length) return null;
    media.add((path: uri, kind: _mediaKind(uri)));
    final time = _milliseconds(item['creation_timestamp']);
    if (time != null) times.add(time);
    final title = item['title'];
    if (title is String && title.isNotEmpty) titles.add(title);
  }
  final timestamp = _minimum(times) ?? fallbackTime;
  if (media.isEmpty || timestamp == null) return null;
  return (timestamp: timestamp, caption: caption(titles), media: media);
}

num? _minimum(Iterable<num> values) {
  num? result;
  for (final value in values) {
    if (result == null || value < result) result = value;
  }
  return result;
}

num? _milliseconds(Object? value) {
  if (value is! num || !value.isFinite || value < 0) return null;
  return value < 100000000000 ? value * 1000 : value;
}

DateTime? _normalizeDate(num milliseconds, DateTime latest) {
  if (milliseconds < _instagramLaunch.millisecondsSinceEpoch ||
      milliseconds > latest.millisecondsSinceEpoch) {
    return null;
  }
  return DateTime.fromMillisecondsSinceEpoch(milliseconds.toInt(), isUtc: true);
}

InstagramArchiveMediaKind _mediaKind(String path) =>
    switch (path.split('.').last.toLowerCase()) {
      'jpg' || 'jpeg' || 'png' || 'webp' => InstagramArchiveMediaKind.photo,
      'mp4' || 'mov' || 'm4v' => InstagramArchiveMediaKind.video,
      _ => InstagramArchiveMediaKind.unsupported,
    };

String _kindName(InstagramArchiveMediaKind kind) =>
    kind == InstagramArchiveMediaKind.photo ? 'image' : kind.name;

String _firstCaption(Iterable<Object?> values) {
  for (final value in values) {
    if (value is String && value.isNotEmpty) return value;
  }
  return '';
}

String _decodeCaption(String value) {
  if (!_garbledCaption.hasMatch(value) ||
      value.codeUnits.any((point) => point > 255)) {
    return value;
  }
  try {
    return utf8.decode(value.codeUnits);
  } on FormatException {
    return value;
  }
}

/// Keep the browser's v1 identity exactly, including NFC paths, ISO UTC
/// millisecond formatting and media order. Captions/selection are editable.
String _fingerprint(DateTime date, _SourcePost post) {
  final identity = [
    'instagram-import-post-v1',
    date.toIso8601String(),
    for (final media in post.media)
      '${_kindName(media.kind)}:${unorm.nfc(media.path)}',
  ].join('\u0000');
  return sha256.convert(utf8.encode(identity)).toString().substring(0, 32);
}

/// Some exports describe the same files in both metadata formats. Keep one
/// identical record, skip conflicting groups sharing a media reference.
({List<_SourcePost> posts, int skipped}) _mergeRecords(
  List<_SourcePost> input,
) {
  final parents = List.generate(input.length, (index) => index);
  int find(int index) {
    var root = index;
    while (parents[root] != root) {
      root = parents[root];
    }
    var cursor = index;
    while (parents[cursor] != root) {
      final next = parents[cursor];
      parents[cursor] = root;
      cursor = next;
    }
    return root;
  }

  final seen = <String, int>{};
  for (var index = 0; index < input.length; index++) {
    for (final media in input[index].media) {
      final key = unorm.nfc(media.path);
      final other = seen.putIfAbsent(key, () => index);
      final firstRoot = find(index);
      final otherRoot = find(other);
      if (firstRoot != otherRoot) parents[otherRoot] = firstRoot;
    }
  }
  final groups = <int, List<_SourcePost>>{};
  for (var index = 0; index < input.length; index++) {
    (groups[find(index)] ??= []).add(input[index]);
  }

  final accepted = <({String key, _SourcePost post})>[];
  var skipped = 0;
  for (final candidates in groups.values) {
    final first = candidates.first;
    final key = _postKey(first);
    if (candidates.skip(1).any((candidate) => _postKey(candidate) != key)) {
      skipped++;
    } else {
      accepted.add((key: key, post: first));
    }
  }
  accepted.sort((a, b) {
    final time = a.post.timestamp.compareTo(b.post.timestamp);
    return time == 0 ? a.key.compareTo(b.key) : time;
  });
  return (
    posts: accepted.map((entry) => entry.post).toList(),
    skipped: skipped,
  );
}

String _postKey(_SourcePost post) => jsonEncode([
  if (post.timestamp == post.timestamp.toInt())
    post.timestamp.toInt()
  else
    post.timestamp,
  post.caption,
  for (final media in post.media)
    [_kindName(media.kind), unorm.nfc(media.path)],
]);
