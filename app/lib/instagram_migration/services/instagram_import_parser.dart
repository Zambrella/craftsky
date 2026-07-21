import 'dart:convert';
import 'dart:typed_data';

import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';

enum InstagramImportParseErrorCode {
  invalidJson,
  unsupportedShape,
  unsupportedFormat,
  fileTooLarge,
  tooManyEntries,
}

final class InstagramImportParseException implements Exception {
  const InstagramImportParseException(this.code);

  final InstagramImportParseErrorCode code;

  @override
  String toString() => 'InstagramImportParseException(${code.name})';
}

final class InstagramImportParser {
  const InstagramImportParser();

  static const int maxFileBytes = 20 * 1024 * 1024;
  static const int maxEntries = 10000;
  static final RegExp _usernamePattern = RegExp(r'^[A-Za-z0-9._]{1,30}$');

  InstagramImportParseResult parseJson(
    Uint8List bytes, {
    required InstagramRelationshipDirection direction,
  }) {
    if (bytes.length > maxFileBytes) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.fileTooLarge,
      );
    }
    if (bytes.length >= 2 && bytes[0] == 0x50 && bytes[1] == 0x4b) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.unsupportedFormat,
      );
    }
    final Object? decoded;
    try {
      decoded = jsonDecode(utf8.decode(bytes));
    } on FormatException {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.invalidJson,
      );
    }
    final records = _recordsFor(decoded, direction);
    final values = <Object?>[];
    var malformedRecordCount = 0;
    for (final recordValue in records) {
      if (recordValue is! Map<String, dynamic>) {
        malformedRecordCount++;
        continue;
      }
      final stringListData = recordValue['string_list_data'];
      if (stringListData is! List<dynamic>) {
        malformedRecordCount++;
        continue;
      }
      for (final value in stringListData) {
        values.add(value is Map<String, dynamic> ? value['value'] : null);
      }
    }
    return _normalizeValues(
      values,
      direction: direction,
      initialIgnoredCount: malformedRecordCount,
    );
  }

  InstagramImportParseResult parseManual(
    String source, {
    required InstagramRelationshipDirection direction,
  }) {
    if (utf8.encode(source).length > maxFileBytes) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.fileTooLarge,
      );
    }
    return _normalizeValues(
      const LineSplitter().convert(source),
      direction: direction,
    );
  }

  InstagramImportParseResult _normalizeValues(
    Iterable<Object?> values, {
    required InstagramRelationshipDirection direction,
    int initialIgnoredCount = 0,
  }) {
    final entries = <InstagramImportEntry>[];
    final seen = <String>{};
    var ignoredEntryCount = initialIgnoredCount;
    var duplicateEntryCount = 0;
    for (final value in values) {
      final username = value is String ? _normalizeUsername(value) : null;
      if (username == null) {
        ignoredEntryCount++;
        continue;
      }
      if (!seen.add(username)) {
        duplicateEntryCount++;
        continue;
      }
      if (seen.length > maxEntries) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.tooManyEntries,
        );
      }
      entries.add(
        InstagramImportEntry(username: username, direction: direction),
      );
    }
    return InstagramImportParseResult(
      entries: List.unmodifiable(entries),
      ignoredEntryCount: ignoredEntryCount,
      duplicateEntryCount: duplicateEntryCount,
    );
  }

  String? _normalizeUsername(String value) {
    final trimmed = value.trim();
    final withoutAt = trimmed.startsWith('@') ? trimmed.substring(1) : trimmed;
    if (!_usernamePattern.hasMatch(withoutAt)) return null;
    final normalized = withoutAt.toLowerCase();
    return normalized;
  }

  List<dynamic> _recordsFor(
    Object? decoded,
    InstagramRelationshipDirection direction,
  ) {
    switch (direction) {
      case InstagramRelationshipDirection.following:
        if (decoded case {
          'relationships_following': final List<dynamic> records,
        }) {
          return records;
        }
      case InstagramRelationshipDirection.follower:
        if (decoded is List<dynamic>) return decoded;
    }
    throw const InstagramImportParseException(
      InstagramImportParseErrorCode.unsupportedShape,
    );
  }
}
