// Shared rich-text facet syntax rules for parsing, editing, and actions.

/// Opening punctuation that can precede an editable mention/link/hashtag token.
const facetOpeningPunctuation = {'(', '[', '{'};

/// Sentence punctuation trimmed from the visible end of detected links.
const facetTrailingSentencePunctuation = {'.', ',', '!', '?', ';', ':'};

/// Detects final-text mention tokens with a valid handle capture in group 2.
final facetMentionPattern = RegExp(
  r'(^|[\s(\[{])@([A-Za-z0-9](?:[A-Za-z0-9.-]*[A-Za-z0-9])?\.[A-Za-z][A-Za-z0-9.-]*)',
);

/// Detects final-text web link tokens with the visible link capture in group 2.
final facetLinkPattern = RegExp(
  r'(^|[\s(\[{])((?:https?://)?(?:[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?\.)+[A-Za-z]{2,}(?:/[^\s]*)?)',
);

/// Detects final-text hashtag tokens with the tag capture in group 2.
final facetHashtagPattern = RegExp(
  r'(^|[^\p{L}\p{N}_])#([\p{L}\p{N}_]+)',
  unicode: true,
);

final _editableMentionChar = RegExp(r'^[A-Za-z0-9._-]$');
final _editableHashtagChar = RegExp(r'^[\p{L}\p{N}_]$', unicode: true);
final _validMentionQuery = RegExp(r'^[A-Za-z0-9._-]+$');
final _validHashtagQuery = RegExp(r'^[\p{L}\p{N}_]+$', unicode: true);
final _validMentionHandle = RegExp(
  r'^[A-Za-z0-9][A-Za-z0-9.-]*\.[A-Za-z][A-Za-z0-9-]*$',
);
final _whitespace = RegExp(r'\s');

/// Returns whether [triggerIndex] is at a valid facet token boundary in [text].
bool hasFacetTokenBoundary(String text, int triggerIndex) {
  if (triggerIndex == 0) {
    return true;
  }
  final previous = text[triggerIndex - 1];
  return previous.trim().isEmpty || facetOpeningPunctuation.contains(previous);
}

/// Returns whether [char] can appear in an editable mention query.
bool isEditableMentionChar(String char) => _editableMentionChar.hasMatch(char);

/// Returns whether [char] can appear in an editable hashtag query.
bool isEditableHashtagChar(String char) => _editableHashtagChar.hasMatch(char);

/// Returns whether [query] is a valid partial mention query.
bool isValidMentionQuery(String query) => _validMentionQuery.hasMatch(query);

/// Returns whether [query] is a valid partial hashtag query.
bool isValidHashtagQuery(String query) => _validHashtagQuery.hasMatch(query);

/// Returns whether [handle] is a valid visible mention handle.
bool isValidMentionHandle(String handle) =>
    _validMentionHandle.hasMatch(handle);

/// Returns whether [text] contains whitespace.
bool containsFacetWhitespace(String text) => text.contains(_whitespace);

/// Removes sentence and unmatched closing punctuation from a detected link.
String trimFacetLinkText(String link) {
  var trimmed = link;
  while (trimmed.isNotEmpty &&
      facetTrailingSentencePunctuation.contains(trimmed[trimmed.length - 1])) {
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

int _count(String text, String character) => character.allMatches(text).length;
