import 'package:craftsky_app/auth/data/auth_api_client.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auth_api_client_provider.g.dart';

@Riverpod(keepAlive: true)
AuthApiClient authApiClient(Ref ref) =>
    AuthApiClient(ref.watch(anonymousDioProvider));

// The builder's concrete family type is intentionally inferred by Riverpod.
// ignore: specify_nonobvious_property_types
final accountAuthApiClientProvider = FutureProvider.autoDispose
    .family<AuthApiClient, AccountKey>(
      (ref, account) async =>
          AuthApiClient(await ref.watch(accountDioProvider(account).future)),
    );
