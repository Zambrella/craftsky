import 'package:craftsky_app/shared/rich_text/faceted_text_model.dart';
import 'package:craftsky_app/shared/rich_text/faceted_text_span_builder.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_action_providers.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Render-safe rich text for AT Protocol facet metadata.
class FacetedText extends ConsumerStatefulWidget {
  /// Creates faceted text from raw AT Protocol facet JSON.
  const FacetedText({
    required this.text,
    super.key,
    this.facets,
    this.style,
    this.textAlign,
    this.maxLines,
    this.overflow,
  });

  /// Plain visible text.
  final String text;

  /// Raw `app.bsky.richtext.facet`-compatible JSON.
  final List<Map<String, dynamic>>? facets;

  /// Base text style for non-faceted ranges.
  final TextStyle? style;

  /// Text alignment.
  final TextAlign? textAlign;

  /// Maximum number of display lines.
  final int? maxLines;

  /// Overflow behavior.
  final TextOverflow? overflow;

  @override
  ConsumerState<FacetedText> createState() => _FacetedTextState();
}

class _FacetedTextState extends ConsumerState<FacetedText> {
  final _recognizers = <GestureRecognizer>[];

  @override
  void dispose() {
    _disposeRecognizers();
    super.dispose();
  }

  void _disposeRecognizers() {
    for (final recognizer in _recognizers) {
      recognizer.dispose();
    }
    _recognizers.clear();
  }

  @override
  Widget build(BuildContext context) {
    _disposeRecognizers();
    final theme = Theme.of(context);
    final baseStyle =
        widget.style ?? theme.textTheme.bodyMedium ?? const TextStyle();
    final ranges = FacetedTextModel.fromRaw(
      text: widget.text,
      rawFacets: widget.facets,
    );
    final handler = ref.watch(facetActionHandlerProvider);
    final span = FacetedTextSpanBuilder.build(
      text: widget.text,
      ranges: ranges,
      baseStyle: baseStyle,
      facetColor: theme.colorScheme.primary,
      recognizerForRange: (range) {
        final visibleText = widget.text.substring(
          range.charStart,
          range.charEnd,
        );
        final recognizer = TapGestureRecognizer()
          ..onTap = () => handler.handle(
            context,
            feature: range.feature,
            visibleText: visibleText,
          );
        _recognizers.add(recognizer);
        return recognizer;
      },
    );

    return Text.rich(
      span,
      textAlign: widget.textAlign,
      maxLines: widget.maxLines,
      overflow: widget.overflow,
    );
  }
}
