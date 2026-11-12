import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AccountHosting {
  const new({
    required this.did,
    required this.handle,
    this.pdsUrl,
    this.hostingProvider,
    this.accountManagementUrl,
    this.passwordChangeUrl,
  });

  factory fromMap(Map<String, dynamic> map) => .new(
    did: map['did'] as String,
    handle: map['handle'] as String,
    pdsUrl: map['pdsUrl'] as String?,
    hostingProvider: map['hostingProvider'] as String?,
    accountManagementUrl: map['accountManagementUrl'] as String?,
    passwordChangeUrl: map['passwordChangeUrl'] as String?,
  );

  final String did;
  final String handle;
  final String? pdsUrl;
  final String? hostingProvider;
  final String? accountManagementUrl;
  final String? passwordChangeUrl;
}

final FutureProvider<AccountHosting?> accountHostingProvider =
    FutureProvider.autoDispose<AccountHosting?>((
      ref,
    ) async {
      final lease = ref.watch(
        sessionRegistryProvider.select(
          (state) => state.value?.activeLease?.session,
        ),
      );
      if (lease == null) return null;
      final dio = await ref.watch(accountDioProvider(lease.account).future);
      final info = await unwrapApi(() async {
        final response = await dio.get<Map<String, dynamic>>('/v1/whoami');
        return AccountHosting.fromMap(response.data!);
      });
      if (!ref.mounted ||
          ref.read(sessionRegistryProvider).value?.activeLease?.session !=
              lease ||
          info.did != lease.account.did.toString()) {
        return null;
      }
      return info;
    });
