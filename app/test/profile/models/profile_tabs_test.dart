import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/profile/widgets/profile_tab_bar.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('UT-002 profile tab policy', () {
    test('keeps the ordinary tabs and stable identities', () {
      final tabs = ProfileTabPolicy.forProfile(
        accountType: AccountType.regular,
        isBlocked: false,
      );

      expect(tabs, const [
        ProfileTab.projects,
        ProfileTab.posts,
        ProfileTab.comments,
        ProfileTab.reposts,
      ]);
      expect(
        tabs.map((tab) => tab.storageKey),
        const [
          'profile_tab_projects',
          'profile_tab_posts',
          'profile_tab_comments',
          'profile_tab_reposts',
        ],
      );
    });

    test('uses the exact stable business order', () {
      final tabs = ProfileTabPolicy.forProfile(
        accountType: AccountType.business,
        isBlocked: false,
        isOwnProfile: true,
      );

      expect(tabs, const [
        ProfileTab.products,
        ProfileTab.upcomingEvents,
        ProfileTab.about,
        ProfileTab.projects,
        ProfileTab.posts,
        ProfileTab.comments,
        ProfileTab.reposts,
      ]);
    });

    test('visitor business tabs reflect product and upcoming event state', () {
      final empty = ProfileTabPolicy.forProfile(
        accountType: AccountType.business,
        isBlocked: false,
      );
      final hydrated = ProfileTabPolicy.forProfile(
        accountType: AccountType.business,
        isBlocked: false,
        hasProducts: true,
        hasUpcomingEvents: true,
      );

      expect(empty, const [
        ProfileTab.about,
        ProfileTab.projects,
        ProfileTab.posts,
        ProfileTab.comments,
        ProfileTab.reposts,
      ]);
      expect(hydrated, ProfileTabPolicy.businessTabs);
      expect(
        ProfileTabPolicy.forProfile(
          accountType: AccountType.business,
          isBlocked: false,
          hasProducts: true,
        ),
        const [
          ProfileTab.products,
          ProfileTab.about,
          ProfileTab.projects,
          ProfileTab.posts,
          ProfileTab.comments,
          ProfileTab.reposts,
        ],
      );
      expect(
        ProfileTabPolicy.forProfile(
          accountType: AccountType.business,
          isBlocked: false,
          hasUpcomingEvents: true,
        ),
        const [
          ProfileTab.upcomingEvents,
          ProfileTab.about,
          ProfileTab.projects,
          ProfileTab.posts,
          ProfileTab.comments,
          ProfileTab.reposts,
        ],
      );
    });

    test('blocked profiles never receive business tabs', () {
      expect(
        ProfileTabPolicy.forProfile(
          accountType: AccountType.business,
          isBlocked: true,
        ),
        ProfileTabPolicy.ordinaryTabs,
      );
    });

    test(
      'retains logical selection or remaps removed tabs to the first tab',
      () {
        expect(
          ProfileTabPolicy.selectionAfterChange(
            selected: ProfileTab.comments,
            tabs: ProfileTabPolicy.businessTabs,
          ),
          ProfileTab.comments,
        );
        expect(
          ProfileTabPolicy.selectionAfterChange(
            selected: ProfileTab.products,
            tabs: ProfileTabPolicy.ordinaryTabs,
          ),
          ProfileTab.projects,
        );
        expect(
          ProfileTabPolicy.selectionAfterChange(
            selected: ProfileTab.upcomingEvents,
            tabs: ProfileTabPolicy.ordinaryTabs,
          ),
          ProfileTab.projects,
        );
      },
    );
  });
}
