import 'dart:async';

import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/report_post_provider.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

void main() {
  group('ReportPost', () {
    final did = Did.parse('did:plc:bob');
    final rkey = RecordKey.parse('3lf2abc');
    const submission = ReportSubmission(reasonType: 'spam');

    test('ignores repeated submits while in flight', () async {
      var calls = 0;
      final completer = Completer<ReportResult>();
      final repo = FakePostRepository(
        onReport: (_, _, _) {
          calls++;
          return completer.future;
        },
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(repo)],
      );
      addTearDown(container.dispose);

      final notifier = container.read(reportPostProvider.notifier);
      final first = notifier.submit(
        did: did,
        rkey: rkey,
        submission: submission,
      );
      await notifier.submit(did: did, rkey: rkey, submission: submission);

      expect(calls, 1);
      expect(container.read(reportPostProvider).isLoading, isTrue);

      completer.complete(
        const ReportResult(reportId: 'report-post-1', status: 'accepted'),
      );
      await first;

      expect(
        container.read(reportPostProvider).value?.reportId,
        'report-post-1',
      );
    });

    test('surfaces error and allows retry', () async {
      var calls = 0;
      final repo = FakePostRepository(
        onReport: (_, _, _) async {
          calls++;
          if (calls == 1) throw Exception('network');
          return const ReportResult(
            reportId: 'report-post-retry',
            status: 'accepted',
          );
        },
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(repo)],
      );
      addTearDown(container.dispose);

      await container
          .read(reportPostProvider.notifier)
          .submit(did: did, rkey: rkey, submission: submission);
      expect(container.read(reportPostProvider).hasError, isTrue);

      await container
          .read(reportPostProvider.notifier)
          .submit(did: did, rkey: rkey, submission: submission);

      expect(calls, 2);
      expect(
        container.read(reportPostProvider).value?.reportId,
        'report-post-retry',
      );
    });
  });
}
