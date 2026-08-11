import 'package:craftsky_app/auth/data/account_deletion_repository.dart';
import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  test('uncertain duplicate acceptance resolves the same job', () async {
    final ordinary = Dio(BaseOptions(baseUrl: 'https://appview.invalid'))
      ..interceptors.add(const ErrorMappingInterceptor());
    final status = Dio(
      BaseOptions(
        baseUrl: 'https://appview.invalid',
        headers: {
          'Authorization': 'DeletionStatus status-token',
          'X-Craftsky-Device-Id': 'alice-phone',
        },
      ),
    )..interceptors.add(const ErrorMappingInterceptor());
    final ordinaryAdapter = DioAdapter(dio: ordinary);
    final statusAdapter = DioAdapter(dio: status);
    const body = {
      'reauthProof': 'fresh-proof',
      'confirmationHandle': '@alice.test',
    };
    ordinaryAdapter.onPost(
      '/v1/account-deletions/10000000-0000-0000-0000-000000000001',
      (server) => server.throws(
        503,
        DioException(
          requestOptions: RequestOptions(),
          type: DioExceptionType.connectionError,
        ),
      ),
      data: body,
      headers: {'X-Craftsky-Deletion-Status': 'status-token'},
    );
    statusAdapter.onGet(
      '/v1/account-deletions/10000000-0000-0000-0000-000000000001',
      (server) => server.reply(200, {
        'jobId': '10000000-0000-0000-0000-000000000001',
        'status': 'active',
        'phase': 'queued',
        'retryAllowed': false,
        'needsReauthentication': false,
      }),
    );
    final repository = AccountDeletionRepository(
      ordinaryClient: AccountDeletionApiClient(ordinary),
      statusClient: DeletionStatusApiClient(status),
    );

    final resolved = await repository.acceptOrResolve(
      jobId: '10000000-0000-0000-0000-000000000001',
      statusToken: 'status-token',
      reauthProof: 'fresh-proof',
      confirmationHandle: '@alice.test',
    );

    expect(resolved.status, AccountDeletionStatus.active);
    expect(resolved.phase, AccountDeletionPhase.preparing);
  });

  test('status client never sends an ordinary bearer credential', () async {
    final dio = Dio(
      BaseOptions(
        baseUrl: 'https://appview.invalid',
        headers: {
          'Authorization': 'DeletionStatus restricted-token',
          'X-Craftsky-Device-Id': 'alice-phone',
        },
      ),
    )..interceptors.add(const ErrorMappingInterceptor());
    DioAdapter(dio: dio).onGet(
      '/v1/account-deletions/10000000-0000-0000-0000-000000000001',
      (server) => server.reply(200, {
        'jobId': '10000000-0000-0000-0000-000000000001',
        'status': 'needsAttention',
        'phase': 'removingCraftskyRecords',
        'retryAllowed': true,
        'needsReauthentication': false,
      }),
      headers: {
        'Authorization': 'DeletionStatus restricted-token',
        'X-Craftsky-Device-Id': 'alice-phone',
      },
    );

    final view = await DeletionStatusApiClient(dio).getStatus(
      '10000000-0000-0000-0000-000000000001',
    );

    expect(view.canRetry, isTrue);
  });

  test('former bearer recovery returns only a status capability', () async {
    final dio = Dio(
      BaseOptions(
        baseUrl: 'https://appview.invalid',
        headers: {
          'Authorization': 'Bearer former-bearer',
          'X-Craftsky-Device-Id': 'alice-phone',
        },
      ),
    )..interceptors.add(const ErrorMappingInterceptor());
    DioAdapter(dio: dio).onPost(
      '/v1/account-deletions/recover',
      (server) => server.reply(200, {
        'jobId': '10000000-0000-0000-0000-000000000001',
        'statusToken': 'status-only',
        'status': 'active',
        'phase': 'queued',
      }),
    );

    final result = await AccountDeletionRecoveryClient(dio).recover();

    expect(result.statusToken, 'status-only');
    expect(result.snapshot.status, AccountDeletionStatus.active);
  });
}
