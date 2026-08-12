import 'package:craftsky_app/auth/data/account_deletion_repository.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  test('intent and acceptance expose no deletion status capability', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.invalid'));
    DioAdapter(dio: dio)
      ..onPost(
        '/v1/account-deletion/intents',
        (server) => server.reply(201, {
          'jobId': '10000000-0000-0000-0000-000000000001',
          'authUrl': 'https://pds.invalid/oauth',
          'expiresAt': '2026-08-11T12:10:00Z',
        }),
      )
      ..onPost(
        '/v1/account-deletions/10000000-0000-0000-0000-000000000001',
        (server) => server.reply(202, null),
        data: const {
          'reauthProof': 'fresh-proof',
          'confirmationHandle': '@alice.test',
        },
      );
    final client = AccountDeletionApiClient(dio);

    final intent = await client.createIntent();
    await client.accept(
      jobId: intent.jobId,
      reauthProof: 'fresh-proof',
      confirmationHandle: '@alice.test',
    );

    expect(intent.authUrl.host, 'pds.invalid');
    expect(intent.toString(), isNot(contains('fresh-proof')));
  });

  test(
    'cancel uses the authenticated client without a status header',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.invalid'));
      DioAdapter(dio: dio).onDelete(
        '/v1/account-deletion/intents/10000000-0000-0000-0000-000000000001',
        (server) => server.reply(204, null),
      );

      await AccountDeletionApiClient(dio).cancelIntent(
        jobId: '10000000-0000-0000-0000-000000000001',
      );
    },
  );
}
