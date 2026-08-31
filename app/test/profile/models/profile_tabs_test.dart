import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/profile/widgets/profile_tab_bar.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('UT-002 profile tab policy', () {
    test('keeps the ordinary five tabs and stable identities', () {
      final tabs = ProfileTabPolicy.forProfile(
        accountType: AccountType.regular,
        isBlocked: false,
      );

      expect(tabs, const [
        ProfileTab.projects,
        ProfileTab.posts,
        ProfileTab.comments,
        ProfileTab.reposts,
        ProfileTab.about,
      ]);
      expect(
        tabs.map((tab) => tab.storageKey),
        const [
          'profile_tab_projects',
          'profile_tab_posts',
          'profile_tab_comments',
          'profile_tab_reposts',
          'profile_tab_about',
        ],
      );
    });

    test('uses the exact stable business order', () {
      final tabs = ProfileTabPolicy.forProfile(
        accountType: AccountType.business,
        isBlocked: false,
      );

      expect(tabs, const [
        ProfileTab.projects,
        ProfileTab.posts,
        ProfileTab.comments,
        ProfileTab.reposts,
        ProfileTab.products,
        ProfileTab.upcomingEvents,
        ProfileTab.about,
      ]);
    });

    test('does not derive business tabs from product or event state', () {
      final emptyOrLoading = ProfileTabPolicy.forProfile(
        accountType: AccountType.business,
        isBlocked: false,
      );
      final hydrated = ProfileTabPolicy.forProfile(
        accountType: AccountType.business,
        isBlocked: false,
      );

      expect(hydrated, same(emptyOrLoading));
      expect(
        hydrated.map((tab) => tab.storageKey),
        emptyOrLoading.map((tab) => tab.storageKey),
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
      'retains logical selection or remaps removed business tabs to About',
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
          ProfileTab.about,
        );
        expect(
          ProfileTabPolicy.selectionAfterChange(
            selected: ProfileTab.upcomingEvents,
            tabs: ProfileTabPolicy.ordinaryTabs,
          ),
          ProfileTab.about,
        );
      },
    );
  });
}
