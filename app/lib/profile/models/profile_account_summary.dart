import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'profile_account_summary.mapper.dart';

@MappableClass(
  includeCustomMappers: [
    DidMapper(),
    HandleMapper(),
    ProfileCustomisationMapper(),
  ],
)
class ProfileAccountSummary with ProfileAccountSummaryMappable {
  ProfileAccountSummary({
    required String did,
    required String handle,
    required this.isCraftskyProfile,
    this.displayName,
    this.description,
    this.avatar,
    this.muted = false,
    this.blocking = false,
    this.blockedBy = false,
    this.customisation = ProfileCustomisation.defaults,
  }) : did = Did.parse(did),
       handle = Handle.parse(handle);

  final Did did;
  final Handle handle;
  final bool isCraftskyProfile;
  final String? displayName;
  final String? description;
  final String? avatar;
  final bool muted;
  final bool blocking;
  final bool blockedBy;
  final ProfileCustomisation customisation;
}
