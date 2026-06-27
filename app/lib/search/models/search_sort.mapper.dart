// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'search_sort.dart';

class SearchSortMapper extends EnumMapper<SearchSort> {
  SearchSortMapper._();

  static SearchSortMapper? _instance;
  static SearchSortMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SearchSortMapper._());
    }
    return _instance!;
  }

  static SearchSort fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  SearchSort decode(dynamic value) {
    switch (value) {
      case r'chronological':
        return SearchSort.chronological;
      case r'popular':
        return SearchSort.popular;
      default:
        throw MapperException.unknownEnumValue(value);
    }
  }

  @override
  dynamic encode(SearchSort self) {
    switch (self) {
      case SearchSort.chronological:
        return r'chronological';
      case SearchSort.popular:
        return r'popular';
    }
  }
}

extension SearchSortMapperExtension on SearchSort {
  String toValue() {
    SearchSortMapper.ensureInitialized();
    return MapperContainer.globals.toValue<SearchSort>(this) as String;
  }
}

