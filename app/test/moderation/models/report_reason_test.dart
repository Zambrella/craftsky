import 'package:craftsky_app/moderation/models/report_reason.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'intellectual property uses dedicated email and no in-app destination',
    () {
      const reason = ReportReason.intellectualProperty;

      expect(reason.group, ReportGroup.intellectualProperty);
      expect(reason.destination, ReportDestination.intellectualPropertyEmail);
      expect(reason.externalUri.scheme, 'mailto');
      expect(reason.externalUri.path, 'moderation@craftsky.social');
      expect(
        reason.externalUri.queryParameters['subject'],
        contains('copyright'),
      );
      expect(
        ReportGroup.intellectualProperty.reasons,
        [ReportReason.intellectualProperty],
      );
    },
  );

  test('all in-app reasons use camelCase wire values', () {
    final camelCase = RegExp(r'^[a-z][A-Za-z0-9]*$');
    final inAppReasons = ReportReason.values
        .where((reason) => reason.destination == ReportDestination.inApp)
        .toList();

    expect(inAppReasons, hasLength(29));
    for (final reason in inAppReasons) {
      expect(reason.reasonType, matches(camelCase), reason: reason.name);
    }
  });
}
