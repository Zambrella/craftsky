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
      'projectComposerNextAction',
      'projectComposerDetailsPrompt',
      'projectComposerOptionalDetailsPrompt',
      'projectComposerProjectTitleLabel',
      'projectComposerProjectTitleHint',
      'projectComposerCraftTypeLabel',
      'projectComposerStatusLabel',
      'projectComposerMaterialsLabel',
      'projectComposerMaterialsAddHint',
      'projectComposerMaterialsAddAction',
      'projectComposerFieldDisabledLabel',
      'projectComposerMultiSelectMaxSelectedError',
      'projectComposerColoursLabel',
      'projectComposerColoursSearchHint',
      'projectComposerDesignTagsLabel',
      'projectComposerDesignTagsSearchHint',
      'projectComposerPatternSectionLabel',
      'projectComposerPatternInfoSectionLabel',
      'projectComposerMoreDetailsLabel',
      'projectComposerSewingProjectTypeLabel',
      'projectComposerKnittingProjectTypeLabel',
      'projectComposerCrochetProjectTypeLabel',
      'projectComposerQuiltingProjectTypeLabel',
      'projectComposerSizeMadeHint',
      'projectComposerFitNotesHint',
      'projectComposerGaugeStitchesHint',
      'projectComposerGaugeRowsHint',
      'projectComposerGaugeMeasurementHint',
      'projectComposerFinishedSizeHint',
      'projectComposerPatternNameHint',
      'projectComposerPatternUrlHint',
      'projectComposerPatternDesignerHint',
      'projectComposerPatternPublisherHint',
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
    expect(arb['projectComposerColoursSearchHint'], 'Search colours');
    expect(arb['projectComposerPatternNameLabel'], 'Pattern tag or name');
    expect(arb['projectComposerPatternInfoSectionLabel'], 'Pattern info');
    expect(arb['projectComposerPhotoRequiredError'], 'Add at least one photo.');
  });
}

bool _containsEmoji(String text) {
  return RegExp(
    r'[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}]',
    unicode: true,
  ).hasMatch(text);
}
