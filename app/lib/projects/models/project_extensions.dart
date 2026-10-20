import 'package:dart_mappable/dart_mappable.dart';

/// Preserve public future fields while allowing edits/clears of known fields.
class ProjectExtensionsHook extends MappingHook {
  const new(this.known);
  final List<String> known;

  @override
  Object? beforeDecode(Object? value) {
    if (value is! Map) return value;
    final map = Map<String, dynamic>.from(value);
    final extra = <String, dynamic>{
      for (final entry in map.entries)
        if (!known.contains(entry.key)) entry.key: entry.value,
    };
    return {...map, 'extraFields': extra};
  }

  @override
  Object? afterEncode(Object? value) {
    if (value is! Map) return value;
    final map = Map<String, dynamic>.from(value);
    final extra = map.remove('extraFields');
    return {
      if (extra is Map)
        for (final entry in extra.entries)
          if (entry.key is String && !known.contains(entry.key))
            entry.key as String: entry.value,
      ...map,
    };
  }
}
