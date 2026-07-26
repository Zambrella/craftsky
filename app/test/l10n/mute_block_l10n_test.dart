import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-013 every supported locale defines mute and block semantics', () {
    const requiredKeys = <String>[
      'profileMoreActions',
      'profileMuteAction',
      'profileUnmuteAction',
      'profileBlockAction',
      'profileUnblockAction',
      'profileMuteAnnotation',
      'profileBlockingAnnotation',
      'profileBlockedByAnnotation',
      'profileMutualBlockAnnotation',
      'profileRelationshipError',
      'profileMuteSuccess',
      'profileUnmuteSuccess',
      'profileBlockSuccess',
      'profileUnblockSuccess',
      'profileBlockConfirmTitle',
      'profileBlockConfirmBody',
      'profileUnblockConfirmTitle',
      'profileUnblockConfirmBody',
      'destructiveActionHint',
      'settingsMutedAccounts',
      'settingsBlockedAccounts',
      'settingsMutedAccountsEmpty',
      'settingsBlockedAccountsEmpty',
      'settingsMutedAccountsError',
      'settingsBlockedAccountsError',
      'relationshipListRetry',
      'relationshipListLoadMore',
      'relationshipListUnmute',
      'relationshipListUnblock',
      'relationshipListMutationError',
      'postMutedPlaceholder',
      'postUnavailablePlaceholder',
      'postRevealAction',
    ];

    for (final locale in AppLocalizations.supportedLocales) {
      final arb =
          jsonDecode(
                File(
                  'lib/l10n/app_${locale.languageCode}.arb',
                ).readAsStringSync(),
              )
              as Map<String, Object?>;
      for (final key in requiredKeys) {
        final value = arb[key];
        expect(
          value,
          isA<String>(),
          reason: '${locale.languageCode}: $key must be localized',
        );
        expect(
          (value! as String).trim(),
          isNotEmpty,
          reason: '${locale.languageCode}: $key must not be blank',
        );
        expect(
          arb['@$key'],
          isA<Map<String, Object?>>(),
          reason: '${locale.languageCode}: $key must describe its semantics',
        );
      }
    }
  });
}
