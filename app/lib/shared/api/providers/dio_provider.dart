import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart'
    as import_registry;
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/api/providers/session_auth_interceptor.dart';
import 'package:craftsky_app/shared/api/providers/sign_out_on_401_interceptor.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'dio_provider.g.dart';

typedef _ClientTarget = ({String token, String did, int generation});
typedef _AccountClientSelection = ({bool loaded, _ClientTarget? target});

/// Android emulator maps the host machine to 10.0.2.2. iOS simulator
/// reaches localhost directly. Android is the more common footgun so
/// it's the default; iOS devs pass
/// --dart-define=CRAFTSKY_API_BASE_URL=http://localhost:18080.
///
/// Port 18080 (not 8080) matches the host-side mapping in
/// docker-compose.yml — the appview container still listens on 8080
/// internally but is published on 18080 to avoid colliding with other
/// dev servers.
const _devDefaultBaseUrl = 'http://10.0.2.2:18080';

const _baseUrl = String.fromEnvironment(
  'CRAFTSKY_API_BASE_URL',
  defaultValue: kDebugMode ? _devDefaultBaseUrl : '',
);

/// Shared base options for both the session Dio (this file) and the
/// handoff Dio (api_client_provider.dart, family) so HTTP basics stay
/// in sync.
BaseOptions baseDioOptions() {
  if (_baseUrl.isEmpty) {
    throw StateError(
      'CRAFTSKY_API_BASE_URL must be set for non-debug builds. '
      'Pass it via --dart-define.',
    );
  }
  return BaseOptions(
    baseUrl: _baseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 15),
    headers: {'Content-Type': 'application/json'},
  );
}

@Riverpod(keepAlive: true)
Dio anonymousDio(Ref ref) {
  final client = Dio(baseDioOptions());
  client.interceptors.addAll([
    SessionAuthInterceptor.anonymous(
      readDeviceId: () => ref.read(deviceIdProvider.future),
    ),
    const ErrorMappingInterceptor(),
  ]);
  ref.onDispose(() => client.close(force: true));
  return client;
}

@riverpod
Future<Dio> accountDio(Ref ref, AccountKey account) async {
  var selection = ref.watch(
    sessionRegistryProvider.select(
      (state) => _accountClientSelection(state, account),
    ),
  );
  if (!selection.loaded) {
    await ref.read(sessionRegistryProvider.future);
    selection = _accountClientSelection(
      ref.read(sessionRegistryProvider),
      account,
    );
  }
  final deviceId = await ref.watch(deviceIdProvider.future);
  if (!ref.mounted) throw StateError('Account client disposed during build');
  final target = selection.target;
  if (target == null) throw StateError('Account session unavailable');
  final lease = AccountSessionLease(
    account: account,
    sessionGeneration: target.generation,
  );
  final client = Dio(baseDioOptions());
  client.interceptors.addAll([
    SessionAuthInterceptor.fixed(
      token: target.token,
      readDeviceId: () async => deviceId,
    ),
    const ErrorMappingInterceptor(),
    SignOutOn401Interceptor.withLease(
      lease: lease,
      invalidate: ref.read(accountSessionInvalidatorProvider),
    ),
  ]);
  ref.onDispose(() => client.close(force: true));
  return client;
}

@Riverpod(keepAlive: true)
Dio dio(Ref ref) {
  final active = ref.watch(
    sessionRegistryProvider.select(_activeClientTarget),
  );
  final client = Dio(baseDioOptions());
  if (active == null) {
    client.interceptors.addAll([
      SessionAuthInterceptor.anonymous(
        readDeviceId: () => ref.read(deviceIdProvider.future),
      ),
      const ErrorMappingInterceptor(),
    ]);
  } else {
    final lease = AccountSessionLease(
      account: AccountKey(active.did),
      sessionGeneration: active.generation,
    );
    client.interceptors.addAll([
      SessionAuthInterceptor.fixed(
        token: active.token,
        readDeviceId: () => ref.read(deviceIdProvider.future),
      ),
      const ErrorMappingInterceptor(),
      SignOutOn401Interceptor.withLease(
        lease: lease,
        invalidate: ref.read(accountSessionInvalidatorProvider),
      ),
    ]);
  }
  ref.onDispose(() => client.close(force: true));
  return client;
}

_ClientTarget? _activeClientTarget(
  AsyncValue<import_registry.SessionRegistry> state,
) {
  final registry = state.value;
  final activeDid = registry?.activeDid;
  final session = activeDid == null ? null : registry?.sessions[activeDid];
  return session == null
      ? null
      : (
          token: session.token,
          did: session.did.value,
          generation: session.sessionGeneration,
        );
}

_AccountClientSelection _accountClientSelection(
  AsyncValue<import_registry.SessionRegistry> state,
  AccountKey account,
) {
  final session = state.value?.sessions[account.did];
  return (
    loaded: state.hasValue,
    target: session == null
        ? null
        : (
            token: session.token,
            did: session.did.value,
            generation: session.sessionGeneration,
          ),
  );
}
