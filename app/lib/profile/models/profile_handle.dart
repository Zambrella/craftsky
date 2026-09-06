import 'package:craftsky_app/shared/atproto/identifiers.dart';

/// Presentation and alias policy for an authoritative current handle.
final class ProfileHandle {
  const ProfileHandle(this.value);

  static const invalidSentinel = 'handle.invalid';

  final String? value;

  bool get isAvailable => aliasInput != null;

  bool get isAliasEligible => isAvailable;

  String? get aliasInput {
    final candidate = value?.trim();
    if (candidate == null ||
        candidate.isEmpty ||
        candidate == invalidSentinel) {
      return null;
    }
    try {
      Handle.parse(candidate);
    } on Object {
      return null;
    }
    return candidate;
  }

  String currentLabel({
    required String unavailableLabel,
    bool includeAtSign = true,
  }) {
    final alias = aliasInput;
    if (alias == null) return unavailableLabel;
    return includeAtSign ? '@$alias' : alias;
  }

  String displayLabel({
    required String unavailableLabel,
    String? displayName,
    bool includeAtSignWhenNoDisplayName = false,
  }) {
    final name = displayName?.trim();
    return name == null || name.isEmpty
        ? currentLabel(
            unavailableLabel: unavailableLabel,
            includeAtSign: includeAtSignWhenNoDisplayName,
          )
        : name;
  }
}
