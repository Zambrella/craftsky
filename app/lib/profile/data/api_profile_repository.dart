import 'package:craftsky_app/profile/data/profile_api_client.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';

/// Production [ProfileRepository] backed by the AppView HTTP API.
class ApiProfileRepository implements ProfileRepository {
  const ApiProfileRepository(this._api);

  final ProfileApiClient _api;

  @override
  Future<Profile> fetch(String handleOrDid) => _api.getProfile(handleOrDid);

  @override
  Future<Profile> fetchMe() => _api.getMyProfile();

  @override
  Future<Profile> updateMe({
    String? displayName,
    String? description,
    List<String>? crafts,
  }) => _api.updateMyProfile(
    displayName: displayName,
    description: description,
    crafts: crafts,
  );
}
