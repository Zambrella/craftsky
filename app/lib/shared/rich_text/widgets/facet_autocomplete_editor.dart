import 'dart:async';
import 'dart:math' as math;

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_handle.dart';
import 'package:craftsky_app/profile/widgets/compact_profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/facet_autocomplete_controller.dart';
import 'package:craftsky_app/shared/rich_text/facet_syntax.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

/// Text controller that colors the active editable mention/hashtag token.
class FacetTextEditingController extends TextEditingController {
  /// Creates a facet-aware editing controller.
  new({super.text});

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

    return TextSpan(style: baseStyle, children: children);
  }
}

class _EditableFacetTokenRange {
  const new({required this.start, required this.end});

  final int start;
  final int end;
}

List<_EditableFacetTokenRange> _editableFacetTokenRanges(String text) {
  final ranges = <_EditableFacetTokenRange>[];
  var index = 0;
  while (index < text.length) {
    final trigger = text[index];
    if ((trigger == '@' || trigger == '#') &&
        hasFacetTokenBoundary(text, index)) {
      final end = _editableTokenEnd(text, index + 1, trigger);
      if (end > index + 1) {
        ranges.add(_EditableFacetTokenRange(start: index, end: end));
        index = end;
        continue;
      }
    }
    index++;
  }

  for (final match in facetLinkPattern.allMatches(text)) {
    final prefix = match.group(1) ?? '';
    final visibleLink = trimFacetLinkText(match.group(2)!);
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
        ? isEditableMentionChar(char)
        : isEditableHashtagChar(char);
    if (!isValid) {
      return index;
    }
    index += char.length;
  }
  return index;
}

/// Reusable editor with mention/hashtag autocomplete support.
class FacetAutocompleteEditor extends ConsumerStatefulWidget {
  /// Creates an autocomplete editor around the app's branded text field.
  const new({
    required this.label,
    required this.controller,
    super.key,
    this.focusNode,
    this.hintText,
    this.minLines,
    this.maxLines = 1,
    this.maxLength,
    this.maxBytes,
    this.enabled = true,
    this.errorText,
    this.helperText,
    this.betweenLabelAndField,
    this.helperAlignment = AlignmentDirectional.centerStart,
    this.suffixIcon,
    this.keyboardType,
    this.textInputAction,
    this.textCapitalization = TextCapitalization.sentences,
    this.onSubmitted,
    this.onChanged,
    this.allowedTokenKinds,
    this.required = false,
    this.requiredLabel = 'required',
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

  final int? maxLength;
  final int? maxBytes;

  /// Whether the field is enabled.
  final bool enabled;

  /// Optional error text below the field.
  final String? errorText;

  /// Optional helper text below the field.
  final String? helperText;

  /// Optional content rendered between the label and the text field.
  final Widget? betweenLabelAndField;

  /// Alignment for helper text.
  final AlignmentGeometry helperAlignment;

  /// Optional trailing widget inside the field decoration.
  final Widget? suffixIcon;

  /// Keyboard type for the inner text field.
  final TextInputType? keyboardType;

  /// Text input action for the inner text field.
  final TextInputAction? textInputAction;

  /// Keyboard capitalization for prose or short names containing facets.
  final TextCapitalization textCapitalization;

  /// Parent submitted callback.
  final ValueChanged<String>? onSubmitted;

  /// Parent change callback.
  final ValueChanged<String>? onChanged;

  /// Autocomplete token kinds this editor should respond to.
  final Set<ActiveFacetTokenKind>? allowedTokenKinds;

  /// Whether the label should display the required marker.
  final bool required;

  /// Required marker text rendered beside the label.
  final String requiredLabel;

  @override
  ConsumerState<FacetAutocompleteEditor> createState() =>
      _FacetAutocompleteEditorState();
}

class _FacetAutocompleteEditorState
    extends ConsumerState<FacetAutocompleteEditor>
    with WidgetsBindingObserver {
  static const _suggestionOverlayBaseWidth = 300.0;
  static const _maxDisplayedSuggestions = 5;

  final GlobalKey _textFieldKey = GlobalKey();
  FocusNode? _internalFocusNode;
  late FocusNode _focusNode;
  Timer? _debounceTimer;
  OverlayEntry? _suggestionOverlay;
  _SuggestionOverlayGeometry? _suggestionOverlayGeometry;
  ActiveFacetToken? _activeToken;
  List<AccountSuggestion>? _accountSuggestions;
  List<HashtagSuggestion>? _hashtagSuggestions;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _focusNode = widget.focusNode ?? (_internalFocusNode = FocusNode());
    _focusNode.addListener(_onFocusChanged);
  }

  @override
  void didUpdateWidget(covariant FacetAutocompleteEditor oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.focusNode != widget.focusNode) {
      _focusNode.removeListener(_onFocusChanged);
      _focusNode = widget.focusNode ?? (_internalFocusNode ??= FocusNode());
      _focusNode.addListener(_onFocusChanged);
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _debounceTimer?.cancel();
    _focusNode.removeListener(_onFocusChanged);
    _internalFocusNode?.dispose();
    _removeSuggestionOverlay();
    super.dispose();
  }

  @override
  void didChangeMetrics() {
    if (_hasSuggestionSurface) _updateSuggestionOverlay();
  }

  void _onFocusChanged() {
    if (_focusNode.hasFocus) {
      return;
    }
    _debounceTimer?.cancel();
    setState(() {
      _activeToken = null;
      _accountSuggestions = null;
      _hashtagSuggestions = null;
    });
    _removeSuggestionOverlay();
  }

  Future<void> _onChanged(String text) async {
    widget.onChanged?.call(text);
    final token = FacetAutocompleteController.detectActiveToken(
      widget.controller.value,
    );
    _debounceTimer?.cancel();
    if (token == null ||
        !(widget.allowedTokenKinds?.contains(token.kind) ?? true)) {
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

  bool _acceptsReplacement(String text) =>
      (widget.maxLength == null ||
          text.characters.length <= widget.maxLength!) &&
      (widget.maxBytes == null ||
          RecordTextLimit(
            text.characters.length,
            widget.maxBytes!,
          ).accepts(text));

  void _selectMention(AccountSuggestion account) {
    final token = _activeToken;
    final alias = ProfileHandle(account.handle).aliasInput;
    if (token == null || alias == null) {
      return;
    }
    final replacement = FacetAutocompleteController.replaceActiveToken(
      current: widget.controller.value,
      token: token,
      replacementWithSingleTrailingSpace: '@$alias ',
    );
    if (!_acceptsReplacement(replacement.text)) return;
    widget.controller.value = replacement;
    widget.onChanged?.call(widget.controller.text);
    _focusNode.requestFocus();
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
    final replacement = FacetAutocompleteController.replaceActiveToken(
      current: widget.controller.value,
      token: token,
      replacementWithSingleTrailingSpace: '#${hashtag.tag} ',
    );
    if (!_acceptsReplacement(replacement.text)) return;
    widget.controller.value = replacement;
    widget.onChanged?.call(widget.controller.text);
    _focusNode.requestFocus();
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
    final availableWidth = math.max<double>(
      0,
      overlayWidth - (viewportPadding * 2),
    );
    final width = math.min(
      MediaQuery.textScalerOf(context).scale(_suggestionOverlayBaseWidth),
      availableWidth,
    );
    final maxLeft = overlayWidth - viewportPadding - width;
    final left = tokenStart.dx
        .clamp(math.min(viewportPadding, maxLeft), maxLeft)
        .toDouble();

    // Read the view itself: sheets and scaffolds may remove viewInsets from
    // their descendant MediaQuery even though this overlay spans the keyboard.
    final view = View.of(context);
    final viewData = MediaQueryData.fromView(view);
    final visibleTop = overlayBox
        .globalToLocal(
          Offset(0, viewData.padding.top),
        )
        .dy;
    final visibleBottom = overlayBox
        .globalToLocal(
          Offset(
            0,
            viewData.size.height -
                math.max(
                  viewData.viewInsets.bottom,
                  viewData.padding.bottom,
                ),
          ),
        )
        .dy;
    final top = math.max(0, visibleTop) + viewportPadding;
    final bottom =
        math.min(overlayBox.size.height, visibleBottom) - viewportPadding;
    final caretTop = overlayBox
        .globalToLocal(
          renderEditable.localToGlobal(caretRect.topLeft),
        )
        .dy;
    final above = (caretTop - viewportPadding)
        .clamp(top, math.max(top, bottom))
        .toDouble();
    final below = (tokenStart.dy + viewportPadding)
        .clamp(top, math.max(top, bottom))
        .toDouble();
    if (bottom <= top) return null;

    return _SuggestionOverlayGeometry(
      left: left,
      width: width,
      top: top,
      bottom: bottom,
      above: above,
      below: below,
    );
  }

  Widget _buildSuggestionList() {
    return switch (_activeToken?.kind) {
      ActiveFacetTokenKind.mention => _MentionSuggestionList(
        suggestions: (_accountSuggestions ?? const [])
            .take(_maxDisplayedSuggestions)
            .toList(),
        onSelected: _selectMention,
      ),
      ActiveFacetTokenKind.hashtag => _HashtagSuggestionList(
        suggestions: (_hashtagSuggestions ?? const [])
            .take(_maxDisplayedSuggestions)
            .toList(),
        onSelected: _selectHashtag,
      ),
      null => const SizedBox.shrink(),
    };
  }

  @override
  Widget build(BuildContext context) {
    if (_hasSuggestionSurface) _updateSuggestionOverlay();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        BrandTextField(
          label: widget.label,
          textFieldKey: _textFieldKey,
          controller: widget.controller,
          focusNode: _focusNode,
          hintText: widget.hintText,
          minLines: widget.minLines,
          maxLines: widget.maxLines,
          maxLength: widget.maxLength,
          maxBytes: widget.maxBytes,
          enabled: widget.enabled,
          errorText: widget.errorText,
          helperText: widget.helperText,
          betweenLabelAndField: widget.betweenLabelAndField,
          helperAlignment: widget.helperAlignment,
          suffixIcon: widget.suffixIcon,
          required: widget.required,
          requiredLabel: widget.requiredLabel,
          keyboardType: widget.keyboardType,
          textInputAction: widget.textInputAction,
          textCapitalization: widget.textCapitalization,
          onSubmitted: widget.onSubmitted,
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
  const new({
    required this.left,
    required this.width,
    required this.top,
    required this.bottom,
    required this.above,
    required this.below,
  });

  final double left;
  final double width;
  final double top;
  final double bottom;
  final double above;
  final double below;

  double get spaceAbove => above - top;
  double get spaceBelow => bottom - below;
}

class _SuggestionOverlay extends StatelessWidget {
  const new({required this.geometry, required this.child});

  final _SuggestionOverlayGeometry geometry;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Positioned.fill(
      child: CustomSingleChildLayout(
        delegate: _SuggestionOverlayLayout(geometry),
        child: SingleChildScrollView(child: child),
      ),
    );
  }
}

class _SuggestionOverlayLayout extends SingleChildLayoutDelegate {
  const new(this.geometry);

  final _SuggestionOverlayGeometry geometry;

  @override
  BoxConstraints getConstraintsForChild(BoxConstraints constraints) {
    return BoxConstraints(
      minWidth: geometry.width,
      maxWidth: geometry.width,
      maxHeight: math.max(geometry.spaceAbove, geometry.spaceBelow),
    );
  }

  @override
  Offset getPositionForChild(Size size, Size childSize) {
    // Prefer below when the actual list fits; otherwise use the larger side.
    final below =
        childSize.height <= geometry.spaceBelow ||
        geometry.spaceBelow >= geometry.spaceAbove;
    return Offset(
      geometry.left,
      below ? geometry.below : geometry.above - childSize.height,
    );
  }

  @override
  bool shouldRelayout(_SuggestionOverlayLayout oldDelegate) =>
      geometry != oldDelegate.geometry;
}

class _MentionSuggestionList extends StatelessWidget {
  const new({
    required this.suggestions,
    required this.onSelected,
  });

  final List<AccountSuggestion> suggestions;
  final ValueChanged<AccountSuggestion> onSelected;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final unavailableHandleLabel =
        Localizations.of<AppLocalizations>(
          context,
          AppLocalizations,
        )?.handleUnavailable ??
        'Handle unavailable';
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
              onTap: ProfileHandle(suggestion.handle).isAliasEligible
                  ? () => onSelected(suggestion)
                  : null,
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Row(
                  children: [
                    Semantics(
                      label: _avatarLabel(suggestion, unavailableHandleLabel),
                      container: true,
                      child: ExcludeSemantics(
                        child: ProfileAvatar(
                          avatarUrl: suggestion.avatar,
                          size: ProfileAvatarSize.small,
                          seed: ProfileHandle(suggestion.handle).displayLabel(
                            displayName: suggestion.displayName,
                            unavailableLabel: unavailableHandleLabel,
                          ),
                        ),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: CompactProfileIdentity(
                        handle: suggestion.handle,
                        displayName: suggestion.displayName,
                        crafts: suggestion.crafts,
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

  String _avatarLabel(
    AccountSuggestion suggestion,
    String unavailableHandleLabel,
  ) {
    final label = ProfileHandle(suggestion.handle).displayLabel(
      displayName: suggestion.displayName,
      unavailableLabel: unavailableHandleLabel,
    );
    return 'Avatar for $label';
  }
}

class _HashtagSuggestionList extends StatelessWidget {
  const new({
    required this.suggestions,
    required this.onSelected,
  });

  final List<HashtagSuggestion> suggestions;
  final ValueChanged<HashtagSuggestion> onSelected;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final popularityStyle = theme.textTheme.bodyMedium?.copyWith(
      color: theme.colorScheme.onSurfaceVariant,
    );

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
                    Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(
                          CraftskyIcons.trending,
                          size: 16,
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                        const SizedBox(width: 4),
                        Text(
                          '${suggestion.postCount} posts',
                          style: popularityStyle,
                        ),
                      ],
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
