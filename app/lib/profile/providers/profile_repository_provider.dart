import 'package:craftsky_app/profile/data/api_profile_repository.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/providers/profile_api_client_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'profile_repository_provider.g.dart';

@Riverpod(keepAlive: true)
ProfileRepository profileRepository(Ref ref) =>
    ApiProfileRepository(ref.watch(profileApiClientProvider));
