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
    this.linkStyle,
    this.textAlign,
    this.maxLines,
    this.overflow,
    this.suffixText,
    this.actionLabel,
    this.onAction,
  });

  /// Plain visible text.
  final String text;

  /// Raw `app.bsky.richtext.facet`-compatible JSON.
  final List<Map<String, dynamic>>? facets;

  /// Base text style for non-faceted ranges.
  final TextStyle? style;

  /// Additional style applied only to link facets.
  final TextStyle? linkStyle;

  /// Text alignment.
  final TextAlign? textAlign;

  /// Maximum number of display lines.
  final int? maxLines;

  /// Overflow behavior.
  final TextOverflow? overflow;

  /// Plain text appended after the faceted content.
  final String? suffixText;

  /// Accessible label rendered as an inline action after [suffixText].
  final String? actionLabel;

  /// Invoked when the inline [actionLabel] is activated.
  final VoidCallback? onAction;

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
    final effectiveBaseStyle = DefaultTextStyle.of(context).style.merge(
      baseStyle,
    );
    final actionStyle = effectiveBaseStyle.copyWith(
      fontWeight: FontWeight.bold,
    );
    final ranges = FacetedTextModel.fromRaw(
      text: widget.text,
      rawFacets: widget.facets,
    );
    final handler = ref.watch(facetActionHandlerProvider);
    final contentSpan = FacetedTextSpanBuilder.build(
      text: widget.text,
      ranges: ranges,
      baseStyle: baseStyle,
      facetColor: theme.colorScheme.primary,
      linkStyle: widget.linkStyle,
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
    final actionLabel = widget.actionLabel;
    final onAction = widget.onAction;
    final hasAction = actionLabel != null && onAction != null;
    final span = widget.suffixText == null && !hasAction
        ? contentSpan
        : TextSpan(
            style: baseStyle,
            children: [
              contentSpan,
              if (widget.suffixText case final suffix?)
                TextSpan(text: suffix, style: baseStyle),
              if (hasAction)
                WidgetSpan(
                  alignment: PlaceholderAlignment.baseline,
                  baseline: TextBaseline.alphabetic,
                  child: TextButton(
                    style: TextButton.styleFrom(
                      padding: EdgeInsets.zero,
                      minimumSize: Size.zero,
                      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      textStyle: actionStyle,
                    ),
                    onPressed: onAction,
                    child: Text(
                      actionLabel,
                      style: actionStyle,
                      textScaler: TextScaler.noScaling,
                    ),
                  ),
                ),
            ],
          );

    return Text.rich(
      span,
      textAlign: widget.textAlign,
      maxLines: widget.maxLines,
      overflow: widget.overflow,
    );
  }
}
