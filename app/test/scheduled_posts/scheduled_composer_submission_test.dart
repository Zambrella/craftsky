import 'dart:async';

import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/scheduled_posts/composer/scheduled_composer_submission.dart';
import 'package:craftsky_app/scheduled_posts/data/scheduled_post_repository.dart';
import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/services/scheduled_composer_media.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'saving stages media before writing and refreshing the captured account',
    () async {
      final repository = _Repository();
      final events = <String>[];
      final progress = <ScheduledComposerProgress>[];
      final submission = _submission(
        repository,
        refresh: () async => events.add('refresh'),
        media: (images, {required stageMedia, onStaged}) async {
          events.add('stage');
          return [
            {'id': 'selected-media'},
          ];
        },
      );
      final scheduledAt = DateTime.parse('2026-10-15T18:00:00+02:00');
      await submission.save(
        operationId: 'composer-operation',
        scheduledAt: scheduledAt,
        existing: null,
        images: const [],
        payload: (_, media) async {
          events.add('payload');
          return {'kind': 'standard', 'text': 'saved text', 'media': media};
        },
        onProgress: progress.add,
      );
      expect(events, ['stage', 'payload', 'refresh']);
      expect(repository.createdOperationId, 'composer-operation');
      expect(repository.scheduledAt, DateTime.utc(2026, 10, 15, 16));
      expect(repository.payload, {
        'kind': 'standard',
        'text': 'saved text',
        'media': [
          {'id': 'selected-media'},
        ],
      });
      expect(progress.last.saving, isTrue);
    },
  );

  test(
    'account change during media staging prevents payload, write and refresh',
    () async {
      final repository = _Repository();
      final pendingMedia = Completer<List<Map<String, dynamic>>>();
      final started = Completer<void>();
      var current = true;
      var refreshed = false;
      final submission = _submission(
        repository,
        isCurrent: () => current,
        refresh: () async => refreshed = true,
        media: (images, {required stageMedia, onStaged}) {
          started.complete();
          return pendingMedia.future;
        },
      );
      final result = submission.save(
        operationId: 'original-account-operation',
        scheduledAt: DateTime.utc(2026, 10, 16),
        existing: null,
        images: const [],
        payload: (_, _) async {
          fail('superseded editor read its payload');
        },
        onProgress: (_) {},
      );
      final assertion = expectLater(
        result,
        throwsA(isA<AccountOperationDiscarded>()),
      );
      await started.future;
      current = false;
      pendingMedia.complete([]);
      await assertion;
      expect(repository.payload, isNull);
      expect(refreshed, isFalse);
    },
  );

  test(
    'publishing an edited schedule retains its existing frozen metadata',
    () async {
      final repository = _Repository();
      final existing = _detail({
        'kind': 'project',
        'originalMetadata': 'retained',
      });
      await _submission(repository).publishNow(
        existing: existing,
        images: const [],
        payload: (_, _) async => {
          'text': 'edited text',
          'media': <Map<String, dynamic>>[],
        },
        onProgress: (_) {},
      );
      expect(repository.publishedId, existing.id);
      expect(repository.payload, {
        'kind': 'project',
        'originalMetadata': 'retained',
        'text': 'edited text',
        'media': <Map<String, dynamic>>[],
      });
    },
  );

  test(
    'failed schedule write preserves its cause and skips list refresh',
    () async {
      final failure = StateError('write failure');
      final repository = _Repository()..failure = failure;
      var refreshed = false;
      await expectLater(
        _submission(repository, refresh: () async => refreshed = true).save(
          operationId: 'same-operation',
          scheduledAt: DateTime.utc(2026, 10, 16),
          existing: null,
          images: const [],
          payload: (_, _) async => {'kind': 'standard'},
          onProgress: (_) {},
        ),
        throwsA(same(failure)),
      );
      expect(refreshed, isFalse);
    },
  );
}

ScheduledComposerSubmission _submission(
  _Repository repository, {
  bool Function()? isCurrent,
  Future<void> Function()? refresh,
  ScheduledComposerMediaMaterializer? media,
}) => ScheduledComposerSubmission(
  operation: ActiveAccountOperation.withGuard(
    ownership: null,
    isCurrent: isCurrent ?? () => true,
  ),
  repository: () async => repository,
  refresh: refresh ?? () async {},
  materializeMedia:
      media ?? (images, {required stageMedia, onStaged}) async => [],
);

ScheduledPostDetail _detail(Map<String, dynamic> payload) =>
    ScheduledPostDetail(
      id: 'existing-schedule',
      operationId: 'existing-operation',
      status: ScheduledPostStatus.scheduled,
      scheduledAt: ScheduledInstant(DateTime.utc(2026, 10, 16)),
      payload: payload,
    );

final class _Repository extends Fake implements ScheduledPostRepository {
  String? createdOperationId;
  String? publishedId;
  DateTime? scheduledAt;
  Map<String, dynamic>? payload;
  Error? failure;

  @override
  Future<ScheduledPostDetail> create({
    required String operationId,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) async {
    if (failure case final error?) throw error;
    createdOperationId = operationId;
    this.scheduledAt = scheduledAt;
    this.payload = payload;
    return _detail(payload);
  }

  @override
  Future<void> publishNow({
    required String id,
    required Map<String, dynamic> payload,
  }) async {
    publishedId = id;
    this.payload = payload;
  }
}
