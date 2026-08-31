import 'package:craftsky_app/profile/models/profile_save_result.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('partial result retries only the failed record', () {
    final result = CombinedProfileSaveResult(
      ordinary: const PerRecordSaveOutcome<String>.success('ordinary saved'),
      business: PerRecordSaveOutcome<String>.failure(Exception('business')),
    );

    expect(result.isFullSuccess, isFalse);
    expect(result.isPartialSuccess, isTrue);
    expect(result.retryOrdinary, isFalse);
    expect(result.retryBusiness, isTrue);
    expect(result.failedPortions, {ProfileSavePortion.business});
    expect(result.ordinary.value, 'ordinary saved');
  });

  test('inverse partial result retains only ordinary failure', () {
    final result = CombinedProfileSaveResult(
      ordinary: PerRecordSaveOutcome<String>.failure(Exception('ordinary')),
      business: const PerRecordSaveOutcome<String>.success('business saved'),
    );

    expect(result.retryOrdinary, isTrue);
    expect(result.retryBusiness, isFalse);
    expect(result.failedPortions, {ProfileSavePortion.ordinary});
  });

  test('conflict failure is classified from the existing API error', () {
    final outcome = PerRecordSaveOutcome<String>.failure(
      const ApiBadRequest('pds_record_conflict'),
    );

    expect(outcome.failureKind, ProfileSaveFailureKind.conflict);
  });

  test('full success has no retry plan', () {
    const result = CombinedProfileSaveResult(
      ordinary: PerRecordSaveOutcome<String>.success('ordinary'),
      business: PerRecordSaveOutcome<String>.success('business'),
    );

    expect(result.isFullSuccess, isTrue);
    expect(result.isPartialSuccess, isFalse);
    expect(result.failedPortions, isEmpty);
  });

  test('both failures retain both records for retry', () {
    final result = CombinedProfileSaveResult(
      ordinary: PerRecordSaveOutcome<String>.failure(Exception('ordinary')),
      business: PerRecordSaveOutcome<String>.failure(Exception('business')),
    );

    expect(result.isFullSuccess, isFalse);
    expect(result.isPartialSuccess, isFalse);
    expect(result.retryOrdinary, isTrue);
    expect(result.retryBusiness, isTrue);
  });
}
