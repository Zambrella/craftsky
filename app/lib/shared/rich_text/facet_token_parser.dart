import 'dart:convert';

import 'package:craftsky_app/shared/rich_text/facet_syntax.dart';
import 'package:craftsky_app/shared/rich_text/faceted_text_model.dart';

/// Plain-text token detected by Craftsky's supported facet rules.
sealed class FacetToken {
  const FacetToken({
    required this.charStart,
    required this.charEnd,
  });

  /// Inclusive Dart string index start.
  final int charStart;

  /// Exclusive Dart string index end.
  final int charEnd;

  /// Converts this token to raw AT Protocol facet JSON for [text].
  Map<String, dynamic> toRawFacet(String text, {String? did}) {
    return {
      'index': {
        'byteStart': _byteOffset(text, charStart),
        'byteEnd': _byteOffset(text, charEnd),
      },
      'features': [toRawFeature(did: did)],
    };
  }

  /// Converts this token to raw AT Protocol facet feature metadata.
  Map<String, dynamic> toRawFeature({String? did}) {
    return switch (this) {
      MentionFacetToken() => FacetFeature.mention(did ?? '').toRawFeature(),
      LinkFacetToken(:final uri) => FacetFeature.link(uri).toRawFeature(),
      TagFacetToken(:final tag) => FacetFeature.tag(tag).toRawFeature(),
    };
  }
}

/// Plain-text mention token detected before the handle is resolved to a DID.
final class MentionFacetToken extends FacetToken {
  /// Creates a plain-text mention token.
  const MentionFacetToken({
    required super.charStart,
    required super.charEnd,
    required this.handle,
  });

  /// Mention handle without leading `@`.
  final String handle;
}

/// Plain-text link token detected before publish.
final class LinkFacetToken extends FacetToken {
  /// Creates a plain-text link token.
  const LinkFacetToken({
    required super.charStart,
    required super.charEnd,
    required this.uri,
  });

  /// Link URI written to the raw facet.
  final String uri;
}

/// Plain-text hashtag token detected before publish.
final class TagFacetToken extends FacetToken {
  /// Creates a plain-text hashtag token.
  const TagFacetToken({
    required super.charStart,
    required super.charEnd,
    required this.tag,
  });

  /// Hashtag value without leading `#`.
  final String tag;
}

List<FacetToken> detectSupportedFacetTokens(String text) {
  final tokens = <FacetToken>[];
  for (final match in facetMentionPattern.allMatches(text)) {
    final prefix = match.group(1) ?? '';
    final handle = match.group(2)!;
    final charStart = match.start + prefix.length;
    tokens.add(
      MentionFacetToken(
        charStart: charStart,
        charEnd: charStart + handle.length + 1,
        handle: handle,
      ),
    );
  }
  for (final match in facetLinkPattern.allMatches(text)) {
    final prefix = match.group(1) ?? '';
    final visibleLink = trimFacetLinkText(match.group(2)!);
    if (visibleLink.isEmpty) continue;
    final charStart = match.start + prefix.length;
    tokens.add(
      LinkFacetToken(
        charStart: charStart,
        charEnd: charStart + visibleLink.length,
        uri: _linkUri(visibleLink),
      ),
    );
  }
  for (final match in facetHashtagPattern.allMatches(text)) {
    final prefix = match.group(1) ?? '';
    final tag = match.group(2)!;
    final charStart = match.start + prefix.length;
    tokens.add(
      TagFacetToken(
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
  return detectSupportedFacetTokens(
    text,
  ).map((token) => token.toRawFacet(text)).toList();
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
