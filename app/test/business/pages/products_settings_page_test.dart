import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/pages/products_settings_page.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/providers/products_controller.dart';
import 'package:craftsky_app/business/widgets/product_editor.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../accessibility_test_helpers.dart';

void main() {
  for (final constraint in businessAccessibilityMatrix) {
    testWidgets(
      'AT-012 REG-010 Products manager fits '
      '${businessConstraintLabel(constraint)}',
      (tester) async {
        await setBusinessAccessibilityConstraint(tester, constraint);
        final semantics = tester.ensureSemantics();
        await tester.pumpWidget(_app(_identity(_profile(withProducts: true))));

        expect(
          tester.getSemantics(find.byType(CircularProgressIndicator)).label,
          contains('Loading'),
        );
        await tester.pumpAndSettle();
        expect(find.text('One'), findsOneWidget);
        expect(find.byType(CraftskyCard), findsNWidgets(2));
        expect(find.byType(CraftskyContextMenuButton), findsNWidgets(2));
        expect(
          tester.getSize(find.byType(CraftskyFloatingActionButton)).height,
          greaterThanOrEqualTo(48),
        );
        await expectKeyboardFocus(tester);
        expectNoAccessibilityLayoutException(tester);
        semantics.dispose();
      },
    );
  }

  testWidgets('AT-005 manager shows loading then authored products', (
    tester,
  ) async {
    await tester.pumpWidget(
      _app(_identity(_profile(withProducts: true)), repository: _Repository()),
    );
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    await tester.pumpAndSettle();

    expect(find.text('One'), findsOneWidget);
    expect(find.text('Two'), findsOneWidget);
    await tester.tap(find.byTooltip('Edit One'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Move One down'));
    await tester.pump();
    expect(
      tester.getTopLeft(find.text('Two')).dy,
      lessThan(tester.getTopLeft(find.text('One')).dy),
    );

    await tester.tap(find.byTooltip('Edit Two'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Remove Two'));
    await tester.pumpAndSettle();
    expect(find.text('Remove product?'), findsOneWidget);
    await tester.tap(find.widgetWithText(ChunkyButton, 'Remove'));
    await tester.pumpAndSettle();
    expect(find.text('Two'), findsNothing);
  });

  testWidgets('R12 AT-012 Products controls follow order and activate', (
    tester,
  ) async {
    await tester.pumpWidget(
      _app(_identity(_profile(withProducts: true)), repository: _Repository()),
    );
    await tester.pumpAndSettle();

    final manage = find.byTooltip('Edit One');
    final manageIcon = find.descendant(
      of: manage,
      matching: find.byIcon(Icons.more_horiz),
    );
    requestKeyboardFocus(tester, manageIcon);
    await tester.pump();
    expectKeyboardFocusOn(manageIcon);
    await tester.sendKeyEvent(LogicalKeyboardKey.space);
    await tester.pumpAndSettle();
    expect(find.text('Remove One'), findsOneWidget);
    await tester.tap(find.text('Remove One'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(ChunkyButton, 'Remove'));
    await tester.pumpAndSettle();

    expect(find.text('One'), findsNothing);
    expect(find.text('Two'), findsOneWidget);
  });

  testWidgets('AT-005 manager shows empty state and opens add editor', (
    tester,
  ) async {
    await tester.pumpWidget(_app(_identity(_profile())));
    await tester.pumpAndSettle();
    expect(find.text('No featured products yet.'), findsOneWidget);
    expect(find.text('Add product'), findsOneWidget);

    await tester.tap(find.text('Add product'));
    await tester.pumpAndSettle();
    expect(find.text('Save product'), findsOneWidget);
    final route = ModalRoute.of(tester.element(find.byType(ProductEditor)));
    expect(route, isA<MaterialPageRoute<ProductDraft>>());
    expect(
      (route! as MaterialPageRoute<ProductDraft>).fullscreenDialog,
      isTrue,
    );
    expect(find.byKey(const ValueKey('product-submit')), findsOneWidget);
    expect(
      find.byKey(const Key('product-editor-bottom-safe-space')),
      findsOneWidget,
    );
  });

  testWidgets('AT-005 manager disables add at the four-card cap', (
    tester,
  ) async {
    await tester.pumpWidget(_app(_identity(_profile(productCount: 4))));
    await tester.pumpAndSettle();

    final button = tester.widget<CraftskyFloatingActionButton>(
      find.byType(CraftskyFloatingActionButton),
    );
    expect(button.onPressed, isNull);
    expect(find.text('4 of 4 products'), findsOneWidget);
  });

  testWidgets('AT-005 manager exposes retryable load error', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          activeAccountIdentityProvider.overrideWith(
            (_) async => throw StateError('failed'),
          ),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ProductsSettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Products could not be loaded.'), findsOneWidget);
    expect(find.text('Retry'), findsOneWidget);
  });

  testWidgets('AT-005 manager fits narrow phone constraints', (tester) async {
    tester.view.physicalSize = const Size(320, 700);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(_app(_identity(_profile(withProducts: true))));
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(find.text('One'), findsOneWidget);
  });

  testWidgets('AT-005 regular owner guard exposes no manager controls', (
    tester,
  ) async {
    final profile = _profile().copyWith(
      accountType: AccountType.regular,
      business: null,
    );
    await tester.pumpWidget(_app(_identity(profile)));
    await tester.pumpAndSettle();

    expect(
      find.text('Product management is available to business accounts.'),
      findsOneWidget,
    );
    expect(find.text('Add product'), findsNothing);
  });

  testWidgets(
    'AT-011 product reordering persists immediately without a guard',
    (
      tester,
    ) async {
      final repository = _Repository();
      final identity = _identity(_profile(withProducts: true));
      await tester.pumpWidget(_app(identity, repository: repository));
      await tester.pumpAndSettle();
      final container = ProviderScope.containerOf(
        tester.element(find.byType(ProductsSettingsPage)),
      );
      final controller = container.read(productsControllerProvider.notifier);

      expect(
        await controller.move(
          controller.state.requireValue.products.first.id,
          1,
        ),
        isTrue,
      );
      await tester.pumpAndSettle();

      expect(repository.saves, 1);
      expect(find.text('Save products'), findsNothing);
      expect(
        container
            .read(productsControllerProvider)
            .requireValue
            .products
            .map((product) => product.title),
        ['Two', 'One'],
      );
      expect(
        await container
            .read(unsavedWorkGuardProvider)
            .confirmLeave(identity.lease),
        isTrue,
      );
      expect(find.byType(CraftskyDialog), findsNothing);
    },
  );
}

Widget _app(
  ActiveAccountIdentity identity, {
  BusinessRepository? repository,
}) => ProviderScope(
  overrides: [
    activeAccountIdentityProvider.overrideWith((_) async => identity),
    if (repository != null)
      businessRepositoryProvider.overrideWithValue(repository),
  ],
  child: MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: const ProductsSettingsPage(),
  ),
);

ActiveAccountIdentity _identity(Profile profile) => ActiveAccountIdentity(
  lease: AccountSessionLease(
    account: AccountKey('did:plc:owner'),
    sessionGeneration: 1,
  ),
  profile: profile,
);

Profile _profile({bool withProducts = false, int productCount = 0}) => Profile(
  did: 'did:plc:owner',
  handle: 'owner.test',
  crafts: const [],
  accountType: AccountType.business,
  business: BusinessProfile(
    cid: 'bafy-current',
    products: productCount > 0
        ? [
            for (var index = 1; index <= productCount; index++)
              _product('$index'),
          ]
        : withProducts
        ? [_product('One'), _product('Two')]
        : const [],
  ),
);

BusinessProductView _product(String title) => BusinessProductView(
  title: title,
  uri: 'https://shop.example/$title',
  image: BusinessImageView(
    cid: 'bafy-$title',
    mime: 'image/jpeg',
    size: 10,
    alt: title,
    thumb: 'https://cdn.example/$title/thumb',
    fullsize: 'https://cdn.example/$title/full',
  ),
);

final class _Repository extends Fake implements BusinessRepository {
  int saves = 0;

  @override
  Future<RecordMutationResult> putBusinessProfile(
    Map<String, dynamic> body, {
    required Cid? expectedCid,
  }) async {
    saves++;
    return RecordMutationResult(cid: 'bafy-accepted');
  }
}
