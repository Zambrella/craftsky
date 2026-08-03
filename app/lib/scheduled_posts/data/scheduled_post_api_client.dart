import 'dart:typed_data';

import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

final class ScheduledPostApiClient {
  const ScheduledPostApiClient(this._dio);

  final Dio _dio;

  Future<List<ScheduledPostSummary>> list() => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/scheduled-posts',
    );
    final items = response.data!['items']! as List<dynamic>;
    return items
        .map((item) => _summary(item! as Map<String, dynamic>))
        .toList();
  });

  Future<ScheduledPostDetail> get(String id) => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/scheduled-posts/$id',
    );
    return _detail(response.data!);
  });

  Future<ScheduledPostDetail> create({
    required String operationId,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) => unwrapApi(() async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/v1/scheduled-posts',
      data: {
        'operationId': operationId,
        'scheduledAt': scheduledAt.toUtc().toIso8601String(),
        'payload': payload,
      },
    );
    return _detail(response.data!);
  });

  Future<ScheduledPostDetail> update({
    required String id,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) => unwrapApi(() async {
    final response = await _dio.put<Map<String, dynamic>>(
      '/v1/scheduled-posts/$id',
      data: {
        'scheduledAt': scheduledAt.toUtc().toIso8601String(),
        'payload': payload,
      },
    );
    return _detail(response.data!);
  });

  Future<void> delete(String id) => unwrapApi(() async {
    await _dio.delete<void>('/v1/scheduled-posts/$id');
  });

  Future<void> publishNow({
    required String id,
    required Map<String, dynamic> payload,
  }) => unwrapApi(() async {
    await _dio.post<Map<String, dynamic>>(
      '/v1/scheduled-posts/$id/publication',
      data: {'payload': payload},
    );
  });

  Future<void> stageMedia({
    required String id,
    required List<int> bytes,
    required String mimeType,
  }) => unwrapApi(() async {
    await _dio.put<Map<String, dynamic>>(
      '/v1/scheduled-post-media/$id',
      data: bytes,
      options: Options(contentType: mimeType),
    );
  });

  Future<Uint8List> mediaBytes(String id) => unwrapApi(() async {
    final response = await _dio.get<List<int>>(
      '/v1/scheduled-post-media/$id',
      options: Options(responseType: ResponseType.bytes),
    );
    return Uint8List.fromList(response.data!);
  });
}

ScheduledPostSummary _summary(Map<String, dynamic> map) {
  final status = scheduledPostStatusFromWire(map['status']! as String);
  if (status == null) throw const FormatException('unknown scheduled status');
  return ScheduledPostSummary(
    id: map['id']! as String,
    kind: map['kind'] == 'project'
        ? ScheduledPostKind.project
        : ScheduledPostKind.standard,
    status: status,
    text: map['textPreview']! as String,
    projectTitle: map['projectTitle'] as String?,
    scheduledAt: ScheduledInstant(
      DateTime.parse(map['scheduledAt']! as String),
    ),
    mediaIds: [if (map['firstMediaId'] case final String id) id],
    needsAttentionExpiresAt: switch (map['needsAttentionExpiresAt']) {
      final String value => DateTime.parse(value).toUtc(),
      _ => null,
    },
  );
}

ScheduledPostDetail _detail(Map<String, dynamic> map) {
  final status = scheduledPostStatusFromWire(map['status']! as String);
  if (status == null) throw const FormatException('unknown scheduled status');
  return ScheduledPostDetail(
    id: map['id']! as String,
    operationId: map['operationId']! as String,
    status: status,
    scheduledAt: ScheduledInstant(
      DateTime.parse(map['scheduledAt']! as String),
    ),
    payload: Map<String, dynamic>.from(map['payload']! as Map),
  );
}
