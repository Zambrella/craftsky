import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';

/// Canned [ProfileRepository] for running the app against no backend —
/// design previews, offline UI work, etc. Holds a single in-memory
/// profile so optimistic-update flows behave end-to-end (writes "stick"
/// for the lifetime of the repository instance).
///
/// Not intended for tests. Tests override the repository provider with
/// `FakeProfileRepository` (under `test/profile/fakes/`) which is
/// programmable per-call.
class DummyProfileRepository implements ProfileRepository {
  DummyProfileRepository({
    Profile? seed,
    Duration latency = const Duration(milliseconds: 200),
  }) : _profile = seed ?? _defaultSeed,
       _latency = latency;

  static const _defaultSeed = Profile(
    did: 'did:plc:dummycraftskyuser0000000',
    handle: 'dummy.craftsky.social',
    displayName: 'Dummy Crafter',
    description: 'A canned profile used when running without an AppView.',
    crafts: ['knitting', 'crochet'],
  );

  Profile _profile;
  final Duration _latency;

  @override
  Future<Profile> fetch(String handleOrDid) async {
    await Future<void>.delayed(_latency);
    return _profile;
  }

  @override
  Future<Profile> fetchMe() async {
    await Future<void>.delayed(_latency);
    return _profile;
  }

  @override
  Future<Profile> updateMe({
    String? displayName,
    String? description,
    List<String>? crafts,
  }) async {
    await Future<void>.delayed(_latency);
    return _profile = _profile.copyWith(
      displayName: displayName ?? _profile.displayName,
      description: description ?? _profile.description,
      crafts: crafts ?? _profile.crafts,
    );
  }
}
