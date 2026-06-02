import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/profile/data/profile_api_client.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
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
    List<Map<String, dynamic>>? descriptionFacets,
    List<String>? crafts,
    UploadedBlob? avatar,
    bool clearAvatar = false,
    UploadedBlob? banner,
    bool clearBanner = false,
  }) => _api.updateMyProfile(
    displayName: displayName,
    description: description,
    descriptionFacets: descriptionFacets,
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

  @override
  Future<ReportResult> report(
    String handleOrDid,
    ReportSubmission submission,
  ) => _api.reportProfile(handleOrDid, submission);

  @override
  Future<ProfileAccountPage> listMutualFollowers(
    String handleOrDid, {
    int? limit,
    String? cursor,
  }) => _api.listMutualFollowers(handleOrDid, limit: limit, cursor: cursor);

  @override
  Future<ProfileAccountPage> listFollowersMe({int? limit, String? cursor}) =>
      _api.listFollowersMe(limit: limit, cursor: cursor);

  @override
  Future<ProfileAccountPage> listFollowingMe({int? limit, String? cursor}) =>
      _api.listFollowingMe(limit: limit, cursor: cursor);
}
