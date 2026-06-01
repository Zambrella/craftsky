import 'dart:async';

import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/facet_autocomplete_controller.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

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
  final TextEditingController controller;

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
  Timer? _debounceTimer;
  ActiveFacetToken? _activeToken;
  List<AccountSuggestion>? _accountSuggestions;
  List<HashtagSuggestion>? _hashtagSuggestions;

  @override
  void dispose() {
    _debounceTimer?.cancel();
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
      return;
    }

    setState(() {
      _activeToken = token;
      _accountSuggestions = null;
      _hashtagSuggestions = null;
    });

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
      } else {
        final suggestions = await ref
            .read(hashtagSuggestionRepositoryProvider)
            .searchHashtags(token.query);
        if (!mounted || _activeToken != token) {
          return;
        }
        setState(() => _hashtagSuggestions = suggestions);
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
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        BrandTextField(
          label: widget.label,
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
        if (_activeToken?.kind == ActiveFacetTokenKind.mention &&
            _accountSuggestions != null)
          _MentionSuggestionList(
            suggestions: _accountSuggestions!,
            onSelected: _selectMention,
          ),
        if (_activeToken?.kind == ActiveFacetTokenKind.hashtag &&
            _hashtagSuggestions != null &&
            _hashtagSuggestions!.isNotEmpty)
          _HashtagSuggestionList(
            suggestions: _hashtagSuggestions!,
            onSelected: _selectHashtag,
          ),
      ],
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
      return Padding(
        padding: const EdgeInsets.only(top: 8),
        child: Text('No results', style: theme.textTheme.bodyMedium),
      );
    }

    return Card(
      margin: const EdgeInsets.only(top: 8),
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
      margin: const EdgeInsets.only(top: 8),
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
