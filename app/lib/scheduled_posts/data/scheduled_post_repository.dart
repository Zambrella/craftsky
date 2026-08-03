import 'dart:typed_data';

import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:dio/dio.dart';

abstract interface class ScheduledPostRepository {
  Future<List<ScheduledPostSummary>> list();

  Future<ScheduledPostDetail> get(String id);

  Future<ScheduledPostDetail> create({
    required String operationId,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  });

  Future<ScheduledPostDetail> update({
    required String id,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  });

  Future<void> delete(String id);

  Future<void> publishNow({
    required String id,
    required Map<String, dynamic> payload,
  });

  Future<void> stageMedia({
    required String id,
    required List<int> bytes,
    required String mimeType,
    CancelToken? cancelToken,
  });

  Future<Uint8List> mediaBytes(String id);
}
