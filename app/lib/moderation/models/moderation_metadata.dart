import 'package:dart_mappable/dart_mappable.dart';

part 'moderation_metadata.mapper.dart';

@MappableClass()
class ModerationMetadata with ModerationMetadataMappable {
  const ModerationMetadata({required this.warningKind});

  final String warningKind;
}
