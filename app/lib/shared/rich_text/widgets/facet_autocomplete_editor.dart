import 'dart:async';

import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/facet_autocomplete_controller.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Text controller that colors the active editable mention/hashtag token.
class FacetTextEditingController extends TextEditingController {
  /// Creates a facet-aware editing controller.
  FacetTextEditingController({super.text});

  @override
  TextSpan buildTextSpan({
    required BuildContext context,
    required bool withComposing,
    TextStyle? style,
  }) {
    final ranges = _editableFacetTokenRanges(value.text);
    if (ranges.isEmpty) {
      return super.buildTextSpan(
        context: context,
        style: style,
        withComposing: withComposing,
      );
    }

    final text = value.text;
    final baseStyle = style ?? DefaultTextStyle.of(context).style;
    final facetStyle = baseStyle.copyWith(
      color: Theme.of(context).colorScheme.primary,
    );
    final children = <InlineSpan>[];
    var cursor = 0;
    for (final range in ranges) {
      if (range.start > cursor) {
        children.add(TextSpan(text: text.substring(cursor, range.start)));
      }
      children.add(
        TextSpan(
          text: text.substring(range.start, range.end),
          style: facetStyle,
        ),
      );
      cursor = range.end;
    }
    if (cursor < text.length) {
      children.add(TextSpan(text: text.substring(cursor)));
    }

    return TextSpan(
      style: baseStyle,
      children: children,
    );
  }
}

class _EditableFacetTokenRange {
  const _EditableFacetTokenRange({required this.start, required this.end});

  final int start;
  final int end;
}

List<_EditableFacetTokenRange> _editableFacetTokenRanges(String text) {
  final ranges = <_EditableFacetTokenRange>[];
  var index = 0;
  while (index < text.length) {
    final trigger = text[index];
    if ((trigger == '@' || trigger == '#') &&
        _hasEditableTokenBoundary(text, index)) {
      final end = _editableTokenEnd(text, index + 1, trigger);
      if (end > index + 1) {
        ranges.add(_EditableFacetTokenRange(start: index, end: end));
        index = end;
        continue;
      }
    }
    index++;
  }

  for (final match in _editableLinkPattern.allMatches(text)) {
    final prefix = match.group(1) ?? '';
    final visibleLink = _trimEditableLinkText(match.group(2)!);
    if (visibleLink.isEmpty) {
      continue;
    }
    final start = match.start + prefix.length;
    ranges.add(
      _EditableFacetTokenRange(start: start, end: start + visibleLink.length),
    );
  }

  ranges.sort((a, b) {
    final startComparison = a.start.compareTo(b.start);
    if (startComparison != 0) {
      return startComparison;
    }
    return b.end.compareTo(a.end);
  });

  final nonOverlappingRanges = <_EditableFacetTokenRange>[];
  var previousEnd = -1;
  for (final range in ranges) {
    if (range.start < previousEnd) {
      continue;
    }
    nonOverlappingRanges.add(range);
    previousEnd = range.end;
  }

  return nonOverlappingRanges;
}

int _editableTokenEnd(String text, int start, String trigger) {
  var index = start;
  while (index < text.length) {
    final iterator = text.substring(index).runes.iterator;
    if (!iterator.moveNext()) return index;
    final char = String.fromCharCode(iterator.current);
    final isValid = trigger == '@'
        ? _editableMentionChar.hasMatch(char)
        : _editableHashtagChar.hasMatch(char);
    if (!isValid) {
      return index;
    }
    index += char.length;
  }
  return index;
}

bool _hasEditableTokenBoundary(String text, int triggerIndex) {
  if (triggerIndex == 0) {
    return true;
  }
  final previous = text[triggerIndex - 1];
  return previous.trim().isEmpty ||
      _editableOpeningPunctuation.contains(previous);
}

const _editableOpeningPunctuation = {'(', '[', '{'};
final _editableMentionChar = RegExp(r'^[A-Za-z0-9._-]$');
final _editableHashtagChar = RegExp(r'^[\p{L}\p{N}_]$', unicode: true);
final _editableLinkPattern = RegExp(
  r'(^|[\s(\[{])((?:https?://)?(?:[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?\.)+[A-Za-z]{2,}(?:/[^\s]*)?)',
);
const _editableTrailingSentencePunctuation = {'.', ',', '!', '?', ';', ':'};

String _trimEditableLinkText(String link) {
  var trimmed = link;
  while (trimmed.isNotEmpty &&
      _editableTrailingSentencePunctuation.contains(
        trimmed[trimmed.length - 1],
      )) {
    trimmed = trimmed.substring(0, trimmed.length - 1);
  }
  while (trimmed.endsWith(')') &&
      _countEditableCharacters(trimmed, ')') >
          _countEditableCharacters(trimmed, '(')) {
    trimmed = trimmed.substring(0, trimmed.length - 1);
  }
  while (trimmed.endsWith(']') &&
      _countEditableCharacters(trimmed, ']') >
          _countEditableCharacters(trimmed, '[')) {
    trimmed = trimmed.substring(0, trimmed.length - 1);
  }
  while (trimmed.endsWith('}') &&
      _countEditableCharacters(trimmed, '}') >
          _countEditableCharacters(trimmed, '{')) {
    trimmed = trimmed.substring(0, trimmed.length - 1);
  }
  return trimmed;
}

int _countEditableCharacters(String text, String character) {
  return character.allMatches(text).length;
}

/// Reusable editor with mention/hashtag autocomplete support.
class FacetAutocompleteEditor extends ConsumerStatefulWidget {
  /// Creates an autocomplete editor around the app's branded text field.
  const FacetAutocompleteEditor({
    required this.label,
    required this.controller,
    super.key,
    this.focusNode,
    this.hintText,
    this.minLines,
    this.maxLines = 1,
    this.enabled = true,
    this.errorText,
    this.helperText,
    this.helperAlignment = AlignmentDirectional.centerStart,
    this.keyboardType,
    this.textInputAction,
    this.onChanged,
  });

  /// Field label.
  final String label;

  /// Text controller owned by the parent surface.
  final FacetTextEditingController controller;

  /// Optional focus node owned by the parent surface.
  final FocusNode? focusNode;

  /// Placeholder text.
  final String? hintText;

  /// Minimum lines for multiline editors.
  final int? minLines;

  /// Maximum lines for multiline editors.
  final int? maxLines;

  /// Whether the field is enabled.
  final bool enabled;

  /// Optional error text below the field.
  final String? errorText;

  /// Optional helper text below the field.
  final String? helperText;

  /// Alignment for helper text.
  final AlignmentGeometry helperAlignment;

  /// Keyboard type for the inner text field.
  final TextInputType? keyboardType;

  /// Text input action for the inner text field.
  final TextInputAction? textInputAction;

  /// Parent change callback.
  final ValueChanged<String>? onChanged;

  @override
  ConsumerState<FacetAutocompleteEditor> createState() =>
      _FacetAutocompleteEditorState();
}

class _FacetAutocompleteEditorState
    extends ConsumerState<FacetAutocompleteEditor> {
  final _textFieldKey = GlobalKey();
  Timer? _debounceTimer;
  OverlayEntry? _suggestionOverlay;
  _SuggestionOverlayGeometry? _suggestionOverlayGeometry;
  ActiveFacetToken? _activeToken;
  List<AccountSuggestion>? _accountSuggestions;
  List<HashtagSuggestion>? _hashtagSuggestions;

  @override
  void dispose() {
    _debounceTimer?.cancel();
    _removeSuggestionOverlay();
    super.dispose();
  }

  Future<void> _onChanged(String text) async {
    widget.onChanged?.call(text);
    final token = FacetAutocompleteController.detectActiveToken(
      widget.controller.value,
    );
    _debounceTimer?.cancel();
    if (token == null) {
      setState(() {
        _activeToken = null;
        _accountSuggestions = null;
        _hashtagSuggestions = null;
      });
      _removeSuggestionOverlay();
      return;
    }

    setState(() {
      _activeToken = token;
      _accountSuggestions = null;
      _hashtagSuggestions = null;
    });
    _removeSuggestionOverlay();

    final debounce = ref.read(facetAutocompleteDebounceProvider);
    _debounceTimer = Timer(debounce, () async {
      if (token.kind == ActiveFacetTokenKind.mention) {
        final suggestions = await ref
            .read(accountSuggestionRepositoryProvider)
            .searchAccounts(token.query);
        if (!mounted || _activeToken != token) {
          return;
        }
        setState(() => _accountSuggestions = suggestions);
        _updateSuggestionOverlay();
      } else {
        final suggestions = await ref
            .read(hashtagSuggestionRepositoryProvider)
            .searchHashtags(token.query);
        if (!mounted || _activeToken != token) {
          return;
        }
        setState(() => _hashtagSuggestions = suggestions);
        _updateSuggestionOverlay();
      }
    });
  }

  void _selectMention(AccountSuggestion account) {
    final token = _activeToken;
    if (token == null) {
      return;
    }
    widget.controller.value = FacetAutocompleteController.replaceActiveToken(
      current: widget.controller.value,
      token: token,
      replacementWithSingleTrailingSpace: '@${account.handle} ',
    );
    widget.onChanged?.call(widget.controller.text);
    widget.focusNode?.requestFocus();
    setState(() {
      _activeToken = null;
      _accountSuggestions = null;
      _hashtagSuggestions = null;
    });
    _removeSuggestionOverlay();
  }

  void _selectHashtag(HashtagSuggestion hashtag) {
    final token = _activeToken;
    if (token == null) {
      return;
    }
    widget.controller.value = FacetAutocompleteController.replaceActiveToken(
      current: widget.controller.value,
      token: token,
      replacementWithSingleTrailingSpace: '#${hashtag.tag} ',
    );
    widget.onChanged?.call(widget.controller.text);
    widget.focusNode?.requestFocus();
    setState(() {
      _activeToken = null;
      _accountSuggestions = null;
      _hashtagSuggestions = null;
    });
    _removeSuggestionOverlay();
  }

  bool get _hasSuggestionSurface {
    return switch (_activeToken?.kind) {
      ActiveFacetTokenKind.mention => _accountSuggestions != null,
      ActiveFacetTokenKind.hashtag =>
        _hashtagSuggestions != null && _hashtagSuggestions!.isNotEmpty,
      null => false,
    };
  }

  void _updateSuggestionOverlay() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_hasSuggestionSurface) {
        _removeSuggestionOverlay();
        return;
      }

      final geometry = _computeSuggestionOverlayGeometry();
      if (geometry == null) {
        _removeSuggestionOverlay();
        return;
      }

      _suggestionOverlayGeometry = geometry;
      final overlay = Overlay.of(context);
      if (_suggestionOverlay == null) {
        _suggestionOverlay = OverlayEntry(
          builder: (_) => _SuggestionOverlay(
            geometry: _suggestionOverlayGeometry!,
            child: _buildSuggestionList(),
          ),
        );
        overlay.insert(_suggestionOverlay!);
      } else {
        _suggestionOverlay!.markNeedsBuild();
      }
    });
  }

  void _removeSuggestionOverlay() {
    _suggestionOverlay?.remove();
    _suggestionOverlay = null;
    _suggestionOverlayGeometry = null;
  }

  _SuggestionOverlayGeometry? _computeSuggestionOverlayGeometry() {
    final token = _activeToken;
    final textFieldContext = _textFieldKey.currentContext;
    final overlayBox = Overlay.of(context).context.findRenderObject();
    final textFieldBox = textFieldContext?.findRenderObject();
    if (token == null ||
        overlayBox is! RenderBox ||
        textFieldBox is! RenderBox) {
      return null;
    }

    final renderEditable = _findRenderEditable(textFieldBox);
    if (renderEditable == null) {
      return null;
    }

    final caretRect = renderEditable.getLocalRectForCaret(
      TextPosition(offset: token.start),
    );
    final tokenStart = overlayBox.globalToLocal(
      renderEditable.localToGlobal(caretRect.bottomLeft),
    );
    final overlayWidth = overlayBox.size.width;
    const viewportPadding = 8.0;
    final width = textFieldBox.size.width
        .clamp(
          0.0,
          overlayWidth - (viewportPadding * 2),
        )
        .toDouble();
    final maxLeft = overlayWidth - viewportPadding - width;
    final left = tokenStart.dx.clamp(viewportPadding, maxLeft).toDouble();

    return _SuggestionOverlayGeometry(
      left: left,
      top: tokenStart.dy + viewportPadding,
      width: width,
      maxHeight: overlayBox.size.height - tokenStart.dy - (viewportPadding * 2),
    );
  }

  Widget _buildSuggestionList() {
    return switch (_activeToken?.kind) {
      ActiveFacetTokenKind.mention => _MentionSuggestionList(
        suggestions: _accountSuggestions ?? const [],
        onSelected: _selectMention,
      ),
      ActiveFacetTokenKind.hashtag => _HashtagSuggestionList(
        suggestions: _hashtagSuggestions ?? const [],
        onSelected: _selectHashtag,
      ),
      null => const SizedBox.shrink(),
    };
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        BrandTextField(
          label: widget.label,
          textFieldKey: _textFieldKey,
          controller: widget.controller,
          focusNode: widget.focusNode,
          hintText: widget.hintText,
          minLines: widget.minLines,
          maxLines: widget.maxLines,
          enabled: widget.enabled,
          errorText: widget.errorText,
          helperText: widget.helperText,
          helperAlignment: widget.helperAlignment,
          keyboardType: widget.keyboardType,
          textInputAction: widget.textInputAction,
          onChanged: _onChanged,
        ),
      ],
    );
  }
}

RenderEditable? _findRenderEditable(RenderObject root) {
  if (root is RenderEditable) {
    return root;
  }
  RenderEditable? match;
  root.visitChildren((child) {
    match ??= _findRenderEditable(child);
  });
  return match;
}

class _SuggestionOverlayGeometry {
  const _SuggestionOverlayGeometry({
    required this.left,
    required this.top,
    required this.width,
    required this.maxHeight,
  });

  final double left;
  final double top;
  final double width;
  final double maxHeight;
}

class _SuggestionOverlay extends StatelessWidget {
  const _SuggestionOverlay({required this.geometry, required this.child});

  final _SuggestionOverlayGeometry geometry;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Positioned(
      left: geometry.left,
      top: geometry.top,
      width: geometry.width,
      child: ConstrainedBox(
        constraints: BoxConstraints(maxHeight: geometry.maxHeight),
        child: SingleChildScrollView(child: child),
      ),
    );
  }
}

class _MentionSuggestionList extends StatelessWidget {
  const _MentionSuggestionList({
    required this.suggestions,
    required this.onSelected,
  });

  final List<AccountSuggestion> suggestions;
  final ValueChanged<AccountSuggestion> onSelected;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (suggestions.isEmpty) {
      return Card(
        margin: EdgeInsets.zero,
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Text('No results', style: theme.textTheme.bodyMedium),
        ),
      );
    }

    return Card(
      margin: EdgeInsets.zero,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          for (final suggestion in suggestions)
            InkWell(
              onTap: () => onSelected(suggestion),
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Row(
                  children: [
                    Semantics(
                      label: _avatarLabel(suggestion),
                      container: true,
                      child: ExcludeSemantics(
                        child: CircleAvatar(
                          child: Text(
                            (suggestion.displayName ?? suggestion.handle)
                                .characters
                                .first
                                .toUpperCase(),
                          ),
                        ),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(suggestion.displayName ?? suggestion.handle),
                          Text('@${suggestion.handle}'),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }

  String _avatarLabel(AccountSuggestion suggestion) {
    return 'Avatar for ${suggestion.displayName ?? suggestion.handle}';
  }
}

class _HashtagSuggestionList extends StatelessWidget {
  const _HashtagSuggestionList({
    required this.suggestions,
    required this.onSelected,
  });

  final List<HashtagSuggestion> suggestions;
  final ValueChanged<HashtagSuggestion> onSelected;

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: EdgeInsets.zero,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          for (final suggestion in suggestions)
            InkWell(
              onTap: () => onSelected(suggestion),
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Row(
                  children: [
                    Expanded(child: Text('#${suggestion.tag}')),
                    Text(
                      '${suggestion.postsLast28Days} posts in the last 28 days',
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }
}
