typedef PlatformLogSink = void Function(String record);

// Flutter routes Dart print output to its platform console in release builds.
// The emitter supplies selected, sanitized, bounded JSON before this boundary.
// dart:developer.log is disabled by the Dart runtime in product mode.
// This is the platform sink for the logging framework, not a feature print.
// ignore: avoid_print
void writePlatformDiagnostic(String record) => print(record);
