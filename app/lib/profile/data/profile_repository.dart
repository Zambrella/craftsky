import 'package:craftsky_app/profile/models/profile.dart';

/// Read/write surface the profile providers depend on. The production
/// binding is `ApiProfileRepository`; `DummyProfileRepository` provides
/// canned responses for running the app without a backend, and the
/// test suite swaps in `FakeProfileRepository` (under `test/`).
abstract interface class ProfileRepository {
  /// Fetches any user's profile by handle or DID.
  Future<Profile> fetch(String handleOrDid);

  /// Fetches the authenticated user's profile via `/v1/profiles/me`.
  Future<Profile> fetchMe();

  /// Patches the authenticated user's profile. `null` fields are
  /// omitted from the request — the AppView treats them as "leave
  /// unchanged".
  Future<Profile> updateMe({
    String? displayName,
    String? description,
    List<String>? crafts,
  });
}
