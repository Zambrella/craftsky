final class LinkPreviewCandidate {
  const LinkPreviewCandidate._({
    required this.identity,
    required this.sourceFragment,
  });

  factory LinkPreviewCandidate.parse(String source) {
    final trimmed = source.trim();
    final withScheme =
        RegExp(
          '^[a-zA-Z][a-zA-Z0-9+.-]*://',
        ).hasMatch(trimmed)
        ? trimmed
        : 'https://$trimmed';
    final parsed = Uri.parse(withScheme);
    final scheme = parsed.scheme.toLowerCase();
    if ((scheme != 'http' && scheme != 'https') ||
        parsed.host.isEmpty ||
        parsed.userInfo.isNotEmpty) {
      throw const FormatException('invalid link preview candidate');
    }

    final host = parsed.host.toLowerCase();
    final displayHost = host.contains(':') ? '[$host]' : host;
    final isDefaultPort =
        parsed.hasPort &&
        ((scheme == 'http' && parsed.port == 80) ||
            (scheme == 'https' && parsed.port == 443));
    final port = parsed.hasPort && !isDefaultPort ? ':${parsed.port}' : '';
    final query = parsed.hasQuery ? '?${parsed.query}' : '';
    final identity = Uri.parse(
      '$scheme://$displayHost$port${parsed.path}$query',
    );

    return LinkPreviewCandidate._(
      identity: identity,
      sourceFragment: parsed.hasFragment ? parsed.fragment : null,
    );
  }

  final Uri identity;
  final String? sourceFragment;

  Uri get transportUri => identity;

  Uri navigationUri(Uri appViewFinalUri) {
    final fragment = sourceFragment;
    if (appViewFinalUri.hasFragment || fragment == null || fragment.isEmpty) {
      return appViewFinalUri;
    }
    return appViewFinalUri.replace(fragment: fragment);
  }
}
