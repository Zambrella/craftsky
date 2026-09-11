import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/moderation/data/api_moderation_repository.dart';
import 'package:craftsky_app/moderation/models/account_moderation.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  test('uses owner endpoints and round-trips cursor opaquely', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
    DioAdapter(dio: dio)
      ..onGet(
        '/v1/moderation/standing',
        (server) => server.reply(200, {
          'activeStrikeCount': 0,
          'strikeThreshold': 3,
          'thresholdSuspended': false,
          'severeSuspended': false,
          'suspended': false,
        }),
      )
      ..onGet(
        '/v1/moderation/history',
        (server) => server.reply(200, {'items': <Object>[]}),
        queryParameters: {'cursor': 'opaque:abc', 'limit': '20'},
      );
    final repository = ApiModerationRepository(dio);

    expect((await repository.getStanding()).strikeThreshold, 3);
    expect(
      (await repository.getHistory(cursor: 'opaque:abc', limit: 20)).items,
      isEmpty,
    );
  });

  test('canonical reference is the only identifier in detail path', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
    DioAdapter(dio: dio).onGet(
      '/v1/moderation/history/MOD-550e8400-e29b-41d4-a716-446655440000',
      (server) => server.reply(200, {'items': <Object>[]}),
    );

    await ApiModerationRepository(dio).getHistoryEntry(
      ModerationCaseReference.parse(
        'mod-550E8400-E29B-41D4-A716-446655440000',
      ),
    );
  });
}
