import 'dart:typed_data';

import 'package:craftsky_app/scheduled_posts/composer/private_staging_state.dart';
import 'package:craftsky_app/scheduled_posts/composer/schedule_composer_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-019 text-only schedules proceed directly to creation', () {
    final initial = PrivateStagingState.initial(
      operationId: 'text-only-operation-id',
      sources: const [],
    );

    expect(initial.begin(ScheduleChoice.now), same(initial));
    expect(
      initial.begin(ScheduleChoice.later).phase,
      PrivateStagingPhase.creating,
    );
  });

  test('UT-019 retains staging sources, progress and ids across retries', () {
    final bytes = Uint8List.fromList([1, 2, 3, 4]);
    final source = PrivateMediaSource(
      id: 'stable-image-id',
      bytes: bytes,
      mimeType: 'image/jpeg',
    );
    final initial = PrivateStagingState.initial(
      operationId: 'stable-operation-id',
      sources: [source],
    );

    expect(initial.begin(ScheduleChoice.now), same(initial));

    final staging = initial
        .begin(ScheduleChoice.later)
        .reportProgress('stable-image-id', sent: 2, total: 4);
    expect(staging.phase, PrivateStagingPhase.staging);
    expect(staging.progress['stable-image-id']?.fraction, 0.5);
    expect(staging.sources.single.bytes, same(bytes));

    final stageFailure = staging.fail(PrivateStagingFailureStage.staging);
    final stageRetry = stageFailure.retry();
    expect(stageRetry.phase, PrivateStagingPhase.staging);
    expect(stageRetry.operationId, 'stable-operation-id');
    expect(stageRetry.sources.single.bytes, same(bytes));

    final creating = stageRetry.markStaged(
      sourceId: 'stable-image-id',
      mediaId: 'stable-private-media-id',
    );
    expect(creating.phase, PrivateStagingPhase.creating);
    expect(
      creating.stagedMediaIds['stable-image-id'],
      'stable-private-media-id',
    );

    final createFailure = creating.fail(PrivateStagingFailureStage.creation);
    final createRetry = createFailure.retry();
    expect(createRetry.phase, PrivateStagingPhase.creating);
    expect(createRetry.operationId, 'stable-operation-id');
    expect(
      createRetry.stagedMediaIds['stable-image-id'],
      'stable-private-media-id',
    );
    expect(createRetry.sources.single.bytes, same(bytes));
  });
}
