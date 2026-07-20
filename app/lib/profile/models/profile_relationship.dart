import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';

part 'profile_relationship.mapper.dart';

@MappableEnum()
enum ProfileRelationshipAction { mute, unmute, block, unblock }

enum ProfileRelationshipKind { none, muted, blocking, blockedBy, mutualBlock }

@immutable
@MappableClass(
  ignoreNull: true,
  hook: ProfileRelationshipWireHook(),
  generateMethods:
      GenerateMethods.decode | GenerateMethods.copy | GenerateMethods.equals,
)
class ProfileRelationship with ProfileRelationshipMappable {
  const ProfileRelationship({
    this.muted = false,
    this.blocking = false,
    this.blockedBy = false,
    this.uri,
    this.cid,
    this.rkey,
    this.pendingAction,
    this.lastError,
    this.confirmedOverlay = false,
    this.initialized = false,
  });

  factory ProfileRelationship.fromProfileFlags({
    required bool muted,
    required bool blocking,
    required bool blockedBy,
  }) => ProfileRelationship(
    muted: muted,
    blocking: blocking,
    blockedBy: blockedBy,
    initialized: true,
  );

  final bool muted;
  final bool blocking;
  final bool blockedBy;
  final String? uri;
  final String? cid;
  final String? rkey;
  final ProfileRelationshipAction? pendingAction;
  final Object? lastError;
  final bool confirmedOverlay;
  final bool initialized;

  bool get hasBlock => blocking || blockedBy;

  ProfileRelationshipKind get kind {
    if (blocking && blockedBy) return ProfileRelationshipKind.mutualBlock;
    if (blocking) return ProfileRelationshipKind.blocking;
    if (blockedBy) return ProfileRelationshipKind.blockedBy;
    if (muted) return ProfileRelationshipKind.muted;
    return ProfileRelationshipKind.none;
  }

  bool samePolicy(ProfileRelationship other) =>
      muted == other.muted &&
      blocking == other.blocking &&
      blockedBy == other.blockedBy;
}

class ProfileRelationshipWireHook extends MappingHook {
  const ProfileRelationshipWireHook();

  @override
  Object? beforeDecode(Object? value) {
    if (value is! Map<String, dynamic>) return value;
    return <String, dynamic>{
      ...value,
      'pendingAction': null,
      'lastError': null,
      'confirmedOverlay': false,
      'initialized': true,
    };
  }
}
