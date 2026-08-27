import 'package:craftsky_app/feed/composer/link_preview_candidate.dart';
import 'package:craftsky_app/shared/rich_text/facet_syntax.dart';
import 'package:craftsky_app/shared/rich_text/facet_token_parser.dart';

const maxLinkPreviewCandidates = 4;

List<LinkPreviewCandidate> deriveLinkPreviewCandidates(
  String text, {
  Uri? retainedIdentity,
}) {
  final candidates = <LinkPreviewCandidate>[];
  final identities = <Uri>{};
  for (final token in detectSupportedFacetTokens(text)) {
    if (token is! LinkFacetToken) continue;
    try {
      final candidate = LinkPreviewCandidate.parse(token.uri);
      if (!hasFacetCompletionBoundary(text, token.charEnd) &&
          candidate.identity != retainedIdentity) {
        continue;
      }
      if (!identities.add(candidate.identity)) {
        continue;
      }
      candidates.add(candidate);
      if (candidates.length == maxLinkPreviewCandidates) {
        break;
      }
    } on FormatException {
      continue;
    }
  }
  return candidates;
}
