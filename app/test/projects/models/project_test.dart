import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('Project', () {
    test('UT-001 parses and serializes common camelCase fields', () {
      final json = {
        'common': {
          'craftType': 'social.craftsky.feed.defs#knitting',
          'status': 'social.craftsky.feed.defs#wip',
          'title': 'Hitchhiker Shawl',
          'duration': '3 weeks',
          'materials': [
            {'text': 'wool'},
            {'text': 'mohair'},
          ],
          'colors': ['blue', 'cream'],
          'designTags': ['social.craftsky.project.defs#stripes'],
          'tags': ['shawl', 'gift'],
        },
      };

      final project = ProjectMapper.fromMap(json);

      expect(project.common, isA<ProjectCommon>());
      expect(project.common.craftType, 'social.craftsky.feed.defs#knitting');
      expect(project.common.status, 'social.craftsky.feed.defs#wip');
      expect(project.common.title, 'Hitchhiker Shawl');
      expect(project.common.duration, '3 weeks');
      expect(project.common.materials, const [
        ProjectMaterial(text: 'wool'),
        ProjectMaterial(text: 'mohair'),
      ]);
      expect(project.common.colors, ['blue', 'cream']);
      expect(project.common.designTags, [
        'social.craftsky.project.defs#stripes',
      ]);
      expect(project.common.tags, ['shawl', 'gift']);
      expect(project.toMap(), json);

      final same = ProjectMapper.fromMap(json);
      expect(project, same);
      expect(
        project.copyWith.common(title: 'New title').common.title,
        'New title',
      );
    });

    test('UT-002 parses and round-trips pattern fields', () {
      final json = {
        'common': {
          'craftType': 'social.craftsky.feed.defs#sewing',
          'pattern': {
            'url': 'https://example.com/pattern',
            'name': 'Linen Dress',
            'difficulty': 'social.craftsky.feed.defs#intermediate',
            'designer': 'Pattern Designer',
            'publisher': 'Pattern Publisher',
          },
        },
      };

      final project = ProjectMapper.fromMap(json);

      expect(project.common.pattern, isA<ProjectPattern>());
      expect(project.common.pattern?.url, 'https://example.com/pattern');
      expect(project.common.pattern?.name, 'Linen Dress');
      expect(
        project.common.pattern?.difficulty,
        'social.craftsky.feed.defs#intermediate',
      );
      expect(project.common.pattern?.designer, 'Pattern Designer');
      expect(project.common.pattern?.publisher, 'Pattern Publisher');
      expect(project.toMap(), json);
    });

    test('UT-009 parses sparse common-only projects without defaults', () {
      final json = {
        'common': {'craftType': 'social.craftsky.feed.defs#embroidery'},
      };

      final project = ProjectMapper.fromMap(json);

      expect(project.common.craftType, 'social.craftsky.feed.defs#embroidery');
      expect(project.common.status, isNull);
      expect(project.common.pattern, isNull);
      expect(project.details, isNull);
      expect(project.toMap(), json);
    });

    test('UT-011 create serialization omits empty optional arrays', () {
      const project = Project(
        common: ProjectCommon(
          craftType: 'social.craftsky.feed.defs#embroidery',
          materials: [],
          colors: [],
          designTags: [],
          tags: [],
        ),
      );

      expect(project.toMap()['common'], containsPair('materials', <String>[]));
      expect(project.toCreateMap(), {
        'common': {'craftType': 'social.craftsky.feed.defs#embroidery'},
      });
    });

    test('UT-020 constructors do not enforce lexicon validation hints', () {
      final project = Project(
        common: ProjectCommon(
          craftType: 'social.craftsky.feed.defs#future-craft',
          title: 'x' * 1000,
          pattern: const ProjectPattern(url: 'not a uri'),
          materials: List.generate(
            25,
            (index) => ProjectMaterial(text: 'material-$index'),
          ),
        ),
        details: const KnittingProjectDetails(
          gauge: ProjectGauge(
            stitches: -1,
            rows: 0,
            measurement: -4,
            unit: 'yards',
          ),
        ),
      );

      expect(
        project.common.craftType,
        'social.craftsky.feed.defs#future-craft',
      );
      expect(project.common.pattern?.url, 'not a uri');
      expect(project.common.materials, hasLength(25));
      expect(project.toMap(), isA<Map<String, dynamic>>());
    });
  });
}
