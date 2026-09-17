import 'dart:io';

final class DartSourceFile {
  DartSourceFile(File file)
    : path = file.path.replaceAll(Platform.pathSeparator, '/'),
      source = file.readAsStringSync();

  final String path;
  final String source;
}

List<DartSourceFile> scanDartSources(
  String root, {
  bool recursive = true,
  bool excludeGenerated = true,
}) {
  final files =
      Directory(root)
          .listSync(recursive: recursive)
          .whereType<File>()
          .where((file) => file.path.endsWith('.dart'))
          .where(
            (file) =>
                !excludeGenerated ||
                (!file.path.endsWith('.g.dart') &&
                    !file.path.endsWith('.mapper.dart') &&
                    !file.path.endsWith('.freezed.dart')),
          )
          .map(DartSourceFile.new)
          .toList()
        ..sort((left, right) => left.path.compareTo(right.path));
  return files;
}

List<String> forbiddenSourceMatches(
  Iterable<DartSourceFile> files,
  Iterable<Pattern> forbidden,
) => [
  for (final file in files)
    for (final pattern in forbidden)
      if (pattern.allMatches(file.source).isNotEmpty) '${file.path}: $pattern',
];
