import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('input limits match canonical and embedded lexicon contracts', () {
    void check(String path, String propertyPath, RecordTextLimit limit) {
      Object? node = jsonDecode(File('../$path').readAsStringSync());
      for (final key in propertyPath.split('.')) {
        node = (node! as Map<String, dynamic>)[key];
      }
      final property = node! as Map<String, dynamic>;
      expect(property['maxGraphemes'], limit.graphemes, reason: propertyPath);
      expect(property['maxLength'], limit.bytes, reason: propertyPath);
    }

    const projectRoot = 'lexicon/social/craftsky/project';
    for (final entry in {
      'projectCommon.properties.title': RecordTextLimits.projectTitle,
      'pattern.properties.name': RecordTextLimits.patternText,
      'pattern.properties.designer': RecordTextLimits.patternText,
      'pattern.properties.publisher': RecordTextLimits.patternText,
      'material.properties.text': RecordTextLimits.material,
    }.entries) {
      check('$projectRoot/defs.json', 'defs.${entry.key}', entry.value);
    }
    check(
      '$projectRoot/sewing.json',
      'defs.details.properties.fitNotes',
      RecordTextLimits.fitNotes,
    );
    check(
      'appview/internal/lexicon/schema/project/sewing.json',
      'defs.details.properties.fitNotes',
      RecordTextLimits.fitNotes,
    );
    for (final craft in ['sewing', 'knitting', 'crochet', 'quilting']) {
      final field = switch (craft) {
        'sewing' => 'sizeMade',
        'quilting' => 'size',
        _ => 'finishedSize',
      };
      check(
        '$projectRoot/$craft.json',
        'defs.details.properties.$field',
        RecordTextLimits.projectSize,
      );
    }
    const post = 'lexicon/social/craftsky/feed/post.json';
    check(post, 'defs.main.record.properties.text', RecordTextLimits.post);
    check(post, 'defs.image.properties.alt', RecordTextLimits.imageAlt);
    const standard = 'appview/internal/lexicon/schema/standard/app/bsky';
    for (final entry in {
      'displayName': RecordTextLimits.displayName,
      'description': RecordTextLimits.bio,
      'pronouns': RecordTextLimits.pronouns,
    }.entries) {
      check(
        '$standard/actor/profile.json',
        'defs.main.record.properties.${entry.key}',
        entry.value,
      );
    }
    check(
      '$standard/embed/video.json',
      'defs.main.properties.alt',
      RecordTextLimits.videoAlt,
    );
    check(
      '$standard/richtext/facet.json',
      'defs.tag.properties.tag',
      RecordTextLimits.hashtag,
    );
    for (final entry in {
      'tagline': RecordTextLimits.businessTagline,
      'hoursNote': RecordTextLimits.businessHours,
      'serviceArea': RecordTextLimits.businessServiceArea,
    }.entries) {
      check(
        'lexicon/social/craftsky/business/profile.json',
        'defs.main.record.properties.${entry.key}',
        entry.value,
      );
    }
    for (final entry in {
      'name': RecordTextLimits.eventName,
      'summary': RecordTextLimits.eventSummary,
      'venueName': RecordTextLimits.eventVenue,
    }.entries) {
      check(
        'lexicon/social/craftsky/business/event.json',
        'defs.main.record.properties.${entry.key}',
        entry.value,
      );
    }
    check(
      'lexicon/social/craftsky/business/defs.json',
      'defs.product.properties.title',
      RecordTextLimits.productTitle,
    );
    check(
      'lexicon/social/craftsky/business/defs.json',
      'defs.image.properties.alt',
      RecordTextLimits.imageAlt,
    );
  });

  test('UTF-8 formatter preserves complete clusters and composition', () {
    const formatter = Utf8LengthLimitingTextInputFormatter(4);
    final result = formatter.formatEditUpdate(
      TextEditingValue.empty,
      const TextEditingValue(
        text: 'ééé',
        selection: TextSelection.collapsed(offset: 3),
      ),
    );
    expect(result.text, 'éé');
    expect(result.selection.extentOffset, 2);
    expect(
      formatter
          .formatEditUpdate(
            TextEditingValue.empty,
            const TextEditingValue(text: '👨‍👩‍👧‍👦'),
          )
          .text,
      isEmpty,
    );
    const composing = TextEditingValue(
      text: 'ééé',
      composing: TextRange(start: 0, end: 3),
    );
    expect(
      formatter.formatEditUpdate(TextEditingValue.empty, composing),
      composing,
    );
    expect(
      formatter
          .formatEditUpdate(
            composing,
            composing.copyWith(composing: TextRange.empty),
          )
          .text,
      'éé',
    );
  });
}
