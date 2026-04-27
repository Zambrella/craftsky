import 'package:craftsky_app/auth/data/auth_api_client.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auth_api_client_provider.g.dart';

@Riverpod(keepAlive: true)
AuthApiClient authApiClient(Ref ref) => AuthApiClient(ref.watch(dioProvider));
