import 'package:craftsky_app/drafts/data/video_draft_quota.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-007 exact account video quota passes and one byte over fails', () {
    const quota = VideoDraftQuota();
    expect(
      quota.canSave(
        existingSourceBytes: 700000000,
        replacedSourceBytes: 0,
        newSourceBytes: 300000000,
      ),
      isTrue,
    );
    expect(
      quota.canSave(
        existingSourceBytes: 700000000,
        replacedSourceBytes: 0,
        newSourceBytes: 300000001,
      ),
      isFalse,
    );
  });

  test('UT-007 replacement subtracts prior source and never evicts', () {
    const quota = VideoDraftQuota();
    final plan = quota.plan(
      existingSourceBytes: 900000000,
      replacedSourceBytes: 250000000,
      newSourceBytes: 300000000,
    );
    expect(plan.allowed, isTrue);
    expect(plan.resultingSourceBytes, 950000000);
    expect(plan.evictedDraftIds, isEmpty);
  });
}
