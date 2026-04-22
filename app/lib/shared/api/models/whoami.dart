import 'package:dart_mappable/dart_mappable.dart';

part 'whoami.mapper.dart';

@MappableClass()
class WhoAmI with WhoAmIMappable {
  const WhoAmI({required this.did, required this.handle});

  final String did;
  final String handle;
}
