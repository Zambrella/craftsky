import 'package:craftsky_app/profile/data/profile_api_client.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'profile_api_client_provider.g.dart';

@Riverpod(keepAlive: true)
ProfileApiClient profileApiClient(Ref ref) =>
    ProfileApiClient(ref.watch(dioProvider));
