import 'package:craftsky_app/profile/data/profile_api_client.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';

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
    UploadedBlob? avatar,
    bool clearAvatar = false,
    UploadedBlob? banner,
    bool clearBanner = false,
  }) => _api.updateMyProfile(
    displayName: displayName,
    description: description,
    crafts: crafts,
    avatar: avatar,
    clearAvatar: clearAvatar,
    banner: banner,
    clearBanner: clearBanner,
  );

  @override
  Future<Profile> follow(String handleOrDid) => _api.followProfile(handleOrDid);

  @override
  Future<Profile> unfollow(String handleOrDid) =>
      _api.unfollowProfile(handleOrDid);
}
