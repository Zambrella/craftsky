import 'package:craftsky_app/projects/composer/project_composer_draft_state.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Project composer draft state', () {
    test('UT-014 unchanged state has no draft', () {
      expect(
        ProjectComposerDraftState.hasDraft(
          bodyText: '',
          initialBodyText: '',
          imageCount: 0,
          formValues: const {},
        ),
        isFalse,
      );
    });

    test('UT-014 detects body text, image and metadata changes', () {
      expect(
        ProjectComposerDraftState.hasDraft(
          bodyText: 'hello',
          initialBodyText: '',
          imageCount: 0,
          formValues: const {},
        ),
        isTrue,
      );
      expect(
        ProjectComposerDraftState.hasDraft(
          bodyText: '',
          initialBodyText: '',
          imageCount: 1,
          formValues: const {},
        ),
        isTrue,
      );
      expect(
        ProjectComposerDraftState.hasDraft(
          bodyText: '',
          initialBodyText: '',
          imageCount: 0,
          formValues: const {ProjectComposerFields.title: 'Hat'},
        ),
        isTrue,
      );
    });

    test('UT-014 collapsed detail values still count as draft changes', () {
      expect(
        ProjectComposerDraftState.hasDraft(
          bodyText: '',
          initialBodyText: '',
          imageCount: 0,
          formValues: const {
            ProjectComposerFields.knittingFinishedSize: '42in chest',
          },
        ),
        isTrue,
      );
    });

    test('UT-014 treats initial default form values as unchanged', () {
      expect(
        ProjectComposerDraftState.hasDraft(
          bodyText: '',
          initialBodyText: '',
          imageCount: 0,
          formValues: const {
            ProjectComposerFields.status:
                ProjectOptionCatalogs.finishedStatusToken,
          },
          initialFormValues: const {
            ProjectComposerFields.status:
                ProjectOptionCatalogs.finishedStatusToken,
          },
        ),
        isFalse,
      );
    });

    test('UT-014 detects material entry changes', () {
      expect(
        ProjectComposerDraftState.hasDraft(
          bodyText: '',
          initialBodyText: '',
          imageCount: 0,
          formValues: const {
            ProjectComposerFields.materials: [
              ProjectMaterial(text: 'Wool roving'),
            ],
          },
        ),
        isTrue,
      );
    });
  });
}
