import 'dart:convert';

import 'package:craftsky_app/shared/rich_text/faceted_text_model.dart';

/// Supported render/generation token kind.
enum FacetTokenKind { mention, link, tag }

/// Plain-text token detected by Craftsky's supported facet rules.
class FacetToken {
  const FacetToken({
    required this.kind,
    required this.charStart,
    required this.charEnd,
    this.handle,
    this.uri,
    this.tag,
  });

  final FacetTokenKind kind;
  final int charStart;
  final int charEnd;
  final String? handle;
  final String? uri;
  final String? tag;

  Map<String, dynamic> toRawFacet(String text) {
    return {
      'index': {
        'byteStart': _byteOffset(text, charStart),
        'byteEnd': _byteOffset(text, charEnd),
      },
      'features': [toFeature()],
    };
  }

  FacetFeature toFeature() {
    switch (kind) {
      case FacetTokenKind.mention:
        return const FacetFeature.mention('');
      case FacetTokenKind.link:
        return FacetFeature.link(uri!);
      case FacetTokenKind.tag:
        return FacetFeature.tag(tag!);
    }
  }

  Map<String, dynamic> toRawFeature({String? did}) {
    switch (kind) {
      case FacetTokenKind.mention:
        return {r'$type': 'app.bsky.richtext.facet#mention', 'did': did ?? ''};
      case FacetTokenKind.link:
        return {r'$type': 'app.bsky.richtext.facet#link', 'uri': uri};
      case FacetTokenKind.tag:
        return {r'$type': 'app.bsky.richtext.facet#tag', 'tag': tag};
    }
  }
}

List<FacetToken> detectSupportedFacetTokens(String text) {
  final tokens = <FacetToken>[];
  for (final match in _mentionPattern.allMatches(text)) {
    final prefix = match.group(1) ?? '';
    final handle = match.group(2)!;
    final charStart = match.start + prefix.length;
    tokens.add(
      FacetToken(
        kind: FacetTokenKind.mention,
        charStart: charStart,
        charEnd: charStart + handle.length + 1,
        handle: handle,
      ),
    );
  }
  for (final match in _linkPattern.allMatches(text)) {
    final prefix = match.group(1) ?? '';
    final visibleLink = _trimLinkText(match.group(2)!);
    if (visibleLink.isEmpty) continue;
    final charStart = match.start + prefix.length;
    tokens.add(
      FacetToken(
        kind: FacetTokenKind.link,
        charStart: charStart,
        charEnd: charStart + visibleLink.length,
        uri: _linkUri(visibleLink),
      ),
    );
  }
  for (final match in _hashtagPattern.allMatches(text)) {
    final prefix = match.group(1) ?? '';
    final tag = match.group(2)!;
    final charStart = match.start + prefix.length;
    tokens.add(
      FacetToken(
        kind: FacetTokenKind.tag,
        charStart: charStart,
        charEnd: charStart + tag.length + 1,
        tag: tag,
      ),
    );
  }

  tokens.sort((a, b) {
    final startComparison = a.charStart.compareTo(b.charStart);
    if (startComparison != 0) return startComparison;
    return b.charEnd.compareTo(a.charEnd);
  });

  final nonOverlapping = <FacetToken>[];
  var previousCharEnd = -1;
  for (final token in tokens) {
    if (token.charStart < previousCharEnd) continue;
    nonOverlapping.add(token);
    previousCharEnd = token.charEnd;
  }
  return nonOverlapping;
}

List<Map<String, dynamic>> rawFacetsForPlainText(String text) {
  return detectSupportedFacetTokens(text).map((token) {
    return {
      'index': {
        'byteStart': _byteOffset(text, token.charStart),
        'byteEnd': _byteOffset(text, token.charEnd),
      },
      'features': [token.toRawFeature()],
    };
  }).toList();
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

int _count(String text, String character) => character.allMatches(text).length;
