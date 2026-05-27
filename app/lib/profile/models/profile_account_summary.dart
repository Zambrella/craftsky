import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'profile_account_summary.mapper.dart';

@MappableClass(includeCustomMappers: [DidMapper(), HandleMapper()])
class ProfileAccountSummary with ProfileAccountSummaryMappable {
  ProfileAccountSummary({
    required String did,
    required String handle,
    required this.isCraftskyProfile,
    this.displayName,
    this.description,
    this.avatar,
  }) : did = Did.parse(did),
       handle = Handle.parse(handle);

  final Did did;
  final Handle handle;
  final bool isCraftskyProfile;
  final String? displayName;
  final String? description;
  final String? avatar;
}
