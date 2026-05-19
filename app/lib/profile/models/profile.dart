import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'profile.mapper.dart';

/// Wire shape for `/v1/profiles/*` responses on the AppView.
///
/// `displayName`, `description`, `avatar`, `banner`, and `createdAt` may
/// be absent from the JSON; `crafts` is always present (empty list when
/// the user has none). `avatar` and `banner` are CDN URLs synthesised
/// server-side from blob CIDs — clients never see the raw blob.
@MappableClass(includeCustomMappers: [DidMapper(), HandleMapper()])
class Profile with ProfileMappable {
  Profile({
    required String did,
    required String handle,
    required this.crafts,
    this.displayName,
    this.description,
    this.avatar,
    this.banner,
    this.createdAt,
  }) : did = Did.parse(did),
       handle = Handle.parse(handle);

  final Did did;
  final Handle handle;
  final String? displayName;
  final String? description;
  final String? avatar;
  final String? banner;
  final List<String> crafts;
  final DateTime? createdAt;
}
