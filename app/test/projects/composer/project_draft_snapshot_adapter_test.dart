import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/composer/project_draft_snapshot_adapter.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('whitelists partial project fields and adapts materials explicitly', () {
    const adapter = ProjectDraftSnapshotAdapter();
    final encoded = adapter.encodeKnownFields({
      ProjectComposerFields.craftType: 'knitting',
      ProjectComposerFields.title: 'Half-finished cardigan',
      ProjectComposerFields.materials: const [
        ProjectMaterial(
          text: 'Wool',
          facets: [
            {'type': 'mention'},
          ],
        ),
      ],
      ProjectComposerFields.knittingGaugeRows: null,
      ProjectComposerFields.colours: <dynamic>['blue', 'cream'],
      ProjectComposerFields.designTags: <dynamic>['cables'],
      'unknownRuntimeField': Object(),
    });

    expect(encoded, isNot(contains('unknownRuntimeField')));
    expect(encoded[ProjectComposerFields.knittingGaugeRows], isNull);
    expect(encoded[ProjectComposerFields.materials], [
      {
        'text': 'Wool',
        'facets': [
          {'type': 'mention'},
        ],
      },
    ]);

    final decoded = adapter.decodeKnownFields(encoded);
    expect(decoded[ProjectComposerFields.title], 'Half-finished cardigan');
    expect(
      decoded[ProjectComposerFields.materials],
      isA<List<ProjectMaterial>>(),
    );
    expect(
      (decoded[ProjectComposerFields.materials]! as List<ProjectMaterial>)
          .single
          .text,
      'Wool',
    );
    expect(
      decoded[ProjectComposerFields.colours],
      isA<List<String>>().having((values) => values, 'values', [
        'blue',
        'cream',
      ]),
    );
    expect(
      decoded[ProjectComposerFields.designTags],
      isA<List<String>>().having((values) => values, 'values', ['cables']),
    );
  });

  test('builds an incomplete project draft without publication validation', () {
    const adapter = ProjectDraftSnapshotAdapter();

    final request = adapter.toWriteRequest(
      id: '96ad7199-292f-4388-a6cd-b4f74230116b',
      owner: AccountKey('did:plc:alice'),
      body: '',
      languages: const ['en'],
      schedule: const DraftScheduleIntent.now(),
      formValues: const {ProjectComposerFields.title: 'Only a title'},
      images: const [],
    );

    expect(request.kind, LocalPostDraftKind.project);
    final content = request.content as ProjectDraftContent;
    expect(content.body, isEmpty);
    expect(
      content.knownProjectFieldValues[ProjectComposerFields.title],
      'Only a title',
    );
  });
}
