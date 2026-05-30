import 'dart:async';

import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/report_profile_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_profile_repository.dart';

void main() {
  group('ReportProfile', () {
    const submission = ReportSubmission(reasonType: 'impersonation');

    test('ignores repeated submits while in flight', () async {
      var calls = 0;
      final completer = Completer<ReportResult>();
      final repo = FakeProfileRepository(
        onReport: (_, _) {
          calls++;
          return completer.future;
        },
      );
      final container = ProviderContainer.test(
        overrides: [profileRepositoryProvider.overrideWithValue(repo)],
      );
      addTearDown(container.dispose);

      final notifier = container.read(reportProfileProvider.notifier);
      final first = notifier.submit(
        handleOrDid: 'bob.craftsky.social',
        submission: submission,
      );
      await notifier.submit(
        handleOrDid: 'bob.craftsky.social',
        submission: submission,
      );

      expect(calls, 1);
      expect(container.read(reportProfileProvider).isLoading, isTrue);

      completer.complete(
        const ReportResult(reportId: 'report-profile-1', status: 'accepted'),
      );
      await first;

      expect(
        container.read(reportProfileProvider).value?.reportId,
        'report-profile-1',
      );
    });
  });
}
