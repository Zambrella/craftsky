import 'dart:convert';

import 'package:dart_mappable/dart_mappable.dart';

part 'service_status_document.mapper.dart';

@MappableEnum()
enum ServiceStatusMode { normal, announcement, maintenance }

@MappableClass(
  ignoreNull: true,
  generateMethods: GenerateMethods.decode | GenerateMethods.encode,
)
final class ServiceStatusDocument with ServiceStatusDocumentMappable {
  const ServiceStatusDocument({
    required this.mode,
    required this.revision,
    this.schemaVersion = 1,
    this.title,
    this.message,
    this.estimatedRecoveryAt,
  });

  factory ServiceStatusDocument.decode(List<int> bytes) {
    if (bytes.length > maxBytes) throw const FormatException('Status size');
    final value = jsonDecode(utf8.decode(bytes));
    if (value is! Map<String, dynamic> ||
        value['schemaVersion'] is! int ||
        value['schemaVersion'] != 1) {
      throw const FormatException('Status schema');
    }
    // Mapping handles types and enum decoding; validation handles the contract.
    if (value.containsKey('estimatedRecoveryAt')) {
      _estimate(value['estimatedRecoveryAt']);
    }
    final ServiceStatusDocument document;
    try {
      document = ServiceStatusDocumentMapper.fromMap(value);
    } on MapperException {
      throw const FormatException('Status fields');
    }
    if (value['revision'] is! String ||
        !_revision.hasMatch(document.revision)) {
      throw const FormatException('Status revision');
    }
    void validateText(String key, String? text, int limit) {
      if (!value.containsKey(key) &&
          document.mode == ServiceStatusMode.normal) {
        return;
      }
      if (value[key] is! String ||
          text == null ||
          text.trim().isEmpty ||
          text.runes.length > limit ||
          text.runes.any((rune) => rune >= 0xd800 && rune <= 0xdfff)) {
        throw const FormatException('Status text');
      }
    }

    validateText('title', document.title, 160);
    validateText('message', document.message, 2400);
    return document;
  }

  static const maxBytes = 16384;
  static final _revision = RegExp(r'^[A-Za-z0-9_-]{1,64}$');
  static final _timestamp = RegExp(
    r'^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.(\d{1,6}))?(Z|[+-]\d{2}:\d{2})$',
  );

  static void _estimate(Object? value) {
    final match = value is String ? _timestamp.firstMatch(value) : null;
    if (match == null) throw const FormatException('Status estimate');
    final year = int.parse(match[1]!);
    final month = int.parse(match[2]!);
    final day = int.parse(match[3]!);
    final hour = int.parse(match[4]!);
    final minute = int.parse(match[5]!);
    final second = int.parse(match[6]!);
    final date = DateTime.utc(year, month, day);
    final zone = match[8]!;
    if (year < 1 ||
        date.year != year ||
        date.month != month ||
        date.day != day ||
        hour > 23 ||
        minute > 59 ||
        second > 59 ||
        (zone != 'Z' &&
            (int.parse(zone.substring(1, 3)) > 23 ||
                int.parse(zone.substring(4)) > 59))) {
      throw const FormatException('Status estimate');
    }
  }

  final int schemaVersion;
  final ServiceStatusMode mode;
  final String revision;
  final String? title;
  final String? message;
  final DateTime? estimatedRecoveryAt;

  @override
  String toString() => 'ServiceStatusDocument(mode: ${mode.name})';
}
