import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';

/// Read/write surface the profile providers depend on. The production
/// binding is `ApiProfileRepository`; `DummyProfileRepository` provides
/// canned responses for running the app without a backend, and the
/// test suite swaps in `FakeProfileRepository` (under `test/`).
abstract interface class ProfileRepository {
  /// Fetches any user's profile by handle or DID.
  Future<Profile> fetch(String handleOrDid);

  /// Fetches the authenticated user's profile via `/v1/profiles/me`.
  Future<Profile> fetchMe();

  /// Replaces the authenticated user's profile.
  ///
  /// Callers should pass the **full** desired field values, not a diff.
  /// `null` fields are stripped from the wire body, but atproto record
  /// semantics mean any field absent from the request is cleared on the
  /// PDS — see `ProfileApiClient.updateMyProfile` for the gory detail.
  Future<Profile> updateMe({
    String? displayName,
    String? description,
    List<String>? crafts,
    UploadedBlob? avatar,
    bool clearAvatar = false,
    UploadedBlob? banner,
    bool clearBanner = false,
  });
}
