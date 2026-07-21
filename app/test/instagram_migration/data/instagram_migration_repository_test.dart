import 'package:craftsky_app/instagram_migration/data/api_instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_api_client.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  test('repository keeps the injected API client account boundary', () async {
    final dio = Dio(
      BaseOptions(baseUrl: 'https://appview.synthetic.invalid'),
    )..interceptors.add(const ErrorMappingInterceptor());
    DioAdapter(dio: dio).onGet(
      '/v1/migrations/instagram/account',
      (server) => server.reply(200, {
        'integrationAvailable': false,
        'account': null,
      }),
    );
    final repository = ApiInstagramMigrationRepository(
      InstagramMigrationApiClient(dio),
    );

    final status = await repository.getAccount();

    expect(status.integrationAvailable, isFalse);
    expect(status.account, isNull);
  });
}
