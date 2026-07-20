import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';

/// Programmable [ProfileRepository] for unit tests. Each method
/// delegates to an optional callback the test sets up; unstubbed
/// methods complete with `UnimplementedError` so a test that misses a
/// dependency fails loudly instead of silently no-op'ing.
///
/// Usage:
///
/// ```dart
/// final repo = FakeProfileRepository(
///   onFetch: (id) async => somePlaceholder.copyWith(handle: id),
///   onUpdateMe: ({displayName, description, crafts}) async {...},
/// );
/// final container = ProviderContainer.test(
///   overrides: [profileRepositoryProvider.overrideWithValue(repo)],
/// );
/// ```
class FakeProfileRepository implements ProfileRepository {
  FakeProfileRepository({
    this.onFetch,
    this.onFetchMe,
    this.onUpdateMe,
    this.onFollow,
    this.onUnfollow,
    this.onMute,
    this.onUnmute,
    this.onBlock,
    this.onUnblock,
    this.onReport,
    this.onListMutualFollowers,
    this.onListFollowersMe,
    this.onListFollowingMe,
    this.onListMutedProfiles,
    this.onListBlockedProfiles,
  });

  final Future<Profile> Function(String handleOrDid)? onFetch;
  final Future<Profile> Function()? onFetchMe;
  final Future<Profile> Function({
    String? displayName,
    String? description,
    List<String>? crafts,
    UploadedBlob? avatar,
    bool clearAvatar,
    UploadedBlob? banner,
    bool clearBanner,
  })?
  onUpdateMe;
  final Future<Profile> Function(String handleOrDid)? onFollow;
  final Future<Profile> Function(String handleOrDid)? onUnfollow;
  final Future<ProfileRelationship> Function(String handleOrDid)? onMute;
  final Future<ProfileRelationship> Function(String handleOrDid)? onUnmute;
  final Future<ProfileRelationship> Function(String handleOrDid)? onBlock;
  final Future<ProfileRelationship> Function(String handleOrDid)? onUnblock;
  final Future<ReportResult> Function(
    String handleOrDid,
    ReportSubmission submission,
  )?
  onReport;
  final Future<ProfileAccountPage> Function(
    String handleOrDid, {
    int? limit,
    String? cursor,
  })?
  onListMutualFollowers;
  final Future<ProfileAccountPage> Function({int? limit, String? cursor})?
  onListFollowersMe;
  final Future<ProfileAccountPage> Function({int? limit, String? cursor})?
  onListFollowingMe;
  final Future<ProfileAccountPage> Function({int? limit, String? cursor})?
  onListMutedProfiles;
  final Future<ProfileAccountPage> Function({int? limit, String? cursor})?
  onListBlockedProfiles;

  @override
  Future<Profile> fetch(String handleOrDid) =>
      onFetch?.call(handleOrDid) ??
      Future<Profile>.error(UnimplementedError('fetch not stubbed'));

  @override
  Future<Profile> fetchMe() =>
      onFetchMe?.call() ??
      Future<Profile>.error(UnimplementedError('fetchMe not stubbed'));

  @override
  Future<Profile> updateMe({
    String? displayName,
    String? description,
    List<String>? crafts,
    UploadedBlob? avatar,
    bool clearAvatar = false,
    UploadedBlob? banner,
    bool clearBanner = false,
  }) =>
      onUpdateMe?.call(
        displayName: displayName,
        description: description,
        crafts: crafts,
        avatar: avatar,
        clearAvatar: clearAvatar,
        banner: banner,
        clearBanner: clearBanner,
      ) ??
      Future<Profile>.error(UnimplementedError('updateMe not stubbed'));

  @override
  Future<Profile> follow(String handleOrDid) =>
      onFollow?.call(handleOrDid) ??
      Future<Profile>.error(UnimplementedError('follow not stubbed'));

  @override
  Future<Profile> unfollow(String handleOrDid) =>
      onUnfollow?.call(handleOrDid) ??
      Future<Profile>.error(UnimplementedError('unfollow not stubbed'));

  @override
  Future<ProfileRelationship> mute(String handleOrDid) =>
      onMute?.call(handleOrDid) ??
      Future<ProfileRelationship>.error(UnimplementedError('mute not stubbed'));

  @override
  Future<ProfileRelationship> unmute(String handleOrDid) =>
      onUnmute?.call(handleOrDid) ??
      Future<ProfileRelationship>.error(
        UnimplementedError('unmute not stubbed'),
      );

  @override
  Future<ProfileRelationship> block(String handleOrDid) =>
      onBlock?.call(handleOrDid) ??
      Future<ProfileRelationship>.error(
        UnimplementedError('block not stubbed'),
      );

  @override
  Future<ProfileRelationship> unblock(String handleOrDid) =>
      onUnblock?.call(handleOrDid) ??
      Future<ProfileRelationship>.error(
        UnimplementedError('unblock not stubbed'),
      );

  @override
  Future<ReportResult> report(
    String handleOrDid,
    ReportSubmission submission,
  ) =>
      onReport?.call(handleOrDid, submission) ??
      Future<ReportResult>.error(UnimplementedError('report not stubbed'));

  @override
  Future<ProfileAccountPage> listMutualFollowers(
    String handleOrDid, {
    int? limit,
    String? cursor,
  }) =>
      onListMutualFollowers?.call(
        handleOrDid,
        limit: limit,
        cursor: cursor,
      ) ??
      Future<ProfileAccountPage>.error(
        UnimplementedError('listMutualFollowers not stubbed'),
      );

  @override
  Future<ProfileAccountPage> listFollowersMe({int? limit, String? cursor}) =>
      onListFollowersMe?.call(limit: limit, cursor: cursor) ??
      Future<ProfileAccountPage>.error(
        UnimplementedError('listFollowersMe not stubbed'),
      );

  @override
  Future<ProfileAccountPage> listFollowingMe({int? limit, String? cursor}) =>
      onListFollowingMe?.call(limit: limit, cursor: cursor) ??
      Future<ProfileAccountPage>.error(
        UnimplementedError('listFollowingMe not stubbed'),
      );

  @override
  Future<ProfileAccountPage> listMutedProfiles({int? limit, String? cursor}) =>
      onListMutedProfiles?.call(limit: limit, cursor: cursor) ??
      Future<ProfileAccountPage>.error(
        UnimplementedError('listMutedProfiles not stubbed'),
      );

  @override
  Future<ProfileAccountPage> listBlockedProfiles({
    int? limit,
    String? cursor,
  }) =>
      onListBlockedProfiles?.call(limit: limit, cursor: cursor) ??
      Future<ProfileAccountPage>.error(
        UnimplementedError('listBlockedProfiles not stubbed'),
      );
}
