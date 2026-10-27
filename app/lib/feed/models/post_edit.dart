import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

final class PostEditSeed {
  new({
    required this.post,
    required this.baseCid,
    required this.recordRevision,
    required Map<String, Object?> editable,
  }) : editable = _freezeEditMap(editable) {
    if (baseCid != post.cid ||
        editable['text'] is! String ||
        editable['sponsored'] is! bool) {
      throw const FormatException('Incomplete post edit seed');
    }
  }

  factory fromMap(Map<String, dynamic> value) => PostEditSeed(
    post: PostMapper.fromMap(value['post'] as Map<String, dynamic>),
    baseCid: Cid.parse(value['baseCid'] as String),
    recordRevision: value['recordRevision'] as String,
    editable: Map<String, Object?>.from(value['editable'] as Map),
  );

  final Post post;
  final Cid baseCid;
  final String recordRevision;
  final Map<String, Object?> editable;

  @override
  String toString() => 'PostEditSeed(<redacted>)';
}

/// Omission leaves a field unchanged; null clears an optional field.
/// Only supported leaves belong here. Unknown authored data stays on AppView.
final class PostEditChanges {
  new(Map<String, Object?> fields) : _fields = _freezeEditMap(fields) {
    const supported = {
      'text',
      'sponsored',
      'langs',
      'facets',
      'images',
      'embed',
      'project',
    };
    if (_fields.keys.any((key) => !supported.contains(key))) {
      throw const FormatException('Unsupported post edit field');
    }
  }

  final Map<String, Object?> _fields;
  Map<String, Object?> toMap() => _fields;
  bool get isEmpty => _fields.isEmpty;

  @override
  String toString() => 'PostEditChanges(<redacted>)';
}

final class PostEditResult {
  const new({
    required this.acceptedCid,
    required this.changed,
    required this.projectionPending,
    this.post,
    this.editedAt,
    this.editTimeSource,
    this.recordRevision,
    this.observationCid,
    this.observationRevision,
  });

  factory fromMap(Map<String, dynamic> value) {
    final result = PostEditResult(
      acceptedCid: Cid.parse(value['acceptedCid'] as String),
      changed: value['changed'] as bool,
      projectionPending: value['projectionPending'] as bool,
      post: value['post'] == null
          ? null
          : PostMapper.fromMap(value['post'] as Map<String, dynamic>),
      editedAt: value['editedAt'] == null
          ? null
          : DateTime.parse(value['editedAt'] as String).toUtc(),
      editTimeSource: value['editTimeSource'] == null
          ? null
          : PostEditTimeSource.values.firstWhere(
              (source) => source.name == value['editTimeSource'],
              orElse: () => PostEditTimeSource.unknown,
            ),
      recordRevision: value['recordRevision'] as String?,
      observationCid: value['observationCid'] == null
          ? null
          : Cid.parse(value['observationCid'] as String),
      observationRevision: value['observationRevision'] as String?,
    );
    if ((result.changed && result.editedAt == null) ||
        (result.post == null && !result.projectionPending) ||
        (result.post != null && result.post!.cid != result.acceptedCid)) {
      throw const FormatException('Incoherent accepted edit response');
    }
    return result;
  }

  final Cid acceptedCid;
  final bool changed;
  final bool projectionPending;
  final Post? post;
  final DateTime? editedAt;
  final PostEditTimeSource? editTimeSource;

  /// Exact accepted/verified record revision, when available.
  final String? recordRevision;

  /// A recovery observation bound, not an exact write revision.
  final Cid? observationCid;
  final String? observationRevision;

  @override
  String toString() => 'PostEditResult(<redacted>)';
}

Map<String, Object?> _freezeEditMap(Map<String, Object?> value) =>
    Map.unmodifiable(
      value.map((key, item) => MapEntry(key, _freezeEditValue(item))),
    );

Object? _freezeEditValue(Object? value) => switch (value) {
  final Map<String, Object?> map => _freezeEditMap(map),
  final List<Object?> list => List<Object?>.unmodifiable(
    list.map(_freezeEditValue),
  ),
  null || String() || bool() || num() => value,
  _ => throw const FormatException('Invalid post edit value'),
};
