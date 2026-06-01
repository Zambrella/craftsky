import 'package:flutter/widgets.dart';

/// Active autocomplete token kind.
enum ActiveFacetTokenKind { mention, hashtag }

/// A mention or hashtag token currently under the caret.
class ActiveFacetToken {
  /// Creates an active token.
  const ActiveFacetToken({
    required this.kind,
    required this.start,
    required this.end,
    required this.query,
  });

  /// Token kind.
  final ActiveFacetTokenKind kind;

  /// Inclusive token start, including `@` or `#`.
  final int start;

  /// Exclusive token end at the current caret position.
  final int end;

  /// Query text after `@` or `#`.
  final String query;
}

/// Pure helpers for facet autocomplete token behavior.
class FacetAutocompleteController {
  /// Detects an active `@` or `#` token at the collapsed caret.
  static ActiveFacetToken? detectActiveToken(TextEditingValue value) {
    final selection = value.selection;
    if (!selection.isCollapsed || selection.baseOffset < 0) {
      return null;
    }

    final caret = selection.baseOffset;
    if (caret == 0 || caret > value.text.length) {
      return null;
    }

    final textBeforeCaret = value.text.substring(0, caret);
    final atIndex = textBeforeCaret.lastIndexOf('@');
    final hashIndex = textBeforeCaret.lastIndexOf('#');
    final triggerIndex = atIndex > hashIndex ? atIndex : hashIndex;
    if (triggerIndex < 0) {
      return null;
    }

    final trigger = value.text[triggerIndex];
    if (!_hasValidBoundary(value.text, triggerIndex)) {
      return null;
    }

    final query = value.text.substring(triggerIndex + 1, caret);
    if (query.isEmpty || query.contains(RegExp(r'\s'))) {
      return null;
    }
    if (trigger == '#' && !_validHashtagQuery.hasMatch(query)) {
      return null;
    }
    if (trigger == '@' && !_validMentionQuery.hasMatch(query)) {
      return null;
    }

    return ActiveFacetToken(
      kind: trigger == '@'
          ? ActiveFacetTokenKind.mention
          : ActiveFacetTokenKind.hashtag,
      start: triggerIndex,
      end: caret,
      query: query,
    );
  }

  /// Replaces [token] in [current] and places the caret after the replacement.
  static TextEditingValue replaceActiveToken({
    required TextEditingValue current,
    required ActiveFacetToken token,
    required String replacementWithSingleTrailingSpace,
  }) {
    final replacement = '${replacementWithSingleTrailingSpace.trimRight()} ';
    final text = current.text.replaceRange(token.start, token.end, replacement);
    final caret = token.start + replacement.length;
    return TextEditingValue(
      text: text,
      selection: TextSelection.collapsed(offset: caret),
    );
  }
}

/// Debounces autocomplete lookups and ignores superseded scheduled queries.
class DebouncedFacetLookup<T> {
  /// Creates a lookup debouncer with injectable [debounce].
  DebouncedFacetLookup({required this.debounce});

  /// Delay before invoking the scheduled lookup.
  final Duration debounce;

  int _generation = 0;

  /// Schedules [lookup] after [debounce]. Returns `null` if superseded.
  Future<T?> schedule(Future<T> Function() lookup) async {
    final generation = ++_generation;
    await Future<void>.delayed(debounce);
    if (generation != _generation) {
      return null;
    }
    return lookup();
  }
}

bool _hasValidBoundary(String text, int triggerIndex) {
  if (triggerIndex == 0) {
    return true;
  }
  final previous = text[triggerIndex - 1];
  return previous.trim().isEmpty || _openingPunctuation.contains(previous);
}

const _openingPunctuation = {'(', '[', '{'};

final _validMentionQuery = RegExp(r'^[A-Za-z0-9._-]+$');
final _validHashtagQuery = RegExp(r'^[\p{L}\p{N}_]+$', unicode: true);
