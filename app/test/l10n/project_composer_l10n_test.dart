import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-016 project composer and chooser strings are localised', () {
    final arb =
        jsonDecode(
              File('lib/l10n/app_en.arb').readAsStringSync(),
            )
            as Map<String, Object?>;

    const requiredKeys = [
      'postTypeRegularLabel',
      'postTypeRegularDescription',
      'postTypeProjectLabel',
      'postTypeProjectDescription',
      'projectComposerTitle',
      'projectComposerProjectTitleLabel',
      'projectComposerCraftTypeLabel',
      'projectComposerStatusLabel',
      'projectComposerMaterialsLabel',
      'projectComposerMaterialsAddHint',
      'projectComposerMaterialsAddAction',
      'projectComposerFieldDisabledLabel',
      'projectComposerMultiSelectMaxSelectedError',
      'projectComposerColoursLabel',
      'projectComposerDesignTagsLabel',
      'projectComposerAddPatternAction',
      'projectComposerMoreDetailsLabel',
      'projectComposerSewingProjectTypeLabel',
      'projectComposerKnittingProjectTypeLabel',
      'projectComposerCrochetProjectTypeLabel',
      'projectComposerQuiltingProjectTypeLabel',
      'projectComposerGaugeInvalidError',
      'projectComposerPhotoRequiredError',
    ];

    for (final key in requiredKeys) {
      final value = arb[key];
      expect(value, isA<String>(), reason: '$key should be localised');
      final text = value! as String;
      expect(text.trim(), isNotEmpty, reason: '$key should not be blank');
      expect(
        _containsEmoji(text),
        isFalse,
        reason: '$key should not use emoji',
      );
    }

    expect(arb['projectComposerColoursLabel'], 'Colours');
    expect(arb['projectComposerPhotoRequiredError'], 'Add at least one photo.');
  });
}

bool _containsEmoji(String text) {
  return RegExp(
    r'[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}]',
    unicode: true,
  ).hasMatch(text);
}
