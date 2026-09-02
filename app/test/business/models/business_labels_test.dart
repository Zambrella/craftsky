import 'package:craftsky_app/business/models/business_labels.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('UT-003 maps every known business catalog value', (tester) async {
    late AppLocalizations l10n;
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Builder(
          builder: (context) {
            l10n = AppLocalizations.of(context);
            return const SizedBox();
          },
        ),
      ),
    );

    expect(
      BusinessLabels.businessTypes.map(
        (value) => BusinessLabels.openValue(value, l10n),
      ),
      const [
        'Dyer',
        'Fiber producer',
        'Fiber processor',
        'Yarn shop',
        'Fabric shop',
        'Craft supply shop',
        'Pattern designer',
        'Finished goods maker',
        'Tool maker',
        'Teacher',
        'Craft studio',
        'Repair service',
        'Technical editor',
        'Photographer',
        'Publisher',
        'Other craft business',
      ],
    );
    expect(
      BusinessLabels.offerings.map(
        (value) => BusinessLabels.openValue(value, l10n),
      ),
      const [
        'Yarn',
        'Fiber',
        'Fabric',
        'Patterns',
        'Kits',
        'Notions',
        'Tools',
        'Finished goods',
        'Custom work',
        'Repairs',
        'Classes',
        'Studio hire',
        'Wholesale',
        'Digital products',
        'Technical editing',
        'Photography services',
        'Fiber processing',
      ],
    );
    expect(
      BusinessLabels.actions.map((type) => BusinessLabels.action(type, l10n)),
      const [
        'Shop',
        'Browse patterns',
        'Request custom order',
        'Book class',
        'Book appointment',
        'View event calendar',
        'Email',
        'Visit website',
        'Wholesale enquiries',
      ],
    );
  });

  testWidgets('UT-003 bounds unknown values and formats hydrated location', (
    tester,
  ) async {
    late AppLocalizations l10n;
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Builder(
          builder: (context) {
            l10n = AppLocalizations.of(context);
            return const SizedBox();
          },
        ),
      ),
    );

    expect(
      BusinessLabels.openValue(
        const BusinessOpenValue(value: 'loom-consulting', known: false),
        l10n,
      ),
      'Other: Loom consulting',
    );
    final bounded = BusinessLabels.openValue(
      BusinessOpenValue(value: 'x' * 500, known: false),
      l10n,
    );
    expect(bounded.length, lessThanOrEqualTo(71));
    expect(bounded, isNot(contains('\n')));
    expect(
      BusinessLabels.location(
        const BusinessLocation(country: 'GB', locality: 'Bristol'),
        l10n,
      ),
      'Bristol, United Kingdom',
    );
    expect(
      BusinessLabels.location(
        const BusinessLocation(country: 'US'),
        l10n,
      ),
      'United States',
    );
  });
}
