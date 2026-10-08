import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';

/// Projects cached owner identity against the current account-bound lease.
/// The authoritative profile and its PDS-backed business data remain untouched.
Profile projectOwnerFeatureAccess(Profile profile, SubscriptionAccess? access) {
  final matched = access?.did.value == profile.did;
  final plus = matched && access!.allowsPlus;
  final business = matched && access!.allowsBusiness;
  return profile.copyWith(
    accountType: business ? AccountType.business : AccountType.regular,
    business: business ? profile.business : null,
    hasUpcomingEvents: business && profile.hasUpcomingEvents,
    customisation: plus ? profile.customisation : ProfileCustomisation.defaults,
  );
}
