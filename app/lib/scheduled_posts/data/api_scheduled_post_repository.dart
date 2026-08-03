import 'dart:typed_data';

import 'package:craftsky_app/scheduled_posts/data/scheduled_post_api_client.dart';
import 'package:craftsky_app/scheduled_posts/data/scheduled_post_repository.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';

final class ApiScheduledPostRepository implements ScheduledPostRepository {
  const ApiScheduledPostRepository(this._api);

  final ScheduledPostApiClient _api;

  @override
  Future<List<ScheduledPostSummary>> list() => _api.list();

  @override
  Future<ScheduledPostDetail> get(String id) => _api.get(id);

  @override
  Future<ScheduledPostDetail> create({
    required String operationId,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) => _api.create(
    operationId: operationId,
    scheduledAt: scheduledAt,
    payload: payload,
  );

  @override
  Future<ScheduledPostDetail> update({
    required String id,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) => _api.update(id: id, scheduledAt: scheduledAt, payload: payload);

  @override
  Future<void> delete(String id) => _api.delete(id);

  @override
  Future<void> publishNow({
    required String id,
    required Map<String, dynamic> payload,
  }) => _api.publishNow(id: id, payload: payload);

  @override
  Future<void> stageMedia({
    required String id,
    required List<int> bytes,
    required String mimeType,
  }) => _api.stageMedia(id: id, bytes: bytes, mimeType: mimeType);

  @override
  Future<Uint8List> mediaBytes(String id) => _api.mediaBytes(id);
}
