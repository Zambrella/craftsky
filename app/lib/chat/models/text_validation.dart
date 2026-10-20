import 'dart:convert';

import 'package:characters/characters.dart';

enum MessageValidationError { blank, graphemeLimit, byteLimit, invalidUnicode }

enum ReportValidationError {
  unsupportedReason,
  graphemeLimit,
  byteLimit,
  invalidUnicode,
}

ReportValidationError? validateReportInput(String reasonType, String? reason) {
  const supportedReasons = {
    'com.atproto.moderation.defs#reasonSpam',
    'com.atproto.moderation.defs#reasonViolation',
    'com.atproto.moderation.defs#reasonMisleading',
    'com.atproto.moderation.defs#reasonSexual',
    'com.atproto.moderation.defs#reasonRude',
    'com.atproto.moderation.defs#reasonOther',
  };
  if (!supportedReasons.contains(reasonType)) {
    return ReportValidationError.unsupportedReason;
  }
  if (reason != null) {
    if (!_isWellFormed(reason)) return ReportValidationError.invalidUnicode;
    if (utf8.encode(reason).length > 20000) {
      return ReportValidationError.byteLimit;
    }
    if (reason.characters.length > 2000) {
      return ReportValidationError.graphemeLimit;
    }
  }
  return null;
}

/// Validate without truncating or changing the owner's draft.
MessageValidationError? validateMessageText(String text) {
  if (!_isWellFormed(text)) return MessageValidationError.invalidUnicode;
  if (text.trim().isEmpty) return MessageValidationError.blank;
  if (utf8.encode(text).length > 10000) {
    return MessageValidationError.byteLimit;
  }
  if (text.characters.length > 1000) {
    return MessageValidationError.graphemeLimit;
  }
  return null;
}

bool _isWellFormed(String text) {
  for (var index = 0; index < text.length; index++) {
    final unit = text.codeUnitAt(index);
    if (unit >= 0xd800 && unit <= 0xdbff) {
      if (++index == text.length) return false;
      final low = text.codeUnitAt(index);
      if (low < 0xdc00 || low > 0xdfff) return false;
    } else if (unit >= 0xdc00 && unit <= 0xdfff) {
      return false;
    }
  }
  return true;
}
