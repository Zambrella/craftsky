import 'dart:convert';
import 'dart:typed_data';

import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';

enum InstagramImportParseErrorCode {
  invalidJson,
  unsupportedShape,
  unsupportedFormat,
  invalidArchive,
  archiveTooLarge,
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

  InstagramImportParseResult parseJson(Uint8List bytes) {
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
    final records = _recordsFor(decoded);
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
      var hasLegacyValue = false;
      for (final value in stringListData) {
        if (value is Map<String, dynamic> && value.containsKey('value')) {
          hasLegacyValue = true;
          values.add(value['value']);
        }
      }
      if (!hasLegacyValue) {
        values.add(_usernameFromProfileRecord(recordValue, stringListData));
      }
    }
    return _normalizeValues(
      values,
      initialIgnoredCount: malformedRecordCount,
    );
  }

  InstagramImportParseResult parseManual(String source) {
    if (utf8.encode(source).length > maxFileBytes) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.fileTooLarge,
      );
    }
    return _normalizeValues(const LineSplitter().convert(source));
  }

  InstagramImportParseResult _normalizeValues(
    Iterable<Object?> values, {
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
      entries.add(InstagramImportEntry(username: username));
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

  String? _usernameFromProfileRecord(
    Map<String, dynamic> record,
    List<dynamic> stringListData,
  ) {
    if (stringListData.length != 1) return null;
    final data = stringListData.single;
    if (data is! Map<String, dynamic>) return null;
    final href = data['href'];
    final title = record['title'];
    if (href is! String || title is! String) return null;

    final uri = Uri.tryParse(href);
    if (uri == null ||
        uri.scheme != 'https' ||
        uri.host != 'www.instagram.com' ||
        uri.authority != 'www.instagram.com' ||
        uri.userInfo.isNotEmpty ||
        uri.hasQuery ||
        uri.hasFragment ||
        uri.pathSegments.length != 2 ||
        uri.pathSegments.first != '_u') {
      return null;
    }
    final urlUsername = uri.pathSegments.last;
    if (href != 'https://www.instagram.com/_u/$urlUsername' ||
        !_usernamePattern.hasMatch(urlUsername)) {
      return null;
    }
    final normalizedUrlUsername = urlUsername.toLowerCase();
    final normalizedTitle = _normalizeUsername(title);
    if (normalizedTitle != normalizedUrlUsername) return null;
    return normalizedUrlUsername;
  }

  List<dynamic> _recordsFor(Object? decoded) {
    if (decoded case {
      'relationships_following': final List<dynamic> records,
    }) {
      return records;
    }
    throw const InstagramImportParseException(
      InstagramImportParseErrorCode.unsupportedShape,
    );
  }
}
