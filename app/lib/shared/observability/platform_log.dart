import 'dart:convert';

typedef PlatformLogSink = void Function(String record);

/// Flutter's iOS console truncates long individual print lines. Format the
/// already protected record for reading, then print each short line separately.
/// This also works in product mode, where dart:developer.log is disabled.
void writePlatformDiagnostic(String record) {
  // This is the logging framework's platform sink, not a feature print.
  // ignore: avoid_print
  platformDiagnosticLines(record).forEach(print);
}

/// No total record/stack budget: wrapping limits physical lines, never content.
/// Ordinary records remain pretty JSON. Exceptionally long string values wrap
/// across console lines too; joining those continuations recovers the value.
Iterable<String> platformDiagnosticLines(String record) sync* {
  final formatted = const JsonEncoder.withIndent('  ')
      .convert(jsonDecode(record));
  for (final line in const LineSplitter().convert(formatted)) {
    var buffer = StringBuffer();
    var bytes = 0;
    for (final rune in line.runes) {
      final width = rune <= 0x7f
          ? 1
          : rune <= 0x7ff
          ? 2
          : rune <= 0xffff
          ? 3
          : 4;
      if (bytes + width > 800) {
        yield buffer.toString();
        buffer = StringBuffer();
        bytes = 0;
      }
      buffer.writeCharCode(rune);
      bytes += width;
    }
    yield buffer.toString();
  }
}
