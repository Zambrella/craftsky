import 'package:craftsky_app/feed/composer/post_edit_changes.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-002 clearing the last visible detail preserves opaque siblings', () {
    const initial = Project(
      common: ProjectCommon(craftType: 'social.craftsky.feed.defs#sewing'),
      details: SewingProjectDetails(
        sizeMade: 'M',
        extraFields: {'futureDetail': 'keep'},
      ),
    );
    expect(
      buildPublishedProjectPatch(
        initial: initial,
        current: initial.copyWith(details: null),
      ),
      {
        'details': {'sizeMade': null},
      },
    );
  });
  test(
    'UT-002 clearing the last visible pattern field preserves opaque siblings',
    () {
      const initial = Project(
        common: ProjectCommon(
          craftType: 'social.craftsky.feed.defs#sewing',
          pattern: ProjectPattern(
            name: 'Old name',
            extraFields: {'futurePattern': 'keep'},
          ),
        ),
      );
      final current = initial.copyWith(
        common: initial.common.copyWith(pattern: null),
      );
      expect(buildPublishedProjectPatch(initial: initial, current: current), {
        'common': {
          'pattern': {'name': null},
        },
      });
    },
  );
  for (final (craft, before, after, expected)
      in <(String, ProjectDetails, ProjectDetails, Map<String, Object?>)>[
        (
          'sewing',
          const SewingProjectDetails(sizeMade: 'M', fitNotes: 'Old'),
          const SewingProjectDetails(sizeMade: 'M'),
          {'fitNotes': null},
        ),
        (
          'knitting',
          const KnittingProjectDetails(
            needleSizeMm: '3',
            gauge: ProjectGauge(
              stitches: 20,
              rows: 30,
              measurement: 10,
              unit: 'cm',
            ),
          ),
          const KnittingProjectDetails(
            needleSizeMm: '4',
            gauge: ProjectGauge(
              stitches: 22,
              rows: 30,
              measurement: 10,
              unit: 'cm',
            ),
          ),
          {
            'needleSizeMm': '4',
            'gauge': {'stitches': 22},
          },
        ),
        (
          'crochet',
          const CrochetProjectDetails(hookSizeMm: '3'),
          const CrochetProjectDetails(hookSizeMm: '3', finishedSize: '20 cm'),
          {'finishedSize': '20 cm'},
        ),
        (
          'quilting',
          const QuiltingProjectDetails(size: 'Large', quiltingMethod: 'hand'),
          const QuiltingProjectDetails(size: 'Large'),
          {'quiltingMethod': null},
        ),
      ]) {
    test('UT-002 $craft details preserve untouched leaves', () {
      final initial = Project(
        common: ProjectCommon(craftType: 'social.craftsky.feed.defs#$craft'),
        details: before,
      );
      final current = initial.copyWith(details: after);
      expect(buildPublishedProjectPatch(initial: initial, current: current), {
        'details': expected,
      });
    });
  }
  test('UT-002 additional craft edits retain old identities and include new '
      'entries', () {
    const initial = Project(
      common: ProjectCommon(
        craftType: 'social.craftsky.feed.defs#sewing',
        additionalCrafts: [
          ProjectCraft(
            craftType: 'social.craftsky.feed.defs#knitting',
            extraFields: {'existingIndex': 0},
          ),
          ProjectCraft(
            craftType: 'social.craftsky.feed.defs#crochet',
            extraFields: {'existingIndex': 1, 'futureCraft': 'leave on server'},
          ),
        ],
      ),
    );
    final current = initial.copyWith(
      common: initial.common.copyWith(
        additionalCrafts: [
          initial.common.additionalCrafts!.last,
          const ProjectCraft(craftType: 'social.craftsky.feed.defs#embroidery'),
        ],
      ),
    );
    expect(buildPublishedProjectPatch(initial: initial, current: current), {
      'common': {
        'additionalCrafts': [
          {'existingIndex': 1},
          {'craftType': 'social.craftsky.feed.defs#embroidery'},
        ],
      },
    });
  });
  test(
    'UT-002 reordered retained materials send indices and changed leaves',
    () {
      const first = ProjectMaterial(
        text: ' Linen ',
        extraFields: {'existingIndex': 0, 'futureMaterial': 'kept on server'},
      );
      const second = ProjectMaterial(
        text: 'Thread',
        extraFields: {'existingIndex': 1},
      );
      const initial = Project(
        common: ProjectCommon(
          craftType: 'social.craftsky.feed.defs#sewing',
          materials: [first, second],
        ),
      );
      final current = initial.copyWith(
        common: initial.common.copyWith(
          materials: [
            second.copyWith(text: 'Silk thread'),
            first,
          ],
        ),
      );
      expect(buildPublishedProjectPatch(initial: initial, current: current), {
        'common': {
          'materials': [
            {'existingIndex': 1, 'text': 'Silk thread'},
            {'existingIndex': 0},
          ],
        },
      });
    },
  );
  test(
    'UT-002 project edits send supported leaf changes and explicit clears only',
    () {
      const initial = Project(
        common: ProjectCommon(
          craftType: 'social.craftsky.feed.defs#sewing',
          title: 'Before',
          pattern: ProjectPattern(
            name: 'My pattern',
            selfDrafted: true,
            extraFields: {'futurePattern': 'leave on server'},
          ),
          extraFields: {'futureCommon': 'kept on server'},
        ),
        extraFields: {'futureProject': true},
      );
      final current = initial.copyWith(
        common: initial.common.copyWith(
          title: null,
          pattern: initial.common.pattern!.copyWith(selfDrafted: false),
        ),
      );
      expect(buildPublishedProjectPatch(initial: initial, current: current), {
        'common': {
          'title': null,
          'pattern': {'selfDrafted': false},
        },
      });
      expect(
        buildPublishedProjectPatch(initial: initial, current: initial),
        isEmpty,
      );
    },
  );
}
