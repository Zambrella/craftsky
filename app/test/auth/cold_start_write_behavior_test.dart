import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/profile/data/profile_api_client.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/api/providers/session_auth_interceptor.dart';
import 'package:craftsky_app/shared/api/providers/sign_out_on_401_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

final class _RegistryStorage implements SessionRegistryStorage {
  const _RegistryStorage(this.registry);

  final SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async {}
}

final class _RequestRecorder extends Interceptor {
  RequestOptions? request;

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    request = options;
    handler.next(options);
  }
}

void main() {
  setUpAll(initializeMappers);

  test('AT-012 pending cold-start validation does not gate a write', () async {
    final harness = await _ColdStartHarness.start();
    addTearDown(harness.dispose);
    DioAdapter(dio: harness.dio).onPut(
      '/v1/profiles/me',
      (server) => server.reply(200, _profile()),
      data: {'description': 'updated while validation is pending'},
    );

    final updated = await ProfileApiClient(harness.dio)
        .updateMyProfile(description: 'updated while validation is pending')
        .timeout(const Duration(seconds: 1));

    expect(harness.authState, isA<SignedIn>());
    expect(harness.validation.isCompleted, isFalse);
    expect(updated.description, 'updated while validation is pending');
    expect(harness.request?.uri.host, 'appview.example');
    expect(harness.request?.headers['Authorization'], 'Bearer craftsky-token');
    expect(harness.invalidated, isEmpty);
  });

  test(
    'IT-015 pds_session_expired during pending validation uses exact-lease '
    '401 recovery',
    () async {
      final harness = await _ColdStartHarness.start();
      addTearDown(harness.dispose);
      DioAdapter(dio: harness.dio).onPut(
        '/v1/profiles/me',
        (server) => server.reply(401, {
          'error': 'pds_session_expired',
          'message': 'Reauthorize this account to continue.',
          'requestId': 'request-cold-start',
        }),
        data: {'description': 'stale write'},
      );

      await expectLater(
        ProfileApiClient(
          harness.dio,
        ).updateMyProfile(description: 'stale write'),
        throwsA(
          isA<ApiUnauthorized>()
              .having(
                (error) => error.details.appViewError,
                'error',
                'pds_session_expired',
              )
              .having(
                (error) => error.details.requestId,
                'requestId',
                'request-cold-start',
              ),
        ),
      );
      await Future<void>.delayed(Duration.zero);

      expect(harness.validation.isCompleted, isFalse);
      expect(harness.invalidated, [harness.lease]);
      expect(harness.request?.uri.host, 'appview.example');
      expect(
        harness.request?.headers['Authorization'],
        'Bearer craftsky-token',
      );
      expect(
        harness.request?.headers.keys.map((key) => key.toLowerCase()),
        isNot(contains('dpop')),
      );
    },
  );
}

final class _ColdStartHarness {
  _ColdStartHarness._({
    required this.container,
    required this.dio,
    required this.lease,
    required this.validation,
    required this.invalidated,
    required this.requestRecorder,
    required this.authState,
  });

  static Future<_ColdStartHarness> start() async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'craftsky-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final lease = registry.activeLease!.session;
    final validation = Completer<void>();
    final invalidated = <AccountSessionLease>[];
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(registry),
        ),
        sessionValidationLauncherProvider.overrideWithValue(
          (_) => validation.future,
        ),
      ],
    );
    final authState = await container.read(authSessionProvider.future);
    final requestRecorder = _RequestRecorder();
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example'))
      ..interceptors.addAll([
        SessionAuthInterceptor.fixed(
          token: 'craftsky-token',
          readDeviceId: () async => 'device-abc',
        ),
        requestRecorder,
        const ErrorMappingInterceptor(),
        SignOutOn401Interceptor.withLease(
          lease: lease,
          invalidate: (captured) async => invalidated.add(captured),
        ),
      ]);
    return _ColdStartHarness._(
      container: container,
      dio: dio,
      lease: lease,
      validation: validation,
      invalidated: invalidated,
      requestRecorder: requestRecorder,
      authState: authState,
    );
  }

  final ProviderContainer container;
  final Dio dio;
  final AccountSessionLease lease;
  final Completer<void> validation;
  final List<AccountSessionLease> invalidated;
  final _RequestRecorder requestRecorder;
  final AuthState authState;

  RequestOptions? get request => requestRecorder.request;

  void dispose() {
    if (!validation.isCompleted) validation.complete();
    dio.close(force: true);
    container.dispose();
  }
}

Map<String, dynamic> _profile() => {
  'did': 'did:plc:alice',
  'handle': 'alice.test',
  'description': 'updated while validation is pending',
  'crafts': <String>[],
};
