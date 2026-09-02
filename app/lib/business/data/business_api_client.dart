import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dio/dio.dart';

class BusinessApiClient {
  const BusinessApiClient(this._dio);

  final Dio _dio;

  Future<AccountType> updateAccountType(AccountType value) =>
      unwrapApi(() async {
        final response = await _dio.put<Map<String, dynamic>>(
          '/v1/profiles/me/account-type',
          data: {'accountType': value.toValue()},
        );
        return AccountTypeMapper.fromValue(response.data!['accountType']);
      });

  Future<RecordMutationResult> putBusinessProfile(
    Map<String, dynamic> body, {
    required Cid? expectedCid,
  }) => unwrapApi(() async {
    final response = await _dio.put<Map<String, dynamic>>(
      '/v1/profiles/me/business',
      data: body,
      options: Options(
        headers: {'If-Match': expectedCid?.toString() ?? '*'},
      ),
    );
    return RecordMutationResultMapper.fromMap(response.data!);
  });

  Future<BusinessEventPage> listProfileEvents(
    AtIdentifier owner, {
    String? cursor,
    int limit = 10,
  }) => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/profiles/$owner/events',
      queryParameters: {
        'limit': limit.toString(),
        'cursor': ?cursor,
      },
    );
    return BusinessEventPageMapper.fromMap(response.data!);
  });

  Future<BusinessEventPage> listOwnerEvents(
    OwnerEventFilter filter, {
    String? cursor,
    int limit = 20,
  }) => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/events',
      queryParameters: {
        'filter': filter.name,
        'limit': limit.toString(),
        'cursor': ?cursor,
      },
    );
    return BusinessEventPageMapper.fromMap(response.data!);
  });

  Future<BusinessEvent> getEvent(Did owner, RecordKey rkey) =>
      unwrapApi(() async {
        final response = await _dio.get<Map<String, dynamic>>(
          '/v1/events/$owner/$rkey',
        );
        return BusinessEventMapper.fromMap(response.data!);
      });

  Future<RecordMutationResult> createEvent(Map<String, dynamic> body) =>
      unwrapApi(() async {
        final response = await _dio.post<Map<String, dynamic>>(
          '/v1/events',
          data: body,
        );
        return RecordMutationResultMapper.fromMap(response.data!);
      });

  Future<RecordMutationResult> updateEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid,
    Map<String, dynamic> body,
  ) => unwrapApi(() async {
    final response = await _dio.put<Map<String, dynamic>>(
      '/v1/events/$owner/$rkey',
      data: body,
      options: Options(headers: {'If-Match': expectedCid.toString()}),
    );
    return RecordMutationResultMapper.fromMap(response.data!);
  });

  Future<RecordMutationResult> deleteEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid,
  ) => unwrapApi(() async {
    final response = await _dio.delete<Map<String, dynamic>>(
      '/v1/events/$owner/$rkey',
      options: Options(headers: {'If-Match': expectedCid.toString()}),
    );
    return RecordMutationResultMapper.fromMap(response.data!);
  });

  Future<ReportResult> reportEvent(
    Did owner,
    RecordKey rkey,
    ReportSubmission submission,
  ) => unwrapApi(() async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/v1/events/$owner/$rkey/reports',
      data: submission.toMap(),
    );
    return ReportResultMapper.fromMap(response.data!);
  });
}
