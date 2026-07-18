import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/api/providers/session_auth_interceptor.dart';
import 'package:craftsky_app/shared/api/providers/sign_out_on_401_interceptor.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _RequestHandler extends RequestInterceptorHandler {
  @override
  void next(RequestOptions options) {}
}

final class _ErrorHandler extends ErrorInterceptorHandler {
  @override
  void next(DioException err) {}
}

void main() {
  test('dioProvider builds with the debug-default base URL', () {
    final container = ProviderContainer.test();

    final dio = container.read(dioProvider);

    expect(dio.options.baseUrl, 'http://10.0.2.2:18080');
    // Signed-out startup gets device identity and error mapping only. Once a
    // registry account is active, the provider rebuilds with lease-scoped 401
    // invalidation as well.
    expect(dio.interceptors, hasLength(3));
  });

  test(
    'IT-003 IT-008 active Dio captures A and scopes delayed 401 after B switch',
    () async {
      final initial = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-b',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'token-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          sessionValidationLauncherProvider.overrideWithValue((_) async {}),
          deviceIdProvider.overrideWith((ref) async => 'device-id'),
        ],
      );
      await container.read(authSessionProvider.future);

      final dioA = container.read(dioProvider);
      expect(await _bearerFrom(dioA), 'Bearer token-a');
      final leaseB = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:bob'))!;
      await container.read(sessionRegistryProvider.notifier).activate(leaseB);
      final dioB = container.read(dioProvider);

      expect(dioB, isNot(same(dioA)));
      expect(await _bearerFrom(dioB), 'Bearer token-b');

      final request = RequestOptions(path: '/v1/feed');
      dioA.interceptors.whereType<SignOutOn401Interceptor>().single.onError(
        DioException(
          requestOptions: request,
          response: Response<void>(requestOptions: request, statusCode: 401),
          type: DioExceptionType.badResponse,
        ),
        _ErrorHandler(),
      );
      for (var index = 0; index < 10; index++) {
        await Future<void>.delayed(Duration.zero);
      }

      final registry = container.read(sessionRegistryProvider).requireValue;
      expect(
        registry.sessions.containsKey(AccountKey('did:plc:alice').did),
        isFalse,
      );
      expect(
        registry.sessions.containsKey(AccountKey('did:plc:bob').did),
        isTrue,
      );
      expect(registry.activeDid?.value, 'did:plc:bob');
    },
  );

  test(
    'IT-003 metadata-only registry writes preserve fixed Dio clients',
    () async {
      final initial = SessionRegistry.empty().upsertAndActivate(
        token: 'token-a',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          sessionValidationLauncherProvider.overrideWithValue((_) async {}),
          deviceIdProvider.overrideWith((ref) async => 'device-id'),
        ],
      );
      await container.read(authSessionProvider.future);
      final account = AccountKey('did:plc:alice');
      final accountClientSubscription = container.listen(
        accountDioProvider(account),
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(accountClientSubscription.close);

      final activeBefore = container.read(dioProvider);
      final accountBefore = await container.read(
        accountDioProvider(account).future,
      );
      final registry = container.read(sessionRegistryProvider).requireValue;
      final lease = registry.leaseFor(account)!;
      await container
          .read(sessionRegistryProvider.notifier)
          .updateCachedIdentity(
            lease,
            displayName: 'Alice',
            avatarUrl: 'https://example.test/alice.jpg',
          );

      expect(container.read(dioProvider), same(activeBefore));
      expect(
        await container.read(accountDioProvider(account).future),
        same(accountBefore),
      );
      expect(await _bearerFrom(activeBefore), 'Bearer token-a');
      expect(await _bearerFrom(accountBefore), 'Bearer token-a');
    },
  );
}

Future<String?> _bearerFrom(Dio dio) async {
  final options = RequestOptions(path: '/v1/feed');
  dio.interceptors.whereType<SessionAuthInterceptor>().single.onRequest(
    options,
    _RequestHandler(),
  );
  for (var index = 0; index < 5; index++) {
    await Future<void>.delayed(Duration.zero);
  }
  return options.headers['Authorization'] as String?;
}
