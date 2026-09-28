import 'package:craftsky_app/moderation/data/moderation_repository.dart';
import 'package:craftsky_app/moderation/models/account_moderation.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

final class ApiModerationRepository implements ModerationRepository {
  const ApiModerationRepository(this._dio);

  final Dio _dio;

  @override
  Future<AccountStanding> getStanding() => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/moderation/standing',
    );
    return AccountStandingMapper.fromMap(response.data!);
  });

  @override
  Future<ModerationHistoryPage> getHistory({String? cursor, int? limit}) =>
      unwrapApi(() async {
        final response = await _dio.get<Map<String, dynamic>>(
          '/v1/moderation/history',
          queryParameters: {'cursor': ?cursor, 'limit': ?limit?.toString()},
        );
        return ModerationHistoryPageMapper.fromMap(response.data!);
      });

  @override
  Future<ModerationHistoryPage> getHistoryEntry(
    ModerationCaseReference reference,
  ) => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/moderation/history/${reference.value}',
    );
    return ModerationHistoryPageMapper.fromMap(response.data!);
  });
}
