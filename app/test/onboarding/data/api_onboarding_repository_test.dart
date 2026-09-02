import 'package:craftsky_app/onboarding/data/api_onboarding_repository.dart';
import 'package:craftsky_app/onboarding/data/onboarding_api_client.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  test(
    'reads status and completes through the authenticated API contract',
    () async {
      final dio = Dio();
      final adapter = DioAdapter(dio: dio);
      final repository = ApiOnboardingRepository(OnboardingApiClient(dio));
      adapter
        ..onGet('/v1/onboarding/status', (server) {
          server.reply(200, {'completed': false});
        })
        ..onPost('/v1/onboarding/completion', (server) {
          server.reply(200, {
            'completed': true,
            'completedAt': '2026-08-31T12:00:00Z',
          });
        });

      expect((await repository.readStatus()).completed, isFalse);
      final completed = await repository.complete();
      expect(completed.completed, isTrue);
      expect(completed.completedAt, DateTime.utc(2026, 8, 31, 12));
    },
  );
}
