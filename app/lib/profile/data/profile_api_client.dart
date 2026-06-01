import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:dio/dio.dart';

/// Profile-related AppView endpoints. Assumes the attached [Dio] has
/// the auth + error interceptors installed (see `dioProvider`); each
/// call is wrapped in `unwrapApi` so consumers see sealed
/// `ApiException` subtypes instead of raw `DioException`s.
class ProfileApiClient {
  const ProfileApiClient(this._dio);

  final Dio _dio;

  /// GET /v1/profiles/@{handleOrDid} — fetches any user's profile.
  /// `handleOrDid` is interpolated raw; the AppView accepts either an
  /// atproto handle (`alice.test`) or a DID (`did:plc:...`).
  Future<Profile> getProfile(String handleOrDid) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/profiles/@$handleOrDid',
    );
    return ProfileMapper.fromMap(res.data!);
  });

  /// GET /v1/profiles/me — fetches the authenticated user's profile.
  /// Equivalent to `getProfile(myDid)` but doesn't require the client
  /// to know its own DID up front.
  Future<Profile> getMyProfile() => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>('/v1/profiles/me');
    return ProfileMapper.fromMap(res.data!);
  });

  /// PUT /v1/profiles/me — updates the authenticated user's profile.
  ///
  /// **Callers should send the full desired field values, not a diff.**
  /// This method strips `null` fields from the body, but the AppView's
  /// PDS write path treats absent keys as cleared — atproto records
  /// are atomic, so a partial body produces a partial profile record.
  /// To leave a field unchanged, send its current value.
  ///
  /// Avatar and banner are not writable in v1.
  Future<Profile> updateMyProfile({
    String? displayName,
    String? description,
    List<String>? crafts,
    UploadedBlob? avatar,
    bool clearAvatar = false,
    UploadedBlob? banner,
    bool clearBanner = false,
  }) => unwrapApi(() async {
    final body = <String, dynamic>{
      'displayName': ?displayName,
      'description': ?description,
      'crafts': ?crafts,
    };
    if (clearAvatar) {
      body['avatar'] = null;
    } else if (avatar != null) {
      body['avatar'] = _blobToMap(avatar);
    }
    if (clearBanner) {
      body['banner'] = null;
    } else if (banner != null) {
      body['banner'] = _blobToMap(banner);
    }
    final res = await _dio.put<Map<String, dynamic>>(
      '/v1/profiles/me',
      data: body,
    );
    return ProfileMapper.fromMap(res.data!);
  });

  /// POST /v1/profiles/@{handleOrDid}/follows — follow a profile.
  Future<Profile> followProfile(String handleOrDid) => unwrapApi(() async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/v1/profiles/@$handleOrDid/follows',
    );
    return ProfileMapper.fromMap(res.data!);
  });

  /// DELETE /v1/profiles/@{handleOrDid}/follows — unfollow a profile.
  Future<Profile> unfollowProfile(String handleOrDid) => unwrapApi(() async {
    final res = await _dio.delete<Map<String, dynamic>>(
      '/v1/profiles/@$handleOrDid/follows',
    );
    return ProfileMapper.fromMap(res.data!);
  });

  /// POST /v1/profiles/@{handleOrDid}/reports — private AppView report intake.
  Future<ReportResult> reportProfile(
    String handleOrDid,
    ReportSubmission submission,
  ) => unwrapApi(() async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/v1/profiles/@$handleOrDid/reports',
      data: submission.toMap(),
    );
    return _reportResultFromMap(res.data!);
  });

  Future<ProfileAccountPage> listMutualFollowers(
    String handleOrDid, {
    int? limit,
    String? cursor,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/profiles/@$handleOrDid/mutual-followers',
      queryParameters: _pageQuery(limit: limit, cursor: cursor),
    );
    return ProfileAccountPageMapper.fromMap(res.data!);
  });

  Future<ProfileAccountPage> listFollowersMe({int? limit, String? cursor}) =>
      unwrapApi(() async {
        final res = await _dio.get<Map<String, dynamic>>(
          '/v1/profiles/me/followers',
          queryParameters: _pageQuery(limit: limit, cursor: cursor),
        );
        return ProfileAccountPageMapper.fromMap(res.data!);
      });

  Future<ProfileAccountPage> listFollowingMe({int? limit, String? cursor}) =>
      unwrapApi(() async {
        final res = await _dio.get<Map<String, dynamic>>(
          '/v1/profiles/me/following',
          queryParameters: _pageQuery(limit: limit, cursor: cursor),
        );
        return ProfileAccountPageMapper.fromMap(res.data!);
      });

  Map<String, dynamic> _pageQuery({int? limit, String? cursor}) => {
    'limit': ?limit,
    'cursor': ?cursor,
  };

  Map<String, dynamic> _blobToMap(UploadedBlob blob) => {
    r'$type': blob.type,
    'ref': {r'$link': blob.ref.link},
    'mimeType': blob.mimeType,
    'size': blob.size,
  };

  ReportResult _reportResultFromMap(Map<String, dynamic> data) {
    return ReportResult(
      reportId: data['reportId'] as String,
      status: data['status'] as String,
    );
  }
}
