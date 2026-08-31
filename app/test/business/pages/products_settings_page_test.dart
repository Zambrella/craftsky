import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/pages/products_settings_page.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/providers/products_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
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
        expect(
          tester
              .getSemantics(
                find.widgetWithIcon(IconButton, Icons.delete_outline).first,
              )
              .label,
          contains('Remove One'),
        );
        expect(
          tester
              .getSize(find.widgetWithText(OutlinedButton, 'Add product'))
              .height,
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
    await tester.pumpWidget(_app(_identity(_profile(withProducts: true))));
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    await tester.pumpAndSettle();

    expect(find.text('One'), findsOneWidget);
    expect(find.text('Two'), findsOneWidget);
    expect(find.byTooltip('Move One down'), findsOneWidget);
    expect(find.byTooltip('Move Two up'), findsOneWidget);

    await tester.tap(find.byTooltip('Move One down'));
    await tester.pump();
    expect(
      tester.getTopLeft(find.text('Two')).dy,
      lessThan(tester.getTopLeft(find.text('One')).dy),
    );

    await tester.tap(find.byTooltip('Remove Two'));
    await tester.pump();
    expect(find.text('Two'), findsNothing);
  });

  testWidgets('R12 AT-012 Products controls follow order and activate', (
    tester,
  ) async {
    await tester.pumpWidget(_app(_identity(_profile(withProducts: true))));
    await tester.pumpAndSettle();

    final moveDown = find.byTooltip('Move One down');
    final remove = find.byTooltip('Remove One');
    final moveDownIcon = find.descendant(
      of: moveDown,
      matching: find.byIcon(Icons.arrow_downward),
    );
    final removeIcon = find.descendant(
      of: remove,
      matching: find.byIcon(Icons.delete_outline),
    );
    requestKeyboardFocus(tester, moveDownIcon);
    await tester.pump();
    expectKeyboardFocusOn(moveDownIcon);
    await pressTabAndExpectFocus(tester, removeIcon);
    await tester.sendKeyEvent(LogicalKeyboardKey.space);
    await tester.pump();

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
  });

  testWidgets('AT-005 manager disables add at the four-card cap', (
    tester,
  ) async {
    await tester.pumpWidget(_app(_identity(_profile(productCount: 4))));
    await tester.pumpAndSettle();

    final button = tester.widget<OutlinedButton>(
      find.widgetWithText(OutlinedButton, 'Add product'),
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
    'AT-011 IT-010 REG-008 guards dirty products and clears after save/dispose',
    (tester) async {
      final repository = _Repository();
      final identity = _identity(_profile(withProducts: true));
      await tester.pumpWidget(_app(identity, repository: repository));
      await tester.pumpAndSettle();
      final container = ProviderScope.containerOf(
        tester.element(find.byType(ProductsSettingsPage)),
      );
      final guard = container.read(unsavedWorkGuardProvider);
      final controller = container.read(productsControllerProvider.notifier);
      controller.move(controller.state.requireValue.products.first.id, 1);
      await tester.pump();

      var registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'owner-token',
            did: 'did:plc:owner',
            handle: 'owner.test',
          )
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      registry = registry.activate(identity.lease);
      final target = registry.leaseFor(AccountKey('did:plc:bob'))!;
      var activations = 0;
      final activation = AccountActivationCoordinator(
        readRegistry: () => registry,
        commitActivation: (lease) async {
          activations++;
          registry = registry.activate(lease);
        },
        invalidateAccountState: () async {},
        resetToHome: () async {},
        confirmLeave: guard.confirmLeave,
      ).activate(target);
      await tester.pumpAndSettle();
      expect(find.text('Discard changes?'), findsOneWidget);
      await tester.tap(find.text('Keep editing'));
      await tester.pumpAndSettle();
      expect(await activation, AccountActivationResult.cancelled);
      expect(activations, 0);

      await tester.tap(find.text('Save products'));
      await tester.pumpAndSettle();
      expect(repository.saves, 1);
      expect(await guard.confirmLeave(identity.lease), isTrue);
      expect(find.text('Discard changes?'), findsNothing);

      controller.move(controller.state.requireValue.products.first.id, 1);
      await tester.pump();
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
      expect(await guard.confirmLeave(identity.lease), isTrue);
      expect(find.text('Discard changes?'), findsNothing);
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
