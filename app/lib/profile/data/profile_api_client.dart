import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
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
  /// Only fields present in the body are written; passing `null` for a
  /// field omits it from the request, leaving the existing value
  /// untouched. Avatar and banner are not writable in v1.
  Future<Profile> updateMyProfile({
    String? displayName,
    String? description,
    List<String>? crafts,
  }) => unwrapApi(() async {
    final body = <String, dynamic>{
      if (displayName != null) 'displayName': displayName,
      if (description != null) 'description': description,
      if (crafts != null) 'crafts': crafts,
    };
    final res = await _dio.put<Map<String, dynamic>>(
      '/v1/profiles/me',
      data: body,
    );
    return ProfileMapper.fromMap(res.data!);
  });
}
