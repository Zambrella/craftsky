import 'dart:ui' show Tristate;

import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/widgets/business_profile_summary.dart';
import 'package:craftsky_app/business/widgets/product_editor.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_tab_bar.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_about_tab.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'accessibility_test_helpers.dart';

void main() {
  for (final constraint in businessAccessibilityMatrix) {
    final (:size, textScale: scale) = constraint;
    testWidgets(
      'AT-012 summary/About fit ${size.width}x${size.height} at $scale',
      (tester) async {
        await setBusinessAccessibilityConstraint(tester, constraint);
        await _pump(
          tester,
          DefaultTabController(
            length: ProfileTabPolicy.businessTabs.length,
            child: CustomScrollView(
              slivers: [
                const SliverToBoxAdapter(
                  child: ProfileIdentity(
                    handle: 'quality.test',
                    displayName: 'Quality Yarns',
                    businessLabel: 'Business',
                  ),
                ),
                SliverToBoxAdapter(
                  child: BusinessProfileSummary(
                    business: BusinessProfile(
                      cid: 'bafyquality',
                      tagline: 'Tools and yarn for patient makers',
                      primaryAction: const BusinessAction(
                        type: 'shop',
                        destination: 'https://seller.example/shop',
                      ),
                    ),
                    confirmExternal: (_, _) async => false,
                  ),
                ),
                const SliverToBoxAdapter(
                  child: SizedBox(
                    height: ProfileTabBarDelegate.height,
                    child: ProfileTabBar(tabs: ProfileTabPolicy.businessTabs),
                  ),
                ),
                ProfileAboutTab(
                  profile: Profile(
                    did: 'did:plc:quality',
                    handle: 'quality.test',
                    crafts: const [],
                    accountType: AccountType.business,
                    business: BusinessProfile(
                      cid: 'bafyquality',
                      businessTypes: const [
                        BusinessOpenValue(value: 'dyer', known: true),
                      ],
                      offerings: const [
                        BusinessOpenValue(value: 'yarn', known: true),
                      ],
                      location: const BusinessLocation(
                        country: 'GB',
                        locality: 'Bristol',
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        );

        expect(tester.takeException(), isNull);
        expect(find.text('Business'), findsOneWidget);
        expect(
          tester
              .getSemantics(find.widgetWithText(OutlinedButton, 'Shop'))
              .label,
          contains('Shop'),
        );
        expect(find.text('Products'), findsOneWidget);
        expect(find.text('Upcoming Events'), findsOneWidget);
        expect(
          tester.getSemantics(find.text('Products')).flagsCollection.isSelected,
          Tristate.isTrue,
        );
        await tester.sendKeyEvent(LogicalKeyboardKey.tab);
        expect(FocusManager.instance.primaryFocus, isNotNull);
      },
    );

    testWidgets(
      'AT-012 product editor fits ${size.width}x${size.height} at $scale',
      (tester) async {
        await setBusinessAccessibilityConstraint(tester, constraint);
        await _pump(
          tester,
          ProductEditor(
            initial: const ProductDraft(
              id: 'product',
              title: 'Yarn',
              destination: 'https://seller.example/yarn',
              image: MissingBusinessImageDraft(),
            ),
            onSave: (_) {},
          ),
        );

        expect(tester.takeException(), isNull);
        expect(find.text('Edit product'), findsOneWidget);
        final saveSize = tester.getSize(
          find.widgetWithText(ChunkyButton, 'Save product'),
        );
        expect(saveSize.height, greaterThanOrEqualTo(48));
      },
    );
  }

  testWidgets('AT-012 controls have semantic names and keyboard focus', (
    tester,
  ) async {
    await setBusinessAccessibilityConstraint(
      tester,
      businessAccessibilityMatrix[2],
    );
    final semantics = tester.ensureSemantics();
    await _pump(
      tester,
      ProductEditor(
        initial: const ProductDraft(
          id: 'product',
          title: 'Yarn',
          destination: 'https://seller.example/yarn',
          image: MissingBusinessImageDraft(),
        ),
        onSave: (_) {},
      ),
    );

    expect(
      tester.getSemantics(find.byKey(const ValueKey('product-title'))).label,
      contains('Title'),
    );
    expect(
      tester
          .getSemantics(find.byKey(const ValueKey('product-destination')))
          .label,
      contains('Destination'),
    );
    expect(
      tester
          .getSemantics(find.widgetWithText(ChunkyButton, 'Save product'))
          .label,
      contains('Save product'),
    );

    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    expect(FocusManager.instance.primaryFocus, isNotNull);
    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    expect(FocusManager.instance.primaryFocus, isNotNull);
    semantics.dispose();
  });
}

Future<void> _pump(WidgetTester tester, Widget child) => tester.pumpWidget(
  ProviderScope(
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(body: child),
    ),
  ),
);
