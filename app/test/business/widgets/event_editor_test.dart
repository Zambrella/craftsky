import 'dart:async';
import 'dart:typed_data';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/pages/event_editor_dialog.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_image_failure.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/shared/image/basic_image_editor.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:image_picker/image_picker.dart';
import 'package:material_ui/material_ui.dart';

import '../../test_support/deterministic_pump.dart';
import '../../test_support/session_registry_storage.dart';
import '../accessibility_test_helpers.dart';

void main() {
  for (final (error, message, reports) in <(Object, String, bool)>[
    (
      const ProfileImagePickException(
        reason: ProfileImageFailure.tooManyPixels,
      ),
      'That photo is over 48 megapixels. Choose a smaller photo or reduce its '
          'resolution and try again.',
      false,
    ),
    (
      const ProfileImagePickException(reason: ProfileImageFailure.fileTooLarge),
      'That photo is over 100 MB. Choose a smaller file and try again.',
      false,
    ),
    (
      StateError('PRIVATE_IMAGE_DEPENDENCY_PROSE'),
      'The image could not be uploaded. Try again.',
      true,
    ),
  ]) {
    testWidgets('event photo failure shows actionable copy: $message', (
      tester,
    ) async {
      final reporter = _ImageReporter();
      await tester.pumpWidget(
        _app(
          EventEditorDialog(
            initialDraft: _draft(),
            onSubmit: (_) async => false,
            pickImage: (_, _) async =>
                Error.throwWithStackTrace(error, StackTrace.current),
          ),
          reporter: reporter,
        ),
      );
      final add = find.byKey(const Key('event-add-image'));
      await tester.ensureVisible(add);
      await tester.tap(add);
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('event-choose-photos')));
      await tester.pumpAndSettle();
      expect(find.text(message), findsOneWidget);
      expect(find.textContaining('PRIVATE_IMAGE'), findsNothing);
      expect(reporter.errors, reports ? [same(error)] : isEmpty);
      if (reports) {
        expect(reporter.contexts.single.operation, 'business.event.image');
        expect(reporter.stacks.single, isNotNull);
      }
    });
  }

  testWidgets('Done dismisses event keyboard without submitting', (
    tester,
  ) async {
    addTearDown(tester.view.resetViewInsets);
    var submitted = false;
    await tester.pumpWidget(
      _app(
        EventEditorDialog(
          initialDraft: _draft(),
          onSubmit: (_) async {
            submitted = true;
            return false;
          },
        ),
      ),
    );
    final field = find.byKey(const ValueKey('event-summary'));
    await tester.ensureVisible(field);
    await tester.pumpAndSettle();
    const text = 'A craft fair\nwith workshops';
    await tester.enterText(field, text);
    tester.view.viewInsets = FakeViewPadding(
      bottom: 300 * tester.view.devicePixelRatio,
    );
    await tester.pumpAndSettle();

    final done = find.byKey(const Key('event-composer-done'));
    expect(done, findsOneWidget);
    expect(find.byKey(const ValueKey('event-submit')), findsNothing);
    await tester.tap(done);
    await tester.pump();
    expect(tester.testTextInput.isVisible, isFalse);
    expect(tester.widget<TextField>(field).controller!.text, text);
    expect(submitted, isFalse);

    tester.view.resetViewInsets();
    await tester.pumpAndSettle();
    expect(done, findsNothing);
    expect(find.byKey(const ValueKey('event-submit')), findsOneWidget);
  });

  for (final constraint in businessAccessibilityMatrix) {
    testWidgets('AT-012 REG-010 event editor fits '
        '${businessConstraintLabel(constraint)}', (tester) async {
      await setBusinessAccessibilityConstraint(tester, constraint);
      final semantics = tester.ensureSemantics();
      final submit = Completer<bool>();
      await tester.pumpWidget(
        _app(
          EventEditorDialog(
            initialDraft: _draft(),
            onSubmit: (_) => submit.future,
          ),
        ),
      );

      expect(
        tester.getSemantics(find.byKey(const ValueKey('event-name'))).label,
        contains('Event name'),
      );
      expect(
        tester.getSemantics(find.byKey(const ValueKey('event-name'))).label,
        contains('required'),
      );
      expect(find.byType(BrandTextField), findsWidgets);
      expect(find.widgetWithText(ChunkyButton, 'Create event'), findsOneWidget);
      expect(
        tester
            .widget<CraftskySearchableMultiSelectInput<String>>(
              find.byType(CraftskySearchableMultiSelectInput<String>),
            )
            .values,
        contains('vendor'),
      );
      await expectKeyboardFocus(tester);
      await tester.scrollUntilVisible(
        find.byKey(const ValueKey('event-registration-uri')),
        250,
        scrollable: find.byType(Scrollable).last,
      );
      expect(
        tester
            .getSemantics(find.byKey(const ValueKey('event-registration-uri')))
            .label,
        contains('Registration link'),
      );

      await tester.tap(find.byKey(const ValueKey('event-submit')));
      await tester.pump();
      expect(
        tester.getSemantics(find.byType(CircularProgressIndicator)).label,
        contains('Saving'),
      );
      expectNoAccessibilityLayoutException(tester);
      submit.complete(false);
      await pumpUntilAbsent(
        tester,
        find.byType(CircularProgressIndicator),
        description: 'the event submission progress indicator',
      );
      semantics.dispose();
    });
  }

  testWidgets('AT-007 requires event name, roles, and valid boundaries', (
    tester,
  ) async {
    await tester.pumpWidget(_app(const EventEditorDialog()));
    expect(find.text('Choose at least one role.'), findsNothing);
    expect(find.text('Autumn yarn fair'), findsOneWidget);
    final roleInput = find.byKey(const Key('event-roles-search-input'));
    await tester.ensureVisible(roleInput);
    await tester.tap(roleInput);
    await tester.pumpAndSettle();
    final role = find.byKey(const Key('event-roles-option-organizer'));
    await tester.ensureVisible(role);
    await tester.tap(role);
    await tester.pump();
    expect(find.text('Choose at least one role.'), findsNothing);
    await tester.tap(find.byKey(const Key('event-roles-remove-organizer')));
    await tester.pump();
    expect(find.text('Choose at least one role.'), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await pumpUntilFound(
      tester,
      find.text('Add an event name.'),
      description: 'the required event-name validation',
    );

    expect(find.text('Add an event name.'), findsOneWidget);
    expect(find.text('Choose at least one role.'), findsOneWidget);
    expect(find.text('Enter a valid start and end.'), findsNWidgets(2));
  });

  testWidgets('end boundary explains when it is not after start', (
    tester,
  ) async {
    await tester.pumpWidget(
      _app(
        EventEditorDialog(
          initialDraft: _draft(
            startsAt: DateTime(2026, 9, 5, 12),
            endsAt: DateTime(2026, 9, 5, 10),
          ),
        ),
      ),
    );

    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await pumpUntilFound(
      tester,
      find.text('End must be after start.'),
      description: 'the invalid event-boundary validation',
    );

    expect(find.text('End must be after start.'), findsOneWidget);
  });

  testWidgets('event boundaries use date and time pickers', (tester) async {
    await tester.pumpWidget(_app(EventEditorDialog(initialDraft: _draft())));

    expect(
      tester.getSemantics(find.byKey(const ValueKey('event-start'))).label,
      contains('Start'),
    );
    expect(
      tester.getSemantics(find.byKey(const ValueKey('event-end'))).label,
      contains('End'),
    );
    expect(find.text('All-day event'), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('event-start')));
    await tester.pumpAndSettle();
    expect(find.byType(DatePickerDialog), findsOneWidget);

    await tester.tap(find.text('OK'));
    await tester.pumpAndSettle();
    expect(find.byType(TimePickerDialog), findsOneWidget);

    await tester.tap(find.text('OK'));
    await tester.pumpAndSettle();
    expect(find.byType(DatePickerDialog), findsNothing);
    expect(find.byType(TimePickerDialog), findsNothing);

    await tester.scrollUntilVisible(
      find.byKey(const ValueKey('event-time-zone')),
      250,
      scrollable: find.byType(Scrollable).last,
    );
    expect(
      tester.getSemantics(find.byKey(const ValueKey('event-time-zone'))).label,
      contains('Timezone'),
    );
  });

  testWidgets(
    'all-day date picker and exclusive end are preserved when saving',
    (tester) async {
      BusinessEventDraft? submitted;
      await tester.pumpWidget(
        _app(
          EventEditorDialog(
            initialDraft: _draft(
              startsAt: DateTime(2026, 9, 5),
              endsAt: DateTime(2026, 9, 6),
              isAllDay: true,
            ),
            onSubmit: (draft) async {
              submitted = draft;
              return true;
            },
          ),
        ),
      );

      expect(find.text('All-day event'), findsOneWidget);
      await tester.ensureVisible(find.byKey(const ValueKey('event-start')));
      await tester.tap(find.byKey(const ValueKey('event-start')));
      await tester.pumpAndSettle();
      expect(find.byType(DatePickerDialog), findsOneWidget);
      await tester.tap(find.text('OK'));
      await tester.pumpAndSettle();
      expect(find.byType(TimePickerDialog), findsNothing);
      await tester.tap(find.byKey(const ValueKey('event-submit')));
      await tester.pumpAndSettle();

      expect(submitted?.isAllDay, isTrue);
      expect(submitted?.startsAt, DateTime.utc(2026, 9, 5));
      expect(submitted?.endsAt, DateTime.utc(2026, 9, 6));
    },
  );

  testWidgets(
    'switching to all-day rounds boundaries and advances a same-day end',
    (tester) async {
      BusinessEventDraft? submitted;
      await tester.pumpWidget(
        _app(
          EventEditorDialog(
            initialDraft: _draft(
              startsAt: DateTime(2026, 9, 5, 10),
              endsAt: DateTime(2026, 9, 5, 12),
            ),
            onSubmit: (draft) async {
              submitted = draft;
              return true;
            },
          ),
        ),
      );
      await tester.ensureVisible(find.byKey(const ValueKey('event-all-day')));
      await tester.tap(find.byKey(const ValueKey('event-all-day')));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const ValueKey('event-submit')));
      await tester.pumpAndSettle();
      expect(submitted?.isAllDay, isTrue);
      expect(submitted?.startsAt, DateTime.utc(2026, 9, 5));
      expect(submitted?.endsAt, DateTime.utc(2026, 9, 6));
    },
  );

  testWidgets('DST gaps show a correction message without submitting', (
    tester,
  ) async {
    var submissions = 0;
    await tester.pumpWidget(
      _app(
        EventEditorDialog(
          initialDraft: _draft(
            startsAt: DateTime.utc(2026, 3, 29, 1, 30),
            endsAt: DateTime.utc(2026, 3, 29, 3),
          ),
          onSubmit: (_) async {
            submissions++;
            return true;
          },
        ),
      ),
    );

    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();

    expect(submissions, 0);
    expect(
      find.text(
        'This time does not exist in the selected time zone. '
        'Choose another time.',
      ),
      findsOneWidget,
    );
  });

  testWidgets(
    'AT-007 authors complete timed event on compact and wide layouts',
    (tester) async {
      BusinessEventDraft? submitted;
      for (final size in const [Size(390, 844), Size(1100, 900)]) {
        await tester.binding.setSurfaceSize(size);
        await tester.pumpWidget(
          _app(
            EventEditorDialog(
              initialDraft: _draft(),
              onSubmit: (draft) async {
                submitted = draft;
                return true;
              },
            ),
          ),
        );

        await tester.enterText(
          find.byKey(const ValueKey('event-summary')),
          'Meet local makers',
        );
        await tester.tap(find.byKey(const ValueKey('event-submit')));
        await tester.pumpAndSettle();

        expect(submitted?.summary, 'Meet local makers');
        expect(tester.takeException(), isNull);
      }
      await tester.binding.setSurfaceSize(null);
    },
  );

  testWidgets('AT-007 existing event image loads in shared attachment editor', (
    tester,
  ) async {
    final image = ExistingBusinessImageDraft(
      BusinessImageView(
        cid: 'bafy-saved',
        mime: 'image/jpeg',
        size: 123,
        alt: 'Saved fair',
        thumb: 'https://cdn.example/thumb',
        fullsize: 'https://cdn.example/full',
      ),
    );
    BusinessEventDraft? submitted;
    await tester.pumpWidget(
      _app(
        EventEditorDialog(
          initialDraft: _draft(image: image),
          onSubmit: (draft) async {
            submitted = draft;
            return false;
          },
        ),
      ),
    );

    final summary = find.byKey(const Key('event-edit-images'));
    await tester.ensureVisible(summary);
    await tester.tap(summary);
    await tester.pumpAndSettle();
    final preview = find.byKey(const Key('event-preview-image'));
    await tester.scrollUntilVisible(
      preview,
      250,
      scrollable: find.byType(Scrollable).last,
    );

    expect(preview, findsOneWidget);
    expect(
      tester
          .widget<CachedNetworkImage>(find.byType(CachedNetworkImage))
          .imageUrl,
      'https://cdn.example/thumb',
    );
    expect(find.byKey(const Key('event-alt-image')), findsOneWidget);
    expect(find.byKey(const Key('event-remove-image')), findsOneWidget);

    await tester.enterText(
      find.byKey(const Key('event-alt-image')),
      'Updated event description',
    );
    await tester.pageBack();
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();
    expect(submitted?.image.alt, 'Updated event description');

    await tester.ensureVisible(summary);
    await tester.tap(summary);
    await tester.pumpAndSettle();
    final remove = find.byKey(const Key('event-remove-image'));
    await tester.scrollUntilVisible(
      remove,
      250,
      scrollable: find.byType(Scrollable).last,
    );
    await tester.tap(remove);
    await tester.pump(const Duration(milliseconds: 300));
    await tester.pageBack();
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();
    expect(submitted?.image, isA<RemovedBusinessImageDraft>());
  });

  testWidgets('event image uploads only when the event is submitted', (
    tester,
  ) async {
    final prepared = _preparedImage();
    var uploadCount = 0;
    ImageSource? selectedSource;
    BusinessEventDraft? submitted;
    await tester.pumpWidget(
      _app(
        EventEditorDialog(
          initialDraft: _draft(),
          pickImage: (source, onPreviewReady) async {
            selectedSource = source;
            onPreviewReady(prepared.bytes);
            return prepared;
          },
          uploadImage: (image) async {
            uploadCount++;
            expect(identical(image, prepared), isTrue);
            return _uploadedImage('bafy-event-image');
          },
          onSubmit: (draft) async {
            submitted = draft;
            return true;
          },
        ),
      ),
    );

    final addImage = find.byKey(const Key('event-add-image'));
    await tester.ensureVisible(addImage);
    await tester.tap(addImage);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('event-take-photo')));
    await tester.pumpAndSettle();

    expect(uploadCount, 0);
    expect(selectedSource, ImageSource.camera);
    expect(find.byKey(const Key('event-preview-image')), findsOneWidget);

    if (find.byKey(const Key('event-photo-details')).evaluate().isNotEmpty) {
      await tester.pageBack();
      await tester.pumpAndSettle();
    }
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();

    expect(uploadCount, 1);
    final submittedImage = submitted!.image as UploadedBusinessImageDraft;
    expect(submittedImage.cid, 'bafy-event-image');
    expect(submittedImage.aspectRatio?.width, 1);
    expect(submittedImage.aspectRatio?.height, 1);
  });

  for (final outcome in ['apply', 'cancel', 'fail']) {
    testWidgets(
      'event photo details editor $outcome preserves the draft '
      'and defers upload',
      (tester) async {
        final prepared = _preparedImage();
        final reporter = _ImageReporter();
        final failure = StateError('PRIVATE_EVENT_EDIT_FAILURE');
        var edits = 0;
        var uploads = 0;
        BusinessEventDraft? submitted;
        await tester.pumpWidget(
          _app(
            EventEditorDialog(
              initialDraft: _draft(),
              pickImage: (_, _) async => prepared,
              editImage: (request) async {
                edits++;
                expect(request.mode, BasicImageEditMode.composer);
                expect(identical(request.bytes, prepared.bytes), isTrue);
                if (outcome == 'cancel') return null;
                if (outcome == 'fail') {
                  throw failure;
                }
                return BasicImageEditOutput(
                  bytes: Uint8List.fromList(
                    img.encodePng(
                      img.Image(width: 20, height: 10, numChannels: 4),
                    ),
                  ),
                  mimeType: 'image/png',
                );
              },
              uploadImage: (image) async {
                uploads++;
                if (outcome == 'apply') {
                  expect(image.width, 20);
                  expect(image.height, 10);
                } else {
                  expect(identical(image, prepared), isTrue);
                }
                return _uploadedImage('event-edited-photo');
              },
              onSubmit: (draft) async {
                submitted = draft;
                return false;
              },
            ),
            reporter: reporter,
          ),
        );
        final add = find.byKey(const Key('event-add-image'));
        await tester.ensureVisible(add);
        await tester.tap(add);
        await tester.pumpAndSettle();
        await tester.tap(find.byKey(const Key('event-choose-photos')));
        await tester.pumpAndSettle();
        expect(find.byKey(const Key('event-photo-details')), findsOneWidget);
        expect(find.byKey(const Key('event-add-image')), findsNothing);
        expect(find.byKey(const Key('event-move-up-image')), findsNothing);
        expect(find.byKey(const Key('event-move-down-image')), findsNothing);
        expect(find.text('One photo'), findsOneWidget);
        final alt = find.byKey(const Key('event-alt-image'));
        await tester.ensureVisible(alt);
        await tester.enterText(alt, 'A craft fair poster');
        tester.testTextInput.hide();
        final edit = find.byKey(const Key('event-edit-image-image'));
        await tester.ensureVisible(edit);
        expect(edit, findsOneWidget);
        // The host codec uses a real isolate; await the same edit callback
        // outside the widget clock so preparation can finish deterministically.
        final attachment = tester.widget<ComposerImageAttachmentSection>(
          find.byType(ComposerImageAttachmentSection, skipOffstage: false),
        );
        await tester.runAsync(() => attachment.onEditImage!('image'));
        await tester.pumpAndSettle();
        await pumpUntilFound(
          tester,
          edit,
          description: 'the image editor finishing',
        );
        expect(edits, 1);
        expect(uploads, 0);
        expect(find.textContaining('PRIVATE_EVENT_EDIT'), findsNothing);
        expect(reporter.errors, outcome == 'fail' ? [same(failure)] : isEmpty);
        await tester.pageBack();
        await tester.pumpAndSettle();
        expect(find.byKey(const Key('event-edit-images')), findsOneWidget);
        await tester.ensureVisible(find.byKey(const Key('event-edit-images')));
        await tester.tap(find.byKey(const Key('event-edit-images')));
        await tester.pumpAndSettle();
        expect(
          tester.widget<BrandTextField>(alt).controller!.text,
          'A craft fair poster',
        );
        await tester.pageBack();
        await tester.pumpAndSettle();
        await tester.tap(find.byKey(const ValueKey('event-submit')));
        await tester.pumpAndSettle();
        expect(uploads, 1);
        expect(submitted?.image.alt, 'A craft fair poster');
      },
    );
  }

  testWidgets('failed submit upload keeps the prepared event image for retry', (
    tester,
  ) async {
    final prepared = _preparedImage();
    var uploadCount = 0;
    var submitCount = 0;
    await tester.pumpWidget(
      _app(
        EventEditorDialog(
          initialDraft: _draft(),
          pickImage: (_, _) async => prepared,
          uploadImage: (_) async {
            uploadCount++;
            if (uploadCount == 1) throw Exception('upload failed');
            return _uploadedImage('bafy-retried-event-image');
          },
          onSubmit: (_) async {
            submitCount++;
            return true;
          },
        ),
      ),
    );

    final addImage = find.byKey(const Key('event-add-image'));
    await tester.ensureVisible(addImage);
    await tester.tap(addImage);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('event-choose-photos')));
    await tester.pumpAndSettle();
    if (find.byKey(const Key('event-photo-details')).evaluate().isNotEmpty) {
      await tester.pageBack();
      await tester.pumpAndSettle();
    }
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();

    expect(uploadCount, 1);
    expect(submitCount, 0);
    expect(find.byKey(const Key('event-thumbnail-image')), findsOneWidget);
    expect(find.text('The image could not be uploaded. Try again.'), findsOne);

    if (find.byKey(const Key('event-photo-details')).evaluate().isNotEmpty) {
      await tester.pageBack();
      await tester.pumpAndSettle();
    }
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();

    expect(uploadCount, 2);
    expect(submitCount, 1);
  });

  testWidgets('mutation retry reuses the successful event image upload', (
    tester,
  ) async {
    final prepared = _preparedImage();
    var uploadCount = 0;
    var submitCount = 0;
    await tester.pumpWidget(
      _app(
        EventEditorDialog(
          initialDraft: _draft(),
          pickImage: (_, _) async => prepared,
          uploadImage: (_) async {
            uploadCount++;
            return _uploadedImage('bafy-reusable-event-image');
          },
          onSubmit: (_) async {
            submitCount++;
            return submitCount > 1;
          },
        ),
      ),
    );

    final addImage = find.byKey(const Key('event-add-image'));
    await tester.ensureVisible(addImage);
    await tester.tap(addImage);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('event-choose-photos')));
    await tester.pumpAndSettle();

    if (find.byKey(const Key('event-photo-details')).evaluate().isNotEmpty) {
      await tester.pageBack();
      await tester.pumpAndSettle();
    }
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();
    expect(uploadCount, 1);
    expect(submitCount, 1);

    if (find.byKey(const Key('event-photo-details')).evaluate().isNotEmpty) {
      await tester.pageBack();
      await tester.pumpAndSettle();
    }
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();
    expect(uploadCount, 1);
    expect(submitCount, 2);
  });

  testWidgets('AT-007 guards unsaved work before closing', (tester) async {
    var confirmations = 0;
    await tester.pumpWidget(
      _app(
        EventEditorDialog(
          initialDraft: _draft(),
          confirmDiscard: () async {
            confirmations++;
            return false;
          },
        ),
      ),
    );
    await tester.enterText(
      find.byKey(const ValueKey('event-name')),
      'Changed name',
    );

    await tester.tap(find.byTooltip('Close'));
    await tester.pumpAndSettle();

    expect(confirmations, 1);
    expect(find.byType(EventEditorDialog), findsOneWidget);
  });

  testWidgets('guards an incomplete dirty event before closing', (
    tester,
  ) async {
    var confirmations = 0;
    await tester.pumpWidget(
      _app(
        EventEditorDialog(
          confirmDiscard: () async {
            confirmations++;
            return false;
          },
        ),
      ),
    );

    await tester.enterText(
      find.byKey(const ValueKey('event-name')),
      'Incomplete event',
    );
    await tester.tap(find.byTooltip('Close'));
    await tester.pumpAndSettle();

    expect(confirmations, 1);
    expect(find.byType(EventEditorDialog), findsOneWidget);
  });

  testWidgets('AT-006 unknown status can be replaced and corrected', (
    tester,
  ) async {
    BusinessEventDraft? submitted;
    await tester.pumpWidget(
      _app(
        EventEditorDialog(
          event: _event(status: 'independent-status'),
          onSubmit: (draft) async {
            submitted = draft;
            return true;
          },
        ),
      ),
    );

    expect(tester.takeException(), isNull);
    expect(find.text('Other: Independent status'), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();
    expect(submitted, isNull);
    await tester.scrollUntilVisible(
      find.byKey(const ValueKey('event-status')),
      250,
      scrollable: find.byType(Scrollable).last,
    );
    await tester.tap(find.byKey(const ValueKey('event-status')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Scheduled').last);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();

    expect(submitted?.status, 'scheduled');
  });

  testWidgets('AT-007 unknown mode can be replaced and corrected', (
    tester,
  ) async {
    BusinessEventDraft? submitted;
    await tester.pumpWidget(
      _app(
        EventEditorDialog(
          event: _event(mode: 'independent-mode'),
          onSubmit: (draft) async {
            submitted = draft;
            return true;
          },
        ),
      ),
    );

    expect(tester.takeException(), isNull);
    expect(find.text('Other: Independent mode'), findsOneWidget);
    await tester.scrollUntilVisible(
      find.byKey(const ValueKey('event-mode')),
      250,
      scrollable: find.byType(Scrollable).last,
    );
    await tester.tap(find.byKey(const ValueKey('event-mode')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Hybrid').last);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();

    expect(submitted?.mode, 'hybrid');
  });

  testWidgets('AT-007 unknown role is visible, removable, and correctable', (
    tester,
  ) async {
    BusinessEventDraft? submitted;
    await tester.pumpWidget(
      _app(
        EventEditorDialog(
          event: _event(roles: const ['independent-role']),
          onSubmit: (draft) async {
            submitted = draft;
            return true;
          },
        ),
      ),
    );

    expect(tester.takeException(), isNull);
    expect(find.text('Other: Independent role'), findsOneWidget);
    final input = find.byKey(const Key('event-roles-search-input'));
    await Scrollable.ensureVisible(tester.element(input), alignment: 0.5);
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(const Key('event-roles-remove-independent-role')),
    );
    await tester.pumpAndSettle();
    await tester.enterText(input, 'Vendor');
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('event-roles-option-vendor')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();

    expect(submitted?.roles, ['vendor']);
  });
}

BusinessEventDraft _draft({
  BusinessImageDraft? image,
  DateTime? startsAt,
  DateTime? endsAt,
  bool isAllDay = false,
}) => BusinessEventDraft(
  name: 'Fibre fair',
  startsAt: startsAt ?? DateTime(2026, 9, 5, 10),
  endsAt: endsAt ?? DateTime(2026, 9, 5, 12),
  roles: const ['vendor'],
  mode: 'in-person',
  status: 'scheduled',
  timeZone: 'Europe/London',
  isAllDay: isAllDay,
  image: image ?? const MissingBusinessImageDraft(),
);

BusinessEvent _event({
  String mode = 'in-person',
  String status = 'scheduled',
  List<String> roles = const ['vendor'],
}) => BusinessEvent(
  did: 'did:plc:owner',
  rkey: '3m4event',
  uri: 'at://did:plc:owner/social.craftsky.business.event/3m4event',
  cid: 'bafy-current',
  name: 'Fibre fair',
  startsAt: DateTime.utc(2026, 9, 5, 10),
  endsAt: DateTime.utc(2026, 9, 5, 12),
  roles: [
    for (final role in roles)
      BusinessOpenValue(value: role, known: businessEventRoles.contains(role)),
  ],
  mode: BusinessOpenValue(value: mode, known: mode == 'in-person'),
  status: BusinessOpenValue(value: status, known: status == 'scheduled'),
  timeZone: 'UTC',
  isAllDay: false,
  createdAt: DateTime.utc(2026, 8, 30),
  past: false,
  publicSuppressionReasons: const [],
  upcomingExclusionReasons: const [],
);

Widget _app(Widget child, {ErrorReporter? reporter}) => ProviderScope(
  overrides: [
    secureSessionRegistryStorageProvider.overrideWithValue(
      InMemorySessionRegistryStorage(),
    ),
    if (reporter != null) errorReporterProvider.overrideWithValue(reporter),
    businessTimeZoneServiceProvider.overrideWithValue(
      BusinessTimeZoneService.initialized(),
    ),
  ],
  child: MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: appLocalizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: child,
  ),
);

PreparedProfileImage _preparedImage() {
  final bytes = Uint8List.fromList(
    img.encodePng(img.Image(width: 1, height: 1, numChannels: 4)),
  );
  return PreparedProfileImage(
    bytes: bytes,
    mimeType: 'image/png',
    width: 1,
    height: 1,
  );
}

UploadedImageBlob _uploadedImage(String cid) => UploadedImageBlob(
  blob: UploadedBlob(
    type: 'blob',
    ref: UploadedBlobRef(link: cid),
    mimeType: 'image/png',
    size: 68,
  ),
  cid: cid,
  mime: 'image/png',
  size: 68,
);

final class _ImageReporter implements ErrorReporter {
  final errors = <Object>[];
  final stacks = <StackTrace?>[];
  final contexts = <ReportContext>[];
  @override
  bool get enabled => true;
  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    stacks.add(stackTrace);
    contexts.add(context);
    return null;
  }
}
