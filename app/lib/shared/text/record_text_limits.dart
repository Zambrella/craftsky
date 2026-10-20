import 'dart:convert';

import 'package:characters/characters.dart';
import 'package:flutter/services.dart';

/// Lexicon strings are bounded independently by graphemes and UTF-8 bytes.
class RecordTextLimit {
  const new(this.graphemes, this.bytes);

  final int graphemes;
  final int bytes;

  bool accepts(String value) =>
      value.characters.length <= graphemes &&
      utf8.encode(value).length <= bytes;
}

abstract final class RecordTextLimits {
  static const post = RecordTextLimit(2000, 20000);
  static const imageAlt = RecordTextLimit(1000, 10000);
  static const videoAlt = RecordTextLimit(1000, 10000);
  static const displayName = RecordTextLimit(64, 640);
  static const pronouns = RecordTextLimit(20, 200);
  static const bio = RecordTextLimit(256, 2560);
  static const projectTitle = RecordTextLimit(200, 500);
  static const patternText = RecordTextLimit(200, 500);
  static const projectSize = RecordTextLimit(100, 500);
  static const fitNotes = RecordTextLimit(600, 6000);
  static const material = RecordTextLimit(100, 1000);
  static const hashtag = RecordTextLimit(64, 640);
  static const businessTagline = RecordTextLimit(100, 1000);
  static const businessHours = RecordTextLimit(300, 3000);
  static const businessServiceArea = RecordTextLimit(200, 2000);
  static const locality = RecordTextLimit(100, 1000);
  static const eventName = RecordTextLimit(200, 2000);
  static const eventSummary = RecordTextLimit(1000, 10000);
  static const eventVenue = RecordTextLimit(200, 2000);
  static const productTitle = RecordTextLimit(150, 1500);
}

/// Apply only to committed edits, preserving whole grapheme clusters and IME
/// composition. Existing values are validated separately rather than truncated.
class Utf8LengthLimitingTextInputFormatter extends TextInputFormatter {
  const new(this.maxBytes);

  final int maxBytes;

  @override
  TextEditingValue formatEditUpdate(
    TextEditingValue oldValue,
    TextEditingValue newValue,
  ) {
    if (!newValue.composing.isCollapsed ||
        utf8.encode(newValue.text).length <= maxBytes) {
      return newValue;
    }
    final buffer = StringBuffer();
    var bytes = 0;
    for (final cluster in newValue.text.characters) {
      bytes += utf8.encode(cluster).length;
      if (bytes > maxBytes) break;
      buffer.write(cluster);
    }
    final text = buffer.toString();
    return TextEditingValue(
      text: text,
      selection: TextSelection(
        baseOffset: newValue.selection.baseOffset.clamp(0, text.length),
        extentOffset: newValue.selection.extentOffset.clamp(0, text.length),
      ),
    );
  }
}
