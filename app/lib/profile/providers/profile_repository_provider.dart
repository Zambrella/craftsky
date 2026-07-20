import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/profile/data/api_profile_repository.dart';
import 'package:craftsky_app/profile/data/profile_api_client.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/providers/profile_api_client_provider.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'profile_repository_provider.g.dart';

@Riverpod(keepAlive: true)
ProfileRepository profileRepository(Ref ref) =>
    ApiProfileRepository(ref.watch(profileApiClientProvider));

@riverpod
Future<ProfileRepository> accountRelationshipRepository(
  Ref ref,
  AccountKey account,
) async => ApiProfileRepository(
  ProfileApiClient(await ref.watch(accountDioProvider(account).future)),
);
