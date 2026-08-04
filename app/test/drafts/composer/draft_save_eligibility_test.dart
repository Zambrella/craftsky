import 'package:craftsky_app/drafts/composer/draft_save_eligibility.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Draft save eligibility', () {
    test('allows meaningful top-level standard and project drafts only', () {
      const textChange = DraftComposerChanges(textChanged: true);
      const untouched = DraftComposerChanges();

      expect(
        DraftSaveEligibility.canSave(
          kind: DraftComposerKind.standard,
          changes: textChange,
          mediaReady: true,
        ),
        isTrue,
      );
      expect(
        DraftSaveEligibility.canSave(
          kind: DraftComposerKind.project,
          changes: textChange,
          mediaReady: true,
        ),
        isTrue,
      );
      expect(
        DraftSaveEligibility.canSave(
          kind: DraftComposerKind.standard,
          changes: untouched,
          mediaReady: true,
        ),
        isFalse,
      );
      expect(
        DraftSaveEligibility.canSave(
          kind: DraftComposerKind.quote,
          changes: textChange,
          mediaReady: true,
        ),
        isFalse,
      );
      expect(
        DraftSaveEligibility.canSave(
          kind: DraftComposerKind.reply,
          changes: textChange,
          mediaReady: true,
        ),
        isFalse,
      );
    });

    test(
      'recognizes every approved deliberate change and requires ready media',
      () {
        const changes = [
          DraftComposerChanges(textChanged: true),
          DraftComposerChanges(attachmentChanged: true),
          DraftComposerChanges(altTextChanged: true),
          DraftComposerChanges(languageChanged: true),
          DraftComposerChanges(projectFieldChanged: true),
          DraftComposerChanges(scheduleChanged: true),
        ];

        for (final change in changes) {
          expect(
            DraftSaveEligibility.canSave(
              kind: DraftComposerKind.project,
              changes: change,
              mediaReady: true,
            ),
            isTrue,
          );
          expect(
            DraftSaveEligibility.canSave(
              kind: DraftComposerKind.project,
              changes: change,
              mediaReady: false,
            ),
            isFalse,
          );
        }
      },
    );
  });
}
