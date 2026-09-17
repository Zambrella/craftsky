import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/report_profile_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../test_support/async_submit_contract.dart';
import '../fakes/fake_profile_repository.dart';

final _did = Did.parse('did:plc:bob');
const _submission = ReportSubmission(reasonType: 'impersonation');

AsyncSubmitContractSubject<ReportResult> _subject(
  Future<ReportResult> Function() operation,
) {
  final repository = FakeProfileRepository(onReport: (_, _) => operation());
  final container = ProviderContainer.test(
    overrides: [profileRepositoryProvider.overrideWithValue(repository)],
  );
  final subscription = container.listen(reportProfileProvider, (_, _) {});

  return AsyncSubmitContractSubject(
    submit: () => container
        .read(reportProfileProvider.notifier)
        .submit(did: _did, submission: _submission),
    state: () => container.read(reportProfileProvider),
    dispose: () {
      subscription.close();
      container.dispose();
    },
  );
}

void main() {
  asyncSubmitContract(
    name: 'report profile',
    successValue: const ReportResult(
      reportId: 'report-profile-1',
      status: 'accepted',
    ),
    retryValue: const ReportResult(
      reportId: 'report-profile-retry',
      status: 'accepted',
    ),
    createSubject: _subject,
  );

  test('submits the profile DID and report payload', () async {
    String? seenDid;
    ReportSubmission? seenSubmission;
    final repository = FakeProfileRepository(
      onReport: (did, submission) async {
        seenDid = did;
        seenSubmission = submission;
        return const ReportResult(
          reportId: 'report-profile',
          status: 'accepted',
        );
      },
    );
    final container = ProviderContainer.test(
      overrides: [profileRepositoryProvider.overrideWithValue(repository)],
    );
    addTearDown(container.dispose);

    await container
        .read(reportProfileProvider.notifier)
        .submit(did: _did, submission: _submission);

    expect(seenDid, _did.toString());
    expect(seenSubmission, _submission);
  });
}
