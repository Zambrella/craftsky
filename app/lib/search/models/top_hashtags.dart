import 'package:dart_mappable/dart_mappable.dart';

part 'top_hashtags.mapper.dart';

@MappableClass()
class TopHashtagsResponse with TopHashtagsResponseMappable {
  const new({required this.groups});

  final List<TopHashtagGroup> groups;
}

@MappableClass()
class TopHashtagGroup with TopHashtagGroupMappable {
  const new({required this.craftType, required this.items});

  final String craftType;
  final List<TopHashtagItem> items;
}

@MappableClass()
class TopHashtagItem with TopHashtagItemMappable {
  const new({required this.tag, required this.count});

  final String tag;
  final int count;
}
