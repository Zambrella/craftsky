import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'profile_search_page.mapper.dart';

@MappableClass(includeCustomMappers: [DidMapper(), HandleMapper()])
class ProfileSearchResult with ProfileSearchResultMappable {
  ProfileSearchResult({
    required String did,
    required String handle,
    required this.isCraftskyProfile,
    required this.viewerIsFollowing,
    this.crafts = const [],
    this.displayName,
    this.description,
    this.avatar,
  }) : did = Did.parse(did),
       handle = Handle.parse(handle);

  final Did did;
  final Handle handle;
  final bool isCraftskyProfile;
  final bool viewerIsFollowing;
  final List<String> crafts;
  final String? displayName;
  final String? description;
  final String? avatar;

  ProfileAccountSummary get summary => ProfileAccountSummary(
    did: did,
    handle: handle,
    isCraftskyProfile: isCraftskyProfile,
    displayName: displayName,
    description: description,
    avatar: avatar,
  );
}

@MappableClass(ignoreNull: true)
class ProfileSearchPage with ProfileSearchPageMappable {
  const ProfileSearchPage({required this.items, this.cursor});

  final List<ProfileSearchResult> items;
  final String? cursor;
}
