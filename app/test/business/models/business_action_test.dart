import 'package:craftsky_app/business/models/business_action.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/l10n/generated/app_localizations_en.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  final l10n = AppLocalizationsEn();

  test('UT-012 maps every known action to localized label and icon', () {
    expect(
      BusinessActionFormatter.knownTypes.map(
        (type) => BusinessActionFormatter.presentation(type, l10n).label,
      ),
      [
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
    expect(
      BusinessActionFormatter.presentation('email', l10n).icon,
      CraftskyIconsBold.email,
    );
    expect(
      BusinessActionFormatter.presentation('shop', l10n).icon,
      CraftskyIconsBold.externalLink,
    );
  });

  test('UT-012 accepts only exact hydrated HTTPS and mailto destinations', () {
    const https = 'https://seller.example/item?q=one%20two#buy';
    const email = 'mailto:seller@example.com?subject=Custom%20order';

    expect(BusinessActionFormatter.destination(https).toString(), https);
    expect(BusinessActionFormatter.destination(email).toString(), email);
    expect(BusinessActionFormatter.destination(null), isNull);
    expect(BusinessActionFormatter.destination(''), isNull);
    expect(BusinessActionFormatter.destination(' seller.example '), isNull);
    expect(
      BusinessActionFormatter.destination('http://seller.example'),
      isNull,
    );
  });

  testWidgets(
    'UT-012 launches only after confirmation and safely handles false/throw',
    (tester) async {
      final messenger = RecordingMessenger();
      final launched = <Uri>[];
      late BuildContext context;
      await tester.pumpWidget(
        MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Builder(
              builder: (value) {
                context = value;
                return const SizedBox();
              },
            ),
          ),
        ),
      );
      final destination = Uri.parse(
        'https://seller.example/item?q=private#purchase',
      );

      await confirmAndLaunchExternalAction(
        context,
        uri: destination,
        launchUrl: (uri) async {
          launched.add(uri);
          return false;
        },
        confirmOpenLink: (_, _) async => false,
      );
      expect(launched, isEmpty);

      await confirmAndLaunchExternalAction(
        context,
        uri: destination,
        launchUrl: (uri) async {
          launched.add(uri);
          return false;
        },
        confirmOpenLink: (_, uri) async {
          expect(uri, destination);
          return true;
        },
      );
      expect(launched, [destination]);
      expect(messenger.calls.single.$2, "Couldn't open that link.");
      expect(messenger.calls.single.$2, isNot(contains('seller.example')));

      await confirmAndLaunchExternalAction(
        context,
        uri: destination,
        launchUrl: (_) => throw StateError('platform failure'),
        confirmOpenLink: (_, _) async => true,
      );
      expect(messenger.calls, hasLength(2));

      await confirmAndLaunchExternalAction(
        context,
        uri: Uri.parse('http://seller.example/insecure'),
        launchUrl: (uri) async {
          launched.add(uri);
          return true;
        },
        confirmOpenLink: (_, _) async => true,
      );
      expect(launched, [destination]);
    },
  );
}
