import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';
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
  FakeProfileRepository({this.onFetch, this.onFetchMe, this.onUpdateMe});

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
}
