import 'package:craftsky_app/shared/rich_text/facet_token_parser.dart';

const maxPostHashtags = 10;

Iterable<String> hashtagsInText(String text) =>
    detectSupportedFacetTokens(text)
        .whereType<TagFacetToken>()
        .map((token) => token.tag);

Set<String> acceptedHashtags(Iterable<String> tags) {
  final accepted = <String>{};
  for (final raw in tags) {
    final tag = raw.trim().toLowerCase();
    if (tag.isNotEmpty && accepted.length < maxPostHashtags) accepted.add(tag);
  }
  return accepted;
}

/// Preserve non-tag features and their byte offsets; refresh hashtag features
/// from visible text so removing an earlier tag can promote a later one.
List<Map<String, dynamic>> budgetedFacets(
  String text,
  List<Map<String, dynamic>>? facets,
  Set<String> accepted,
) {
  final result = <Map<String, dynamic>>[];
  for (final facet in facets ?? const <Map<String, dynamic>>[]) {
    final features = (facet['features'] as List?)
        ?.where(
          (feature) =>
              feature is! Map ||
              feature[r'$type'] != 'app.bsky.richtext.facet#tag',
        )
        .toList();
    if (features != null && features.isNotEmpty) {
      result.add({...facet, 'features': features});
    }
  }
  for (final token in detectSupportedFacetTokens(
    text,
  ).whereType<TagFacetToken>()) {
    if (!accepted.contains(token.tag.toLowerCase())) continue;
    final facet = token.toRawFacet(text);
    final start = (facet['index'] as Map)['byteStart'] as int;
    final end = (facet['index'] as Map)['byteEnd'] as int;
    // Do not introduce overlaps with retained links, mentions or extensions.
    if (result.any((other) {
      final index = other['index'] as Map;
      return start < (index['byteEnd'] as int) &&
          end > (index['byteStart'] as int);
    })) {
      continue;
    }
    result.add(facet);
  }
  result.sort(
    (a, b) => ((a['index'] as Map)['byteStart'] as int).compareTo(
      (b['index'] as Map)['byteStart'] as int,
    ),
  );
  return result;
}
