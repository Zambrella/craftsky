import 'dart:convert';

/// AT Protocol rich-text facet feature type constants supported by Craftsky.
abstract final class FacetFeatureType {
  /// Mention facet feature type.
  static const mention = 'app.bsky.richtext.facet#mention';

  /// Link facet feature type.
  static const link = 'app.bsky.richtext.facet#link';

  /// Hashtag facet feature type.
  static const tag = 'app.bsky.richtext.facet#tag';
}

/// A supported facet feature selected from raw incoming facet metadata.
sealed class FacetFeature {
  /// Creates a supported facet feature.
  const FacetFeature();

  /// Creates a mention feature.
  const factory FacetFeature.mention(String did) = MentionFacetFeature;

  /// Creates a link feature.
  const factory FacetFeature.link(String uri) = LinkFacetFeature;

  /// Creates a hashtag feature.
  const factory FacetFeature.tag(String tag) = TagFacetFeature;

  /// Creates the first supported feature represented by [raw], or `null`.
  static FacetFeature? fromRaw(Map<dynamic, dynamic> raw) {
    switch (raw[r'$type']) {
      case FacetFeatureType.mention:
        final did = raw['did'];
        if (did is String) {
          return FacetFeature.mention(did);
        }
      case FacetFeatureType.link:
        final uri = raw['uri'];
        if (uri is String) {
          return FacetFeature.link(uri);
        }
      case FacetFeatureType.tag:
        final tag = raw['tag'];
        if (tag is String) {
          return FacetFeature.tag(tag);
        }
    }
    return null;
  }

  /// Converts this feature to raw AT Protocol rich-text facet metadata.
  Map<String, dynamic> toRawFeature() {
    return switch (this) {
      MentionFacetFeature(:final did) => {
        r'$type': FacetFeatureType.mention,
        'did': did,
      },
      LinkFacetFeature(:final uri) => {
        r'$type': FacetFeatureType.link,
        'uri': uri,
      },
      TagFacetFeature(:final tag) => {
        r'$type': FacetFeatureType.tag,
        'tag': tag,
      },
    };
  }
}

/// Mention facet feature data.
final class MentionFacetFeature extends FacetFeature {
  /// Creates mention facet feature data.
  const MentionFacetFeature(this.did);

  /// Mention DID.
  final String did;
}

/// Link facet feature data.
final class LinkFacetFeature extends FacetFeature {
  /// Creates link facet feature data.
  const LinkFacetFeature(this.uri);

  /// Link URI.
  final String uri;
}

/// Hashtag facet feature data.
final class TagFacetFeature extends FacetFeature {
  /// Creates hashtag facet feature data.
  const TagFacetFeature(this.tag);

  /// Hashtag value.
  final String tag;
}

/// A facet range normalized to Dart string character indices.
class NormalizedFacetRange {
  /// Creates a normalized range.
  const NormalizedFacetRange({
    required this.charStart,
    required this.charEnd,
    required this.feature,
  });

  /// Inclusive Dart string index start.
  final int charStart;

  /// Exclusive Dart string index end.
  final int charEnd;

  /// Supported feature for this range.
  final FacetFeature feature;
}

/// Converts raw AT Protocol facet JSON to render-safe ranges.
class FacetedTextModel {
  /// Parses and normalizes [rawFacets], dropping malformed entries safely.
  static List<NormalizedFacetRange> fromRaw({
    required String text,
    required List<Map<String, dynamic>>? rawFacets,
  }) {
    if (rawFacets == null || rawFacets.isEmpty) {
      return const [];
    }

    final byteToChar = _byteToCharIndexMap(text);
    final ranges = <_OrderedRange>[];
    for (var i = 0; i < rawFacets.length; i++) {
      final range = _rangeFromRaw(rawFacets[i], byteToChar, i);
      if (range != null) {
        ranges.add(range);
      }
    }

    ranges.sort((a, b) {
      final startComparison = a.range.charStart.compareTo(b.range.charStart);
      if (startComparison != 0) {
        return startComparison;
      }
      return a.order.compareTo(b.order);
    });

    final normalized = <NormalizedFacetRange>[];
    var previousEnd = -1;
    for (final ordered in ranges) {
      final range = ordered.range;
      if (range.charStart < previousEnd) {
        continue;
      }
      normalized.add(range);
      previousEnd = range.charEnd;
    }
    return normalized;
  }
}

_OrderedRange? _rangeFromRaw(
  Map<String, dynamic> raw,
  Map<int, int> byteToChar,
  int order,
) {
  final index = raw['index'];
  final features = raw['features'];
  if (index is! Map || features is! List) {
    return null;
  }

  final byteStart = index['byteStart'];
  final byteEnd = index['byteEnd'];
  if (byteStart is! int || byteEnd is! int || byteStart >= byteEnd) {
    return null;
  }

  final charStart = byteToChar[byteStart];
  final charEnd = byteToChar[byteEnd];
  if (charStart == null || charEnd == null) {
    return null;
  }

  final feature = _firstSupportedFeature(features);
  if (feature == null) {
    return null;
  }

  return _OrderedRange(
    order: order,
    range: NormalizedFacetRange(
      charStart: charStart,
      charEnd: charEnd,
      feature: feature,
    ),
  );
}

FacetFeature? _firstSupportedFeature(List<dynamic> features) {
  for (final feature in features) {
    if (feature is Map) {
      final parsed = FacetFeature.fromRaw(feature);
      if (parsed != null) return parsed;
    }
  }
  return null;
}

Map<int, int> _byteToCharIndexMap(String text) {
  final byteToChar = <int, int>{0: 0};
  var byteOffset = 0;
  var charOffset = 0;
  for (final rune in text.runes) {
    final character = String.fromCharCode(rune);
    byteOffset += utf8.encode(character).length;
    charOffset += character.length;
    byteToChar[byteOffset] = charOffset;
  }
  return byteToChar;
}

class _OrderedRange {
  const _OrderedRange({required this.order, required this.range});

  final int order;
  final NormalizedFacetRange range;
}
