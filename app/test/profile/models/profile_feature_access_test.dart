import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/models/profile_feature_access.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-010 lapsed owner profile hides cached Business and customisation',
    () {
      final profile = Profile(
        did: 'did:plc:owner',
        handle: 'owner.test',
        crafts: const [],
        accountType: AccountType.business,
        business: BusinessProfile(cid: 'cid', tagline: 'Shop'),
        hasUpcomingEvents: true,
        customisation: const ProfileCustomisation(
          colour: 'orchid',
          background: 'skewdark',
        ),
      );
      final access = SubscriptionAccess(
        did: Did.parse('did:plc:owner'),
        effectiveTier: SubscriptionTier.free,
        givesAccess: false,
      );
      final shown = projectOwnerFeatureAccess(profile, access);
      expect(shown.accountType, AccountType.regular);
      expect(shown.business, isNull);
      expect(shown.hasUpcomingEvents, isFalse);
      expect(shown.customisation, ProfileCustomisation.defaults);
      expect(profile.business?.tagline, 'Shop');
    },
  );
}
