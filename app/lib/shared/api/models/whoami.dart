import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'whoami.mapper.dart';

@MappableClass(includeCustomMappers: [DidMapper(), HandleMapper()])
class WhoAmI with WhoAmIMappable {
  WhoAmI({required String did, required String handle})
    : did = Did.parse(did),
      handle = Handle.parse(handle);

  final Did did;
  final Handle handle;
}
