import 'package:craftsky_app/projects/models/project.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('future fields survive known and unknown project model round trips', () {
    final raw = <String, dynamic>{
      'futureProject': true,
      'common': {
        'craftType': 'social.craftsky.feed.defs#knitting',
        'futureCommon': {'value': 'preserved'},
        'title': 'Original',
        'pattern': {'name': 'Aster', 'futurePattern': true},
        'materials': [
          {'text': 'wool', 'futureMaterial': true},
        ],
        'additionalCrafts': [
          {
            'craftType': 'social.craftsky.feed.defs#sewing',
            'futureCraft': true,
          },
        ],
      },
      'details': {
        r'$type': knittingProjectDetailsType,
        'futureDetail': true,
        'gauge': {
          'stitches': 22,
          'measurement': 10,
          'unit': 'cm',
          'futureGauge': true,
        },
      },
    };
    final project = ProjectMapper.fromMap(raw);
    expect(project.toMap(), raw);
    final edited = project.copyWith.common(title: null);
    final encoded = edited.toMap();
    expect((encoded['common'] as Map).containsKey('title'), isFalse);
    expect((encoded['common'] as Map)['futureCommon'], {'value': 'preserved'});
    expect((encoded['details'] as Map)['futureDetail'], isTrue);
  });

  test(
    'a public extraFields property is not confused with internal storage',
    () {
      final raw = <String, dynamic>{
        'common': {
          'craftType': 'custom',
          'extraFields': {'public': true},
        },
      };
      expect(ProjectMapper.fromMap(raw).toMap(), raw);
    },
  );
}
