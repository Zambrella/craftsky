import 'package:craftsky_app/search/models/project_search_filters.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-006 project filters preserve repeated supported families', () {
    const filters = ProjectSearchFilters(
      craftType: ['knitting', 'crochet'],
      projectType: ['sweater', 'shawl'],
      patternDifficulty: ['easy', 'advanced'],
      color: ['blue', 'red'],
      material: ['wool', 'cotton'],
      designTag: ['cables', 'lace'],
      projectTag: ['gift', 'kal'],
    );

    expect(filters.toQueryParameters(), {
      'craftType': ['knitting', 'crochet'],
      'projectType': ['sweater', 'shawl'],
      'patternDifficulty': ['easy', 'advanced'],
      'color': ['blue', 'red'],
      'material': ['wool', 'cotton'],
      'designTag': ['cables', 'lace'],
      'projectTag': ['gift', 'kal'],
    });
  });
}
