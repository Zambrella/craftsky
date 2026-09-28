import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';

part 'account_moderation.mapper.dart';

@immutable
final class ModerationCaseReference {
  const ModerationCaseReference._(this.value);

  factory ModerationCaseReference.parse(String value) {
    final match = _pattern.firstMatch(value);
    if (match == null) {
      throw const FormatException('Invalid moderation case reference');
    }
    return ModerationCaseReference._(
      'MOD-${match.group(1)!.toLowerCase()}',
    );
  }

  static final _pattern = RegExp(
    '^MOD-([0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}'
    r'-[89ab][0-9a-f]{3}-[0-9a-f]{12})$',
    caseSensitive: false,
  );

  final String value;

  @override
  bool operator ==(Object other) =>
      other is ModerationCaseReference && other.value == value;

  @override
  int get hashCode => value.hashCode;

  @override
  String toString() => value;
}

@MappableClass()
final class AccountStanding with AccountStandingMappable {
  const AccountStanding({
    required this.activeStrikeCount,
    required this.strikeThreshold,
    required this.thresholdSuspended,
    required this.severeSuspended,
    required this.suspended,
  });

  final int activeStrikeCount;
  final int strikeThreshold;
  final bool thresholdSuspended;
  final bool severeSuspended;
  final bool suspended;
}

@MappableEnum(defaultValue: ModerationReason.unknown)
enum ModerationReason {
  harassment,
  hate,
  spam,
  misleading,
  @MappableValue('suspected_ai_generated')
  suspectedAiGenerated,
  @MappableValue('adult_or_graphic')
  adultOrGraphic,
  impersonation,
  @MappableValue('off_topic')
  offTopic,
  @MappableValue('intellectual_property')
  intellectualProperty,
  other,
  unknown,
}

@MappableEnum(defaultValue: ModerationEffectType.unknown)
enum ModerationEffectType {
  formalWarning,
  visibilityWarn,
  visibilityHide,
  visibilityTakedown,
  strike,
  severeSuspension,
  unknown,
}

@MappableEnum(defaultValue: ModerationEffectAction.unknown)
enum ModerationEffectAction { apply, negate, expire, restore, unknown }

@MappableEnum(defaultValue: ModerationAppealState.unknown)
enum ModerationAppealState { none, pending, upheld, changed, unknown }

@MappableClass()
final class ModerationSafeSnapshot with ModerationSafeSnapshotMappable {
  const ModerationSafeSnapshot({
    required this.type,
    required this.did,
    this.collection,
    this.rkey,
    this.uri,
    this.cid,
    this.submittedHandle,
  });

  final String type;
  final String did;
  final String? collection;
  final String? rkey;
  final String? uri;
  final String? cid;
  final String? submittedHandle;

  String? get displayReference {
    if (uri case final value? when value.isNotEmpty) return value;
    if (submittedHandle case final value? when value.isNotEmpty) return value;
    if (did.isNotEmpty) return did;
    return null;
  }
}

@MappableClass()
final class ModerationHistoryEffect with ModerationHistoryEffectMappable {
  const ModerationHistoryEffect({
    required this.type,
    required this.action,
    this.dueAt,
  });

  final ModerationEffectType type;
  final ModerationEffectAction action;
  final DateTime? dueAt;
}

@MappableClass()
final class ModerationHistoryEntry with ModerationHistoryEntryMappable {
  ModerationHistoryEntry({
    required String caseReference,
    required this.eventType,
    required this.effects,
    required this.safeSnapshot,
    required this.occurredAt,
    this.reason = ModerationReason.unknown,
    this.userSafeDetail = '',
    this.appealStatus = ModerationAppealState.none,
  }) : caseReference = ModerationCaseReference.parse(caseReference).value;

  final String caseReference;
  final String eventType;
  final ModerationReason reason;
  final String userSafeDetail;
  final List<ModerationHistoryEffect> effects;
  final ModerationSafeSnapshot safeSnapshot;
  final ModerationAppealState appealStatus;
  final DateTime occurredAt;

  ModerationCaseReference get reference =>
      ModerationCaseReference.parse(caseReference);

  bool get canStartAppeal =>
      eventType == 'decision' && appealStatus == ModerationAppealState.none;
}

@MappableClass()
final class ModerationHistoryPage with ModerationHistoryPageMappable {
  const ModerationHistoryPage({required this.items, this.cursor});

  final List<ModerationHistoryEntry> items;
  final String? cursor;
}
