import 'dart:convert';

import 'package:craftsky_app/service_status/models/status_app_version.dart';

import 'package:dart_mappable/dart_mappable.dart';

part 'service_status_document.mapper.dart';

// A presentation summary only; independent controls live on the document.
@MappableEnum()
enum ServiceStatusMode { normal, announcement, maintenance }

@MappableClass(
  ignoreNull: true,
  generateMethods: GenerateMethods.decode | GenerateMethods.encode,
)
final class ServiceStatusDocument with ServiceStatusDocumentMappable {
  const new({
    required this.revision,
    this.schemaVersion = 2,
    this.maintenance,
    this.minimumVersions,
    this.message,
  });

  factory decode(List<int> bytes) {
    if (bytes.length > maxBytes) throw const FormatException('Status size');
    final value = jsonDecode(utf8.decode(bytes));
    if (value is! Map<String, dynamic> ||
        value['schemaVersion'] is! int ||
        value['schemaVersion'] != 2) {
      throw const FormatException('Status schema');
    }
    _validateRevision(value['revision']);
    for (final section in ['maintenance', 'minimumVersions', 'message']) {
      if (!value.containsKey(section)) continue;
      final item = value[section];
      if (item is! Map<String, dynamic>) {
        throw const FormatException('Status section');
      }
      if (section == 'minimumVersions') {
        for (final platform in ['ios', 'android', 'web']) {
          if (item.containsKey(platform)) _version(item[platform]);
        }
        continue;
      }
      _text(item['title'], 160);
      _text(item['description'], 2400);
      if (section == 'maintenance' && item.containsKey('estimatedRecoveryAt')) {
        _estimate(item['estimatedRecoveryAt']);
      }
      if (section == 'message') {
        _validateRevision(item['revision']);
        _version(item['minVersion']);
        if (item.containsKey('maxVersion')) {
          _version(item['maxVersion']);
          if (StatusAppVersion.parse(item['minVersion'] as String).compareTo(
                StatusAppVersion.parse(item['maxVersion'] as String),
              ) >
              0) {
            throw const FormatException('Status range');
          }
        }
        if (item.containsKey('imageUrl')) _url(item['imageUrl']);
        if (item.containsKey('cta')) {
          final cta = item['cta'];
          if (cta is! Map<String, dynamic> ||
              cta.containsKey('url') == cta.containsKey('destination')) {
            throw const FormatException('Status CTA');
          }
          _text(cta['text'], 160);
          if (cta.containsKey('url')) {
            _url(cta['url']);
          } else if (![
            'feed',
            'projects',
            'search',
            'notifications',
            'profile',
            'settings',
          ].contains(cta['destination'])) {
            throw const FormatException('Status destination');
          }
        }
      }
    }
    try {
      return ServiceStatusDocumentMapper.fromMap(value);
    } on MapperException {
      throw const FormatException('Status fields');
    }
  }

  static const maxBytes = 16384;
  static final _revision = RegExp(r'^[A-Za-z0-9_-]{1,64}$');
  static void _validateRevision(Object? value) {
    if (value is! String || !_revision.hasMatch(value)) {
      throw const FormatException('Status revision');
    }
  }

  static void _text(Object? value, int limit) {
    if (value is! String ||
        value.trim().isEmpty ||
        value.runes.length > limit ||
        value.runes.any((r) => r >= 0xd800 && r <= 0xdfff)) {
      throw const FormatException('Status text');
    }
  }

  static void _version(Object? value) {
    if (value is! String) throw const FormatException('Status version');
    StatusAppVersion.parse(value);
  }

  static void _url(Object? value) {
    if (value is! String ||
        value.codeUnits.any(
          (unit) => unit <= 32 || unit >= 127 || unit == 92,
        ) ||
        RegExp('%(?![0-9A-Fa-f]{2})').hasMatch(value)) {
      throw const FormatException('Status URL');
    }
    final uri = Uri.tryParse(value);
    if (uri == null ||
        uri.scheme != 'https' ||
        uri.host.isEmpty ||
        RegExp('^[^:]+://[^/?#]*@').hasMatch(value) ||
        uri.port > 65535) {
      throw const FormatException('Status URL');
    }
  }

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
  final String revision;
  final StatusMaintenance? maintenance;
  final StatusMinimumVersions? minimumVersions;
  final StatusMessage? message;
  ServiceStatusMode get mode => maintenance != null
      ? ServiceStatusMode.maintenance
      : message != null
      ? ServiceStatusMode.announcement
      : ServiceStatusMode.normal;
  @override
  String toString() => 'ServiceStatusDocument(mode: ${mode.name})';
}

@MappableClass(
  ignoreNull: true,
  generateMethods: GenerateMethods.decode | GenerateMethods.encode,
)
final class StatusMaintenance with StatusMaintenanceMappable {
  const new({
    required this.title,
    required this.description,
    this.estimatedRecoveryAt,
  });
  final String title;
  final String description;
  final DateTime? estimatedRecoveryAt;
  @override
  String toString() => 'StatusMaintenance()';
}

@MappableClass(
  ignoreNull: true,
  generateMethods: GenerateMethods.decode | GenerateMethods.encode,
)
final class StatusMinimumVersions with StatusMinimumVersionsMappable {
  const new({this.ios, this.android, this.web});
  final String? ios;
  final String? android;
  final String? web;
}

@MappableClass(
  ignoreNull: true,
  generateMethods: GenerateMethods.decode | GenerateMethods.encode,
)
final class StatusMessage with StatusMessageMappable {
  const new({
    required this.revision,
    required this.minVersion,
    required this.title,
    required this.description,
    this.maxVersion,
    this.imageUrl,
    this.cta,
  });
  final String revision;
  final String minVersion;
  final String? maxVersion;
  final String title;
  final String description;
  final String? imageUrl;
  final StatusMessageCta? cta;
  @override
  String toString() => 'StatusMessage()';
}

@MappableClass(
  ignoreNull: true,
  generateMethods: GenerateMethods.decode | GenerateMethods.encode,
)
final class StatusMessageCta with StatusMessageCtaMappable {
  const new({required this.text, this.url, this.destination});
  final String text;
  final String? url;
  final String? destination;
  @override
  String toString() => 'StatusMessageCta()';
}
