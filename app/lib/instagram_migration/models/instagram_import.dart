import 'package:flutter/foundation.dart';

enum InstagramImportSourceType {
  manual,
  instagramJson,
  unknown;

  static InstagramImportSourceType fromWire(String value) => switch (value) {
    'manual' => manual,
    'instagramJson' => instagramJson,
    _ => unknown,
  };

  String get wireValue => switch (this) {
    manual => 'manual',
    instagramJson => 'instagramJson',
    unknown => throw StateError('unknown_import_source_type'),
  };
}

enum InstagramImportState {
  active,
  membershipInactive,
  expired,
  unknown;

  static InstagramImportState fromWire(String value) => switch (value) {
    'active' => active,
    'membershipInactive' => membershipInactive,
    'expired' => expired,
    _ => unknown,
  };
}

@immutable
final class InstagramImportEntry {
  const InstagramImportEntry({required this.username});

  final String username;

  Map<String, Object?> toMap() => {'username': username};

  @override
  bool operator ==(Object other) =>
      other is InstagramImportEntry && other.username == username;

  @override
  int get hashCode => username.hashCode;

  @override
  String toString() => 'InstagramImportEntry([REDACTED])';
}

final class InstagramImportParseResult {
  const InstagramImportParseResult({
    required this.entries,
    this.ignoredEntryCount = 0,
    this.duplicateEntryCount = 0,
  });

  final List<InstagramImportEntry> entries;
  final int ignoredEntryCount;
  final int duplicateEntryCount;

  @override
  String toString() => 'InstagramImportParseResult([REDACTED])';
}

final class InstagramImportRequest {
  InstagramImportRequest({
    required this.sourceType,
    required this.retainUnmatched,
    required List<InstagramImportEntry> entries,
  }) : entries = List.unmodifiable(entries);

  final InstagramImportSourceType sourceType;
  final bool retainUnmatched;
  final List<InstagramImportEntry> entries;

  Map<String, Object?> toMap() => {
    'sourceType': sourceType.wireValue,
    'retainUnmatched': retainUnmatched,
    'entries': entries.map((entry) => entry.toMap()).toList(growable: false),
  };

  @override
  String toString() => 'InstagramImportRequest([REDACTED])';
}

final class InstagramImportSummary {
  const InstagramImportSummary({
    required this.importId,
    required this.state,
    required this.sourceType,
    required this.retainUnmatched,
    required this.retentionExpiresAt,
    required this.followingCount,
    required this.createdAt,
  });

  factory InstagramImportSummary.fromMap(Map<String, dynamic> map) {
    final importId = map['importId'];
    final state = map['state'];
    final sourceType = map['sourceType'];
    final retainUnmatched = map['retainUnmatched'];
    final retentionExpiresAt = map['retentionExpiresAt'];
    final followingCount = map['followingCount'];
    final createdAt = map['createdAt'];
    if (importId is! String ||
        state is! String ||
        sourceType is! String ||
        retainUnmatched is! bool ||
        retentionExpiresAt is! String? ||
        followingCount is! int ||
        createdAt is! String) {
      throw const FormatException('invalid_instagram_import');
    }
    return InstagramImportSummary(
      importId: importId,
      state: InstagramImportState.fromWire(state),
      sourceType: InstagramImportSourceType.fromWire(sourceType),
      retainUnmatched: retainUnmatched,
      retentionExpiresAt: retentionExpiresAt == null
          ? null
          : DateTime.parse(retentionExpiresAt).toUtc(),
      followingCount: followingCount,
      createdAt: DateTime.parse(createdAt).toUtc(),
    );
  }

  final String importId;
  final InstagramImportState state;
  final InstagramImportSourceType sourceType;
  final bool retainUnmatched;
  final DateTime? retentionExpiresAt;
  final int followingCount;
  final DateTime createdAt;

  @override
  String toString() => 'InstagramImportSummary([REDACTED])';
}

final class InstagramImportCreateResult {
  const InstagramImportCreateResult({
    required this.import,
    required this.followingCount,
    required this.initialSuggestionCount,
  });

  factory InstagramImportCreateResult.fromMap(Map<String, dynamic> map) {
    final import = map['import'];
    final counts = map['counts'];
    final initialSuggestionCount = map['initialSuggestionCount'];
    if (import is! Map<String, dynamic> ||
        counts is! Map<String, dynamic> ||
        initialSuggestionCount is! int ||
        counts['followingCount'] is! int) {
      throw const FormatException('invalid_instagram_import_result');
    }
    return InstagramImportCreateResult(
      import: InstagramImportSummary.fromMap(import),
      followingCount: counts['followingCount'] as int,
      initialSuggestionCount: initialSuggestionCount,
    );
  }

  final InstagramImportSummary import;
  final int followingCount;
  final int initialSuggestionCount;

  @override
  String toString() => 'InstagramImportCreateResult([REDACTED])';
}

final class InstagramImportPage {
  InstagramImportPage({
    required List<InstagramImportSummary> items,
    required this.cursor,
  }) : items = List.unmodifiable(items);

  factory InstagramImportPage.fromMap(Map<String, dynamic> map) {
    final items = map['items'];
    final cursor = map['cursor'];
    if (items is! List<dynamic> || cursor is! String?) {
      throw const FormatException('invalid_instagram_import_page');
    }
    return InstagramImportPage(
      items: items
          .map((item) {
            if (item is! Map<String, dynamic>) {
              throw const FormatException('invalid_instagram_import_page_item');
            }
            return InstagramImportSummary.fromMap(item);
          })
          .toList(growable: false),
      cursor: cursor,
    );
  }

  final List<InstagramImportSummary> items;
  final String? cursor;

  @override
  String toString() => 'InstagramImportPage([REDACTED])';
}

final class InstagramImportPatch {
  const InstagramImportPatch({this.retainUnmatched, this.reactivate})
    : assert(
        retainUnmatched != null || reactivate != null,
        'At least one import setting must be supplied.',
      );

  final bool? retainUnmatched;
  final bool? reactivate;

  Map<String, Object?> toMap() => {
    if (retainUnmatched != null) 'retainUnmatched': retainUnmatched,
    if (reactivate != null) 'reactivate': reactivate,
  };

  @override
  String toString() => 'InstagramImportPatch([REDACTED])';
}
