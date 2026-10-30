import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/models/status_app_version.dart';
import 'package:craftsky_app/service_status/providers/status_runtime_provider.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  // UT-002 / FR-002, FR-007, RULE-003 / AC-003, AC-010, AC-018
  test('platform minima and inclusive message ranges use runtime version', () {
    const minima = StatusMinimumVersions(ios: '1.8.0', android: '1.9.0');
    const message = StatusMessage(
      revision: 'A',
      minVersion: '1.8.0',
      maxVersion: '1.9.0',
      title: 'News',
      description: 'Body',
    );
    for (final platform in StatusPlatform.values) {
      expect(
        requiresStatusUpdate(
          minima,
          platform,
          StatusAppVersion.parse('1.8.0+13'),
        ),
        platform == StatusPlatform.android,
      );
    }
    for (final version in ['1.8.0+13', '1.9.0', '1.9.0+99']) {
      expect(
        statusMessageEligible(message, StatusAppVersion.parse(version)),
        isTrue,
      );
    }
    for (final version in ['1.7.9', '1.9.1', '1.8.0-alpha']) {
      expect(
        statusMessageEligible(message, StatusAppVersion.parse(version)),
        isFalse,
      );
    }
    expect(statusMessageEligible(message, null), isFalse);
    expect(requiresStatusUpdate(minima, StatusPlatform.ios, null), isFalse);
  });
  // UT-001 / RULE-002 / AC-003, AC-010, AC-016
  test(
    'strict semantic versions ignore builds and preserve numeric precedence',
    () {
      final ordered = [
        '1.8.0-999999999999998',
        '1.8.0-999999999999999',
        '1.8.0-alpha.2',
        '1.8.0-alpha.10',
        '1.8.0-beta',
        '1.8.0',
        '1.9.0',
        '1.10.0',
        '999999999999998.0.0',
        '999999999999999.0.0',
      ];
      for (var i = 1; i < ordered.length; i++) {
        expect(
          StatusAppVersion.parse(ordered[i - 1])
              .compareTo(StatusAppVersion.parse(ordered[i])),
          lessThan(0),
        );
      }
      expect(
        StatusAppVersion.parse(
          '1.8.0+13',
        ).compareTo(StatusAppVersion.parse('1.8.0+999999999999999999999999')),
        0,
      );
      for (final value in [
        '1000000000000000.0.0',
        '1.1000000000000000.0',
        '1.8.1000000000000000',
        '1.8.0-1000000000000000',
        '1.8',
        '01.8.0',
        '1.08.0',
        '1.8.0-01',
        '1.8.0+',
        ' 1.8.0',
        '1.8.0\n',
      ]) {
        expect(
          () => StatusAppVersion.parse(value),
          throwsFormatException,
          reason: value,
        );
      }
    },
  );
}
