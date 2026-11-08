import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:flutter/foundation.dart';

/// Reading has its own identity; a guest never owns an account operation.
@immutable
sealed class BrowsingContext {
  const new();
  String? get viewerDid;
  BrowsingCapabilities get capabilities;
}

final class GuestBrowsingContext extends BrowsingContext {
  const new({required this.epoch});
  final int epoch;
  @override
  String? get viewerDid => null;
  @override
  BrowsingCapabilities get capabilities => const BrowsingCapabilities.public();
  @override
  bool operator ==(Object other) =>
      other is GuestBrowsingContext && other.epoch == epoch;
  @override
  int get hashCode => Object.hash(GuestBrowsingContext, epoch);
}

final class BrowsingLoading extends BrowsingContext {
  const new({required this.epoch});
  final int epoch;
  @override
  String? get viewerDid => null;
  @override
  BrowsingCapabilities get capabilities => const BrowsingCapabilities.loading();
  @override
  bool operator ==(Object other) =>
      other is BrowsingLoading && other.epoch == epoch;
  @override
  int get hashCode => Object.hash(BrowsingLoading, epoch);
}

final class MemberBrowsingContext extends BrowsingContext {
  const new({required this.lease});
  final ActiveAccountLease lease;
  @override
  String get viewerDid => lease.session.account.did;
  @override
  BrowsingCapabilities get capabilities => const BrowsingCapabilities.member();
  @override
  bool operator ==(Object other) =>
      other is MemberBrowsingContext && other.lease == lease;
  @override
  int get hashCode => Object.hash(MemberBrowsingContext, lease);
}

final class BrowsingCapabilities {
  const new public() : canReadPublic = true, canMutate = false;
  const new member() : canReadPublic = true, canMutate = true;
  const new loading() : canReadPublic = false, canMutate = false;
  final bool canReadPublic;
  final bool canMutate;
  bool get canShare => canReadPublic;
  bool get canUseProjectFeedControls => canReadPublic;
  bool get canReadPrivate => canMutate;
  bool get canUseAutocomplete => canMutate;
  bool get canViewRecentUsers => canMutate;
  bool get canViewTopHashtags => canMutate;
  bool get canUseAdvancedSearch => canMutate;
}
