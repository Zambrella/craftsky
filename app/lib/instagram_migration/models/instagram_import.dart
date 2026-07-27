import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';

part 'instagram_import.mapper.dart';

const int _instagramImportValueMethods =
    GenerateMethods.copy | GenerateMethods.equals;
const int _instagramImportDecodeMethods =
    GenerateMethods.decode | _instagramImportValueMethods;

@MappableEnum(defaultValue: InstagramImportSourceType.unknown)
enum InstagramImportSourceType {
  manual,
  instagramJson,
  unknown;

  static InstagramImportSourceType fromWire(String value) =>
      InstagramImportSourceTypeMapper.fromValue(value);

  String get wireValue {
    if (this == unknown) throw StateError('unknown_import_source_type');
    return toValue();
  }
}

@MappableEnum(defaultValue: InstagramImportState.unknown)
enum InstagramImportState {
  active,
  membershipInactive,
  unknown;

  static InstagramImportState fromWire(String value) =>
      InstagramImportStateMapper.fromValue(value);
}

@immutable
@MappableClass(
  generateMethods:
      GenerateMethods.encode | GenerateMethods.copy | GenerateMethods.equals,
)
final class InstagramImportEntry with InstagramImportEntryMappable {
  const InstagramImportEntry({required this.username});

  final String username;

  @override
  String toString() => 'InstagramImportEntry([REDACTED])';
}

@MappableClass(generateMethods: _instagramImportValueMethods)
final class InstagramImportParseResult with InstagramImportParseResultMappable {
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

@MappableClass(generateMethods: _instagramImportValueMethods)
final class InstagramImportRequest with InstagramImportRequestMappable {
  InstagramImportRequest({
    required this.sourceType,
    required List<InstagramImportEntry> entries,
  }) : entries = List.unmodifiable(entries);

  final InstagramImportSourceType sourceType;
  final List<InstagramImportEntry> entries;

  Map<String, Object?> toMap() => {
    'sourceType': sourceType.wireValue,
    'entries': entries.map((entry) => entry.toMap()).toList(growable: false),
  };

  @override
  String toString() => 'InstagramImportRequest([REDACTED])';
}

@MappableClass(generateMethods: _instagramImportDecodeMethods)
final class InstagramImportSummary with InstagramImportSummaryMappable {
  const InstagramImportSummary({
    required this.importId,
    required this.state,
    required this.sourceType,
    required this.followingCount,
    required this.createdAt,
  });

  factory InstagramImportSummary.fromMap(Map<String, dynamic> map) =>
      InstagramImportSummaryMapper.fromMap(map);

  final String importId;
  final InstagramImportState state;
  final InstagramImportSourceType sourceType;
  final int followingCount;
  final DateTime createdAt;

  @override
  String toString() => 'InstagramImportSummary([REDACTED])';
}

@MappableClass(generateMethods: _instagramImportValueMethods)
final class InstagramImportCreateResult
    with InstagramImportCreateResultMappable {
  const InstagramImportCreateResult({
    required this.import,
    required this.followingCount,
  });

  factory InstagramImportCreateResult.fromMap(Map<String, dynamic> map) {
    final import = map['import'];
    final counts = map['counts'];
    if (import is! Map<String, dynamic> ||
        counts is! Map<String, dynamic> ||
        counts['followingCount'] is! int) {
      throw const FormatException('invalid_instagram_import_result');
    }
    return InstagramImportCreateResult(
      import: InstagramImportSummary.fromMap(import),
      followingCount: counts['followingCount'] as int,
    );
  }

  final InstagramImportSummary import;
  final int followingCount;

  @override
  String toString() => 'InstagramImportCreateResult([REDACTED])';
}

@MappableClass(generateMethods: _instagramImportDecodeMethods)
final class InstagramImportPage with InstagramImportPageMappable {
  InstagramImportPage({
    required List<InstagramImportSummary> items,
    required this.cursor,
  }) : items = List.unmodifiable(items);

  factory InstagramImportPage.fromMap(Map<String, dynamic> map) =>
      InstagramImportPageMapper.fromMap(map);

  final List<InstagramImportSummary> items;
  final String? cursor;

  @override
  String toString() => 'InstagramImportPage([REDACTED])';
}

@MappableClass(
  generateMethods:
      GenerateMethods.encode | GenerateMethods.copy | GenerateMethods.equals,
)
final class InstagramImportPatch with InstagramImportPatchMappable {
  const InstagramImportPatch({required this.reactivate})
    : assert(reactivate, 'Only import reactivation is supported.');

  final bool reactivate;

  @override
  String toString() => 'InstagramImportPatch([REDACTED])';
}
