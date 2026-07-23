import 'package:craftsky_app/instagram_migration/data/instagram_migration_api_client.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  Dio buildDio() => Dio(
    BaseOptions(baseUrl: 'https://appview.synthetic.invalid'),
  )..interceptors.add(const ErrorMappingInterceptor());

  group('InstagramMigrationApiClient', () {
    test(
      'IT-014 creates a verification using the exact wire contract',
      () async {
        final dio = buildDio();
        DioAdapter(dio: dio).onPost(
          '/v1/migrations/instagram/verifications',
          (server) => server.reply(201, {
            'verificationId': 'synthetic-verification-id',
            'state': 'pendingDm',
            'challenge': 'CSKY-2345-6789-ABCD-E',
            'expiresAt': '2026-07-19T12:10:00Z',
            'dmUrl': 'https://instagram.com/direct/t/synthetic',
            'futureAdditiveField': true,
          }),
          data: <String, Object?>{},
        );

        final result = await InstagramMigrationApiClient(
          dio,
        ).createVerification();

        expect(result.state, InstagramVerificationState.pendingDm);
        expect(result.challenge, 'CSKY-2345-6789-ABCD-E');
        expect(result.expiresAt, DateTime.utc(2026, 7, 19, 12, 10));
        expect(result.dmUrl, Uri.https('instagram.com', '/direct/t/synthetic'));
        expect(result.toString(), isNot(contains(result.verificationId)));
        expect(result.toString(), isNot(contains(result.challenge)));
        expect(result.toString(), isNot(contains(result.dmUrl.toString())));
      },
    );

    test('IT-014 decodes every verification state and safe unknowns', () async {
      final states = <String, InstagramVerificationState>{
        'pendingDm': InstagramVerificationState.pendingDm,
        'processing': InstagramVerificationState.processing,
        'pendingConfirmation': InstagramVerificationState.pendingConfirmation,
        'confirmed': InstagramVerificationState.confirmed,
        'expired': InstagramVerificationState.expired,
        'cancelled': InstagramVerificationState.cancelled,
        'superseded': InstagramVerificationState.superseded,
        'rejected': InstagramVerificationState.rejected,
        'conflicted': InstagramVerificationState.conflicted,
        'futureState': InstagramVerificationState.unknown,
      };

      for (final MapEntry(key: wire, value: expected) in states.entries) {
        final dio = buildDio();
        DioAdapter(dio: dio).onGet(
          '/v1/migrations/instagram/verifications/synthetic-verification-id',
          (server) => server.reply(200, {
            'verificationId': 'synthetic-verification-id',
            'state': wire,
            'expiresAt': '2026-07-19T12:10:00Z',
            if (wire == 'pendingConfirmation')
              'candidateUsername': 'synthetic.private.username',
            if (wire == 'rejected') 'retryCode': 'profileLookupUnavailable',
          }),
        );

        final result = await InstagramMigrationApiClient(
          dio,
        ).getVerification('synthetic-verification-id');

        expect(result.state, expected);
        if (wire == 'pendingConfirmation') {
          expect(result.candidateUsername, 'synthetic.private.username');
          expect(
            result.toString(),
            isNot(contains('synthetic.private.username')),
          );
        }
        if (wire == 'rejected') {
          expect(
            result.retryCode,
            InstagramVerificationRetryCode.profileLookupUnavailable,
          );
        }
      }
    });

    test('IT-022 reads the nullable current verification contract', () async {
      final withCurrentDio = buildDio();
      DioAdapter(dio: withCurrentDio).onGet(
        '/v1/migrations/instagram/verifications/current',
        (server) => server.reply(200, {
          'verification': {
            'verificationId': 'synthetic-verification-id',
            'state': 'pendingConfirmation',
            'expiresAt': '2026-07-22T16:10:00Z',
            'candidateUsername': 'synthetic.candidate',
          },
        }),
      );
      final current = await InstagramMigrationApiClient(
        withCurrentDio,
      ).getCurrentVerification();
      expect(current?.verificationId, 'synthetic-verification-id');
      expect(
        current?.state,
        InstagramVerificationState.pendingConfirmation,
      );
      expect(current?.challenge, isNull);

      final emptyDio = buildDio();
      DioAdapter(dio: emptyDio).onGet(
        '/v1/migrations/instagram/verifications/current',
        (server) => server.reply(200, {'verification': null}),
      );
      expect(
        await InstagramMigrationApiClient(emptyDio).getCurrentVerification(),
        isNull,
      );
    });

    test(
      'IT-014 cancels a verification with permanent 204 semantics',
      () async {
        final dio = buildDio();
        DioAdapter(dio: dio).onDelete(
          '/v1/migrations/instagram/verifications/synthetic-verification-id',
          (server) => server.reply(204, null),
        );

        await InstagramMigrationApiClient(
          dio,
        ).cancelVerification('synthetic-verification-id');
      },
    );

    test(
      'IT-014 uses the exact account confirmation and settings wires',
      () async {
        final dio = buildDio();
        final account = {
          'state': 'active',
          'username': 'synthetic.private.username',
          'discoverable': true,
          'conflictPending': false,
          'reactivationRequired': false,
          'verifiedAt': '2026-07-19T12:00:00Z',
        };
        DioAdapter(dio: dio)
          ..onPost(
            '/v1/migrations/instagram/verifications/synthetic-verification-id/confirm',
            (server) => server.reply(200, {
              'state': 'confirmed',
              'account': account,
            }),
            data: {'discoverable': true},
          )
          ..onGet(
            '/v1/migrations/instagram/account',
            (server) => server.reply(200, {
              'integrationAvailable': false,
              'account': account,
            }),
          )
          ..onPatch(
            '/v1/migrations/instagram/settings',
            (server) => server.reply(200, {
              'integrationAvailable': false,
              'account': {...account, 'discoverable': false},
            }),
            data: {'discoverable': false},
          )
          ..onDelete(
            '/v1/migrations/instagram/account',
            (server) => server.reply(204, null),
          );

        final api = InstagramMigrationApiClient(dio);
        final confirmation = await api.confirmVerification(
          'synthetic-verification-id',
          discoverable: true,
        );
        final status = await api.getAccount();
        final updated = await api.updateSettings(
          const InstagramAccountSettingsPatch(discoverable: false),
        );
        await api.revokeAccount();

        expect(confirmation.state, InstagramVerificationState.confirmed);
        expect(confirmation.account.state, InstagramAccountLinkState.active);
        expect(status.integrationAvailable, isFalse);
        expect(status.account!.username, 'synthetic.private.username');
        expect(updated.account!.discoverable, isFalse);
        for (final value in [confirmation, status, updated]) {
          expect(
            value.toString(),
            isNot(contains('synthetic.private.username')),
          );
        }
      },
    );

    test('IT-014 uses the exact additive import lifecycle wires', () async {
      final dio = buildDio();
      final import = {
        'importId': 'synthetic-import-id',
        'state': 'active',
        'sourceType': 'instagramJson',
        'followingCount': 2,
        'createdAt': '2026-07-19T12:00:00Z',
      };
      DioAdapter(dio: dio)
        ..onPost(
          '/v1/migrations/instagram/imports',
          (server) => server.reply(201, {
            'import': import,
            'counts': {'followingCount': 2},
            'initialSuggestionCount': 1,
          }),
          data: {
            'sourceType': 'instagramJson',
            'entries': [
              {'username': 'synthetic.one'},
              {'username': 'synthetic.two'},
            ],
          },
        )
        ..onGet(
          '/v1/migrations/instagram/imports',
          (server) => server.reply(200, {
            'items': [import],
            'cursor': 'synthetic-opaque-cursor',
          }),
          queryParameters: {'limit': 20, 'cursor': 'synthetic-input-cursor'},
        )
        ..onGet(
          '/v1/migrations/instagram/imports/synthetic-import-id',
          (server) => server.reply(200, import),
        )
        ..onPatch(
          '/v1/migrations/instagram/imports/synthetic-import-id',
          (server) => server.reply(200, import),
          data: {'reactivate': true},
        )
        ..onDelete(
          '/v1/migrations/instagram/imports/synthetic-import-id',
          (server) => server.reply(204, null),
        );

      final api = InstagramMigrationApiClient(dio);
      final request = InstagramImportRequest(
        sourceType: InstagramImportSourceType.instagramJson,
        entries: const [
          InstagramImportEntry(username: 'synthetic.one'),
          InstagramImportEntry(username: 'synthetic.two'),
        ],
      );
      final created = await api.createImport(request);
      final page = await api.listImports(
        limit: 20,
        cursor: 'synthetic-input-cursor',
      );
      final detail = await api.getImport('synthetic-import-id');
      final updated = await api.updateImport(
        'synthetic-import-id',
        const InstagramImportPatch(reactivate: true),
      );
      await api.deleteImport('synthetic-import-id');

      expect(created.initialSuggestionCount, 1);
      expect(created.followingCount, 2);
      expect(page.cursor, 'synthetic-opaque-cursor');
      expect(page.items.single.state, InstagramImportState.active);
      expect(detail.followingCount, 2);
      expect(updated.state, InstagramImportState.active);
      for (final value in [created, page, detail, updated]) {
        expect(value.toString(), isNot(contains('synthetic-import-id')));
      }
    });

    test('IT-014 uses the exact suggestion review and action wires', () async {
      final dio = buildDio();
      final suggestion = {
        'suggestionId': 'synthetic-suggestion-id',
        'profile': {
          'did': 'did:plc:synthetic-target',
          'handle': 'target.synthetic.invalid',
          'displayName': 'Synthetic Target',
          'avatar': 'https://cdn.synthetic.invalid/private-avatar',
        },
        'reason': 'verifiedInstagramFollow',
        'state': 'pending',
      };
      DioAdapter(dio: dio)
        ..onGet(
          '/v1/migrations/instagram/suggestions',
          (server) => server.reply(200, {
            'items': [suggestion],
            'cursor': 'synthetic-suggestion-cursor',
          }),
          queryParameters: {'limit': 50},
        )
        ..onPost(
          '/v1/migrations/instagram/suggestions/synthetic-suggestion-id/accept',
          (server) => server.reply(200, {
            'suggestionId': 'synthetic-suggestion-id',
            'state': 'accepted',
          }),
        )
        ..onDelete(
          '/v1/migrations/instagram/suggestions/synthetic-suggestion-id',
          (server) => server.reply(204, null),
        );

      final api = InstagramMigrationApiClient(dio);
      final page = await api.listSuggestions(limit: 50);
      final accepted = await api.acceptSuggestion('synthetic-suggestion-id');
      await api.dismissSuggestion('synthetic-suggestion-id');

      expect(page.cursor, 'synthetic-suggestion-cursor');
      expect(page.items.single.state, InstagramSuggestionState.pending);
      expect(
        page.items.single.reason,
        InstagramSuggestionReason.verifiedInstagramFollow,
      );
      expect(accepted.state, InstagramSuggestionState.accepted);
      for (final privateValue in [
        'synthetic-suggestion-id',
        'did:plc:synthetic-target',
        'target.synthetic.invalid',
        'Synthetic Target',
        'https://cdn.synthetic.invalid/private-avatar',
      ]) {
        expect(page.toString(), isNot(contains(privateValue)));
        expect(accepted.toString(), isNot(contains(privateValue)));
      }
    });

    test('IT-014 discards malformed response excerpts', () async {
      const privateCanary = 'synthetic_private_timestamp_canary';
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/migrations/instagram/verifications/synthetic-verification-id',
        (server) => server.reply(200, {
          'verificationId': 'synthetic-verification-id',
          'state': 'pendingDm',
          'expiresAt': privateCanary,
        }),
      );

      Object? thrown;
      try {
        await InstagramMigrationApiClient(
          dio,
        ).getVerification('synthetic-verification-id');
      } on Object catch (error) {
        thrown = error;
      }

      expect(thrown, isA<ApiServerError>());
      expect((thrown! as ApiServerError).message, 'invalid_instagram_response');
      expect(thrown.toString(), isNot(contains(privateCanary)));
    });

    test('IT-014 rejects a non-object response without retaining it', () async {
      const privateCanary = 'synthetic_private_response_body_canary';
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/migrations/instagram/account',
        (server) => server.reply(200, privateCanary),
      );

      Object? thrown;
      try {
        await InstagramMigrationApiClient(dio).getAccount();
      } on Object catch (error) {
        thrown = error;
      }

      expect(thrown, isA<ApiServerError>());
      expect((thrown! as ApiServerError).message, 'invalid_instagram_response');
      expect(thrown.toString(), isNot(contains(privateCanary)));
    });

    test('IT-014 maps the standard AppView error envelope safely', () async {
      const privateCanary = 'synthetic_private_server_message_canary';
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/migrations/instagram/verifications',
        (server) => server.reply(503, {
          'error': 'instagram_verification_unavailable',
          'message': privateCanary,
          'requestId': 'synthetic-request-id',
        }),
        data: <String, Object?>{},
      );

      Object? thrown;
      try {
        await InstagramMigrationApiClient(dio).createVerification();
      } on Object catch (error) {
        thrown = error;
      }

      expect(thrown, isA<ApiServerError>());
      final error = thrown! as ApiServerError;
      expect(error.details.statusCode, 503);
      expect(
        error.details.appViewError,
        'instagram_verification_unavailable',
      );
      expect(error.details.requestId, 'synthetic-request-id');
      expect(error.toString(), isNot(contains(privateCanary)));
    });

    test('IT-014 uses the injected fixed-account Dio unchanged', () async {
      final dio = Dio(
        BaseOptions(
          baseUrl: 'https://appview.synthetic.invalid',
          headers: {
            'Authorization': 'Bearer synthetic-fixed-account-token',
            'X-Craftsky-Device-ID': 'synthetic-fixed-device-id',
          },
        ),
      )..interceptors.add(const ErrorMappingInterceptor());
      DioAdapter(dio: dio).onGet(
        '/v1/migrations/instagram/account',
        (server) => server.reply(200, {
          'integrationAvailable': false,
          'account': null,
        }),
        headers: {
          'Authorization': 'Bearer synthetic-fixed-account-token',
          'X-Craftsky-Device-ID': 'synthetic-fixed-device-id',
        },
      );

      final status = await InstagramMigrationApiClient(dio).getAccount();

      expect(status.integrationAvailable, isFalse);
      expect(status.account, isNull);
    });

    test(
      'IT-014 never accepts challenge plaintext from a status read',
      () async {
        const privateChallenge = 'CSKY-2345-6789-ABCD-E';
        final dio = buildDio();
        DioAdapter(dio: dio).onGet(
          '/v1/migrations/instagram/verifications/synthetic-verification-id',
          (server) => server.reply(200, {
            'verificationId': 'synthetic-verification-id',
            'state': 'pendingDm',
            'expiresAt': '2026-07-19T12:10:00Z',
            'challenge': privateChallenge,
          }),
        );

        Object? thrown;
        try {
          await InstagramMigrationApiClient(
            dio,
          ).getVerification('synthetic-verification-id');
        } on Object catch (error) {
          thrown = error;
        }

        expect(thrown, isA<ApiServerError>());
        expect(thrown.toString(), isNot(contains(privateChallenge)));
      },
    );
  });
}
