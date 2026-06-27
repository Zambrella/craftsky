import 'package:dart_mappable/dart_mappable.dart';

part 'search_sort.mapper.dart';

/// Supported AppView search result sort values.
@MappableEnum()
enum SearchSort {
  chronological,
  popular;

  String get wireValue => name;
}
