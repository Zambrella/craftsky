import 'dart:convert';
import 'dart:io';

/// Opt-in artifacts from synthetic selected sinks, after privacy assertions.
/// Normal test runs neither persist diagnostics nor contact a live collector.
void writeDiagnosticEvidence(
  String name, {
  required List<String> local,
  List<String> exported = const [],
}) {
  final directory = Platform.environment['CRAFTSKY_DIAGNOSTIC_EVIDENCE_DIR'];
  if (directory == null) return;
  File('$directory/$name.json').writeAsStringSync(
    const JsonEncoder.withIndent('  ').convert({
      'local': local,
      'exported': exported.map(jsonDecode).toList(),
    }),
  );
}
