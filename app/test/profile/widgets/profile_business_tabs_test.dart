import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/widgets/product_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_products_tab.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-003 Products preserves authored order', (tester) async {
    await _pumpSliver(
      tester,
      const ProfileProductsTab(
        products: [
          BusinessProductView(title: 'First authored product'),
          BusinessProductView(title: 'Second authored product'),
        ],
        isOwnProfile: false,
      ),
    );

    final cards = tester.widgetList<ProductCard>(find.byType(ProductCard));
    expect(cards.map((card) => card.product.title), [
      'First authored product',
      'Second authored product',
    ]);
  });

  testWidgets('AT-003 owner and visitor Product empty states differ', (
    tester,
  ) async {
    var manageCalls = 0;
    await _pumpSliver(
      tester,
      ProfileProductsTab(
        products: const [],
        isOwnProfile: true,
        onManage: () => manageCalls++,
      ),
    );
    expect(
      find.text('Add featured products to help visitors find your work.'),
      findsOneWidget,
    );
    await tester.tap(find.widgetWithText(TextButton, 'Manage products'));
    expect(manageCalls, 1);

    await _pumpSliver(
      tester,
      const ProfileProductsTab(products: [], isOwnProfile: false),
    );
    expect(find.text('No featured products yet.'), findsOneWidget);
    expect(find.byType(TextButton), findsNothing);
  });
}

Future<void> _pumpSliver(WidgetTester tester, Widget sliver) =>
    tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(body: CustomScrollView(slivers: [sliver])),
      ),
    );
