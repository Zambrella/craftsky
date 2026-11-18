import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/settings/providers/account_hosting_provider.dart';
import 'package:craftsky_app/settings/widgets/account_provider_section.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  for (final bluesky in [true, false]) {
    testWidgets('provider links and account details for Bluesky=$bluesky', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final launched = <Uri>[];
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            accountHostingProvider.overrideWith(
              (ref) async => AccountHosting(
                did: 'did:plc:alice',
                handle: 'alice.example',
                pdsUrl: bluesky
                    ? 'https://morel.host.bsky.network'
                    : 'https://pds.example',
                hostingProvider: bluesky ? 'Bluesky' : 'pds.example',
                accountManagementUrl: bluesky
                    ? 'https://bsky.social/account'
                    : 'https://pds.example/account',
                passwordChangeUrl: bluesky
                    ? 'https://bsky.app/settings/account'
                    : null,
              ),
            ),
          ],
          child: MaterialApp(
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(context)
                  .copyWith(textScaler: const TextScaler.linear(2)),
              child: child!,
            ),
            home: Scaffold(
              body: ListView(
                children: [
                  AccountProviderSection(
                    did: 'did:plc:alice',
                    handle: 'old.example',
                    linkLauncher: (uri) async {
                      launched.add(uri);
                      return true;
                    },
                  ),
                ],
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('alice.example'), findsOneWidget);
      expect(find.text('did:plc:alice'), findsOneWidget);
      expect(find.textContaining('shared AT Protocol account'), findsOneWidget);
      final manage = find.text('Manage account with provider');
      for (
        var attempt = 0;
        attempt < 20 && manage.hitTestable().evaluate().isEmpty;
        attempt++
      ) {
        await tester.drag(find.byType(ListView), const Offset(0, -300));
        await tester.pumpAndSettle();
      }
      expect(manage.hitTestable(), findsOneWidget);
      await tester.tap(manage);
      await tester.pumpAndSettle();
      expect(
        launched.single.toString(),
        bluesky ? 'https://bsky.social/account' : 'https://pds.example/account',
      );
      final password = find.text('Change password in Bluesky');
      expect(password, bluesky ? findsOneWidget : findsNothing);
      if (bluesky) {
        for (
          var attempt = 0;
          attempt < 20 && password.hitTestable().evaluate().isEmpty;
          attempt++
        ) {
          await tester.drag(find.byType(ListView), const Offset(0, -300));
          await tester.pumpAndSettle();
        }
        expect(password.hitTestable(), findsOneWidget);
        await tester.tap(password);
        await tester.pumpAndSettle();
        expect(launched.last.toString(), 'https://bsky.app/settings/account');
      }
      expect(tester.takeException(), isNull);
    });
  }
  testWidgets('hides another account hosting details after account switch', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          accountHostingProvider.overrideWith(
            (ref) async => const AccountHosting(
              did: 'did:plc:old',
              handle: 'old.example',
              hostingProvider: 'Old provider',
              accountManagementUrl: 'https://old.example/account',
              passwordChangeUrl: 'https://bsky.app/settings/account',
            ),
          ),
        ],
        child: const MaterialApp(
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: SingleChildScrollView(
              child: AccountProviderSection(
                did: 'did:plc:new',
                handle: 'new.example',
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('new.example'), findsOneWidget);
    expect(find.text('Old provider'), findsNothing);
    expect(find.text('Manage account with provider'), findsNothing);
    expect(find.text('Change password in Bluesky'), findsNothing);
  });
}
