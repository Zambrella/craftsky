import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/rich_text/faceted_text_model.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';

/// Builds styled spans for normalized facet ranges.
class FacetedTextSpanBuilder {
  /// Returns a [TextSpan] whose facet ranges use [facetColor].
  static TextSpan build({
    required String text,
    required List<NormalizedFacetRange> ranges,
    required TextStyle baseStyle,
    required Color facetColor,
    GestureRecognizer? Function(NormalizedFacetRange range)? recognizerForRange,
  }) {
    if (text.isEmpty) {
      return TextSpan(text: text, style: baseStyle);
    }

    final children = <TextSpan>[];
    var cursor = 0;
    for (final range in ranges) {
      if (range.charStart > cursor) {
        children.add(
          TextSpan(
            text: text.substring(cursor, range.charStart),
            style: baseStyle,
          ),
        );
      }
      final visibleText = text.substring(range.charStart, range.charEnd);
      children.add(
        TextSpan(
          text: _displayTextForRange(range, visibleText),
          style: baseStyle.copyWith(color: facetColor),
          recognizer: recognizerForRange?.call(range),
        ),
      );
      cursor = range.charEnd;
    }
    if (cursor < text.length) {
      children.add(TextSpan(text: text.substring(cursor), style: baseStyle));
    }

    return TextSpan(style: baseStyle, children: children);
  }
}

String _displayTextForRange(NormalizedFacetRange range, String visibleText) {
  return switch (range.feature) {
    LinkFacetFeature(uri: final uriText) => switch (normalizeExternalLinkUri(
      uriText,
    )) {
      final Uri uri => displayExternalLink(uri),
      null => visibleText,
    },
    _ => visibleText,
  };
}
