import 'dart:convert';

const maxDiagnosticTextBytes = 2048;

final _credential = RegExp(
  r'''\b(token|access[_-]?token|refresh[_-]?token|session[_-]?token|service[_-]?token|password|client[_-]?secret|pkce[_-]?verifier|dpop|authorization|cookie|set-cookie|code|state|private[_-]?key|webhook[_-]?signature|dsn|x-amz-signature|x-goog-signature)\s*[:=]\s*(?:"[^"]*"|'[^']*'|[^\s,;]+)''',
  caseSensitive: false,
);
final _bearer = RegExp(r'\bbearer\s+[^\s,;]+', caseSensitive: false);

final _url = RegExp(r'''https?://[^\s<>"']+''', caseSensitive: false);
final _email = RegExp(
  r'[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}',
  caseSensitive: false,
);
final _localPath = RegExp(
  r'''(?:/(?:Users|home|private|tmp|var)/[^\s"']+|[A-Za-z]:\\[^\s"']+)''',
);
final _privateKey = RegExp(
  '-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----',
  dotAll: true,
);

/// Only call after reviewed provenance has admitted the text. This cannot
/// classify arbitrary private prose; unknown exception messages stay excluded.
String sanitizeKnownDiagnosticText(String sourceText) {
  var text = sourceText;
  for (var i = 0; i < 2; i++) {
    try {
      if (!text.contains('%')) break;
      final decoded = Uri.decodeComponent(text);
      if (decoded == text) break;
      text = decoded;
    } on FormatException {
      break;
      // Uri decoding reports malformed input with ArgumentError on some
      // runtimes.
      // ignore: avoid_catching_errors
    } on ArgumentError {
      break;
    }
  }
  final selected = text
      .replaceAll(_privateKey, '[REDACTED]')
      .replaceAllMapped(
        _url,
        (match) => '${Uri.tryParse(match[0]!)?.scheme ?? 'url'}://[REDACTED]',
      )
      .replaceAll(_email, '[REDACTED]')
      .replaceAll(_localPath, '[REDACTED]')
      .replaceAllMapped(_credential, (match) => '${match[1]}=[REDACTED]')
      .replaceAll(_bearer, 'Bearer [REDACTED]');
  final bytes = utf8.encode(selected);
  if (bytes.length <= maxDiagnosticTextBytes) return selected;
  const marker = '[TRUNCATED]';
  var end = maxDiagnosticTextBytes - marker.length;
  while (end > 0 && bytes[end] & 0xc0 == 0x80) {
    end--;
  }
  return '${utf8.decode(bytes.sublist(0, end))}$marker';
}

String boundDiagnosticText(String text, int limit) {
  final selected = sanitizeKnownDiagnosticText(text);
  final bytes = utf8.encode(selected);
  if (bytes.length <= limit) return selected;
  const marker = '[TRUNCATED]';
  var end = limit - marker.length;
  while (end > 0 && bytes[end] & 0xc0 == 0x80) {
    end--;
  }
  return '${utf8.decode(bytes.sublist(0, end))}$marker';
}
