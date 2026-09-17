import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/report_post_provider.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../test_support/async_submit_contract.dart';
import '../fakes/fake_post_repository.dart';

final _did = Did.parse('did:plc:bob');
final _rkey = RecordKey.parse('3lf2abc');
const _submission = ReportSubmission(reasonType: 'spam');

AsyncSubmitContractSubject<ReportResult> _subject(
  Future<ReportResult> Function() operation,
) {
  final repository = FakePostRepository(
    onReport: (_, _, _) => operation(),
  );
  final container = ProviderContainer.test(
    overrides: [postRepositoryProvider.overrideWithValue(repository)],
  );
  final subscription = container.listen(reportPostProvider, (_, _) {});

  return AsyncSubmitContractSubject(
    submit: () => container
        .read(reportPostProvider.notifier)
        .submit(did: _did, rkey: _rkey, submission: _submission),
    state: () => container.read(reportPostProvider),
    dispose: () {
      subscription.close();
      container.dispose();
    },
  );
}

void main() {
  asyncSubmitContract(
    name: 'report post',
    successValue: const ReportResult(
      reportId: 'report-post-1',
      status: 'accepted',
    ),
    retryValue: const ReportResult(
      reportId: 'report-post-retry',
      status: 'accepted',
    ),
    createSubject: _subject,
  );

  test('submits the post DID, record key, and report payload', () async {
    Did? seenDid;
    RecordKey? seenRkey;
    ReportSubmission? seenSubmission;
    final repository = FakePostRepository(
      onReport: (did, rkey, submission) async {
        seenDid = did;
        seenRkey = rkey;
        seenSubmission = submission;
        return const ReportResult(reportId: 'report-post', status: 'accepted');
      },
    );
    final container = ProviderContainer.test(
      overrides: [postRepositoryProvider.overrideWithValue(repository)],
    );
    addTearDown(container.dispose);

    await container
        .read(reportPostProvider.notifier)
        .submit(did: _did, rkey: _rkey, submission: _submission);

    expect(seenDid, _did);
    expect(seenRkey, _rkey);
    expect(seenSubmission, _submission);
  });
}
