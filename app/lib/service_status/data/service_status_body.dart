import 'package:craftsky_app/service_status/models/service_status_document.dart';

Future<List<int>> readStatusBody(Stream<List<int>> stream) async {
  final bytes = <int>[];
  await for (final chunk in stream) {
    if (bytes.length + chunk.length > ServiceStatusDocument.maxBytes) {
      throw const FormatException('Status size');
    }
    bytes.addAll(chunk);
  }
  return bytes;
}
