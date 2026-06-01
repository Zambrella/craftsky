// Interface and constructor shapes intentionally mirror future injectable
// seams.
// ignore_for_file: one_member_abstracts, prefer_initializing_formals

import 'dart:convert';

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
    final facets = <_GeneratedFacet>[];

    for (final match in _mentionPattern.allMatches(text)) {
      final prefix = match.group(1) ?? '';
      final handle = match.group(2)!;
      final charStart = match.start + prefix.length;
      final charEnd = charStart + handle.length + 1;
      final did = await _mentionResolver.didForHandle(handle);
      if (did == null) {
        continue;
      }
      facets.add(
        _GeneratedFacet(
          charStart: charStart,
          charEnd: charEnd,
          feature: {r'$type': 'app.bsky.richtext.facet#mention', 'did': did},
        ),
      );
    }

    for (final match in _linkPattern.allMatches(text)) {
      final prefix = match.group(1) ?? '';
      final visibleLink = _trimLinkText(match.group(2)!);
      if (visibleLink.isEmpty) {
        continue;
      }
      final charStart = match.start + prefix.length;
      final charEnd = charStart + visibleLink.length;
      facets.add(
        _GeneratedFacet(
          charStart: charStart,
          charEnd: charEnd,
          feature: {
            r'$type': 'app.bsky.richtext.facet#link',
            'uri': _linkUri(visibleLink),
          },
        ),
      );
    }

    for (final match in _hashtagPattern.allMatches(text)) {
      final prefix = match.group(1) ?? '';
      final tag = match.group(2)!;
      final charStart = match.start + prefix.length;
      final charEnd = charStart + tag.length + 1;
      facets.add(
        _GeneratedFacet(
          charStart: charStart,
          charEnd: charEnd,
          feature: {r'$type': 'app.bsky.richtext.facet#tag', 'tag': tag},
        ),
      );
    }

    facets.sort((a, b) {
      final startComparison = a.charStart.compareTo(b.charStart);
      if (startComparison != 0) {
        return startComparison;
      }
      return b.charEnd.compareTo(a.charEnd);
    });

    final nonOverlappingFacets = <_GeneratedFacet>[];
    var previousCharEnd = -1;
    for (final facet in facets) {
      if (facet.charStart < previousCharEnd) {
        continue;
      }
      nonOverlappingFacets.add(facet);
      previousCharEnd = facet.charEnd;
    }

    return nonOverlappingFacets.map((facet) => facet.toRaw(text)).toList();
  }
}

final _mentionPattern = RegExp(
  r'(^|[\s(\[{])@([A-Za-z0-9](?:[A-Za-z0-9.-]*[A-Za-z0-9])?\.[A-Za-z][A-Za-z0-9.-]*)',
);

final _linkPattern = RegExp(
  r'(^|[\s(\[{])((?:https?://)?(?:[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?\.)+[A-Za-z]{2,}(?:/[^\s]*)?)',
);

final _hashtagPattern = RegExp(
  r'(^|[^\p{L}\p{N}_])#([\p{L}\p{N}_]+)',
  unicode: true,
);

class _GeneratedFacet {
  const _GeneratedFacet({
    required this.charStart,
    required this.charEnd,
    required this.feature,
  });

  final int charStart;
  final int charEnd;
  final Map<String, dynamic> feature;

  Map<String, dynamic> toRaw(String text) {
    return {
      'index': {
        'byteStart': _byteOffset(text, charStart),
        'byteEnd': _byteOffset(text, charEnd),
      },
      'features': [feature],
    };
  }
}

int _byteOffset(String text, int charIndex) {
  return utf8.encode(text.substring(0, charIndex)).length;
}

String _linkUri(String visibleLink) {
  if (visibleLink.startsWith('http://') || visibleLink.startsWith('https://')) {
    return visibleLink;
  }
  return 'https://$visibleLink';
}

String _trimLinkText(String link) {
  var trimmed = link;
  while (trimmed.isNotEmpty &&
      _trailingSentencePunctuation.contains(trimmed[trimmed.length - 1])) {
    trimmed = trimmed.substring(0, trimmed.length - 1);
  }
  while (trimmed.endsWith(')') && _count(trimmed, ')') > _count(trimmed, '(')) {
    trimmed = trimmed.substring(0, trimmed.length - 1);
  }
  while (trimmed.endsWith(']') && _count(trimmed, ']') > _count(trimmed, '[')) {
    trimmed = trimmed.substring(0, trimmed.length - 1);
  }
  while (trimmed.endsWith('}') && _count(trimmed, '}') > _count(trimmed, '{')) {
    trimmed = trimmed.substring(0, trimmed.length - 1);
  }
  return trimmed;
}

const _trailingSentencePunctuation = {'.', ',', '!', '?', ';', ':'};

int _count(String text, String character) {
  return character.allMatches(text).length;
}
