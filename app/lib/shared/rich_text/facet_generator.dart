// Interface and constructor shapes intentionally mirror future injectable
// seams.
// ignore_for_file: one_member_abstracts, prefer_initializing_formals

import 'package:craftsky_app/shared/rich_text/facet_token_parser.dart';

/// Resolves visible Craftsky handles to DIDs without network access.
abstract interface class MentionResolver {
  /// Returns the DID for [handle], or `null` when the local resolver does not
  /// know the handle.
  Future<String?> didForHandle(String handle);
}

/// Generates raw AT Protocol rich-text facet JSON from final submitted text.
class FacetGenerator {
  /// Creates a generator backed by the injected local [mentionResolver].
  const FacetGenerator({required MentionResolver mentionResolver})
    : _mentionResolver = mentionResolver;

  final MentionResolver _mentionResolver;

  /// Generates non-overlapping facet maps using UTF-8 byte offsets.
  Future<List<Map<String, dynamic>>> generate(String text) async {
    final facets = <Map<String, dynamic>>[];

    for (final token in detectSupportedFacetTokens(text)) {
      switch (token) {
        case MentionFacetToken(:final handle):
          final did = await _mentionResolver.didForHandle(handle);
          if (did == null) {
            continue;
          }
          facets.add(token.toRawFacet(text, did: did));
        case LinkFacetToken():
          facets.add(token.toRawFacet(text));
        case TagFacetToken():
          facets.add(token.toRawFacet(text));
      }
    }

    return facets;
  }
}
