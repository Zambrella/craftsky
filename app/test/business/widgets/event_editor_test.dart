import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';
import 'dart:ui' show Tristate;

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/pages/event_editor_dialog.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../accessibility_test_helpers.dart';

void main() {
  for (final constraint in businessAccessibilityMatrix) {
    testWidgets(
      'AT-012 REG-010 event editor fits '
      '${businessConstraintLabel(constraint)}',
      (tester) async {
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
        expect(
          find.widgetWithText(ChunkyButton, 'Create event'),
          findsOneWidget,
        );
        expect(
          tester
              .getSemantics(find.widgetWithText(FilterChip, 'Vendor'))
              .flagsCollection
              .isSelected,
          Tristate.isTrue,
        );
        await expectKeyboardFocus(tester);
        await tester.scrollUntilVisible(
          find.byKey(const ValueKey('event-registration-uri')),
          250,
          scrollable: find.byType(Scrollable).last,
        );
        expect(
          tester
              .getSemantics(
                find.byKey(const ValueKey('event-registration-uri')),
              )
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
        await tester.pumpAndSettle();
        semantics.dispose();
      },
    );
  }

  testWidgets('AT-007 requires event name, roles, and valid boundaries', (
    tester,
  ) async {
    await tester.pumpWidget(_app(const EventEditorDialog()));

    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();

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
    await tester.pumpAndSettle();

    expect(find.text('End must be after start.'), findsOneWidget);
  });

  testWidgets('event boundaries use date and time pickers', (tester) async {
    await tester.pumpWidget(
      _app(EventEditorDialog(initialDraft: _draft())),
    );

    expect(
      tester.getSemantics(find.byKey(const ValueKey('event-start'))).label,
      contains('Start'),
    );
    expect(
      tester.getSemantics(find.byKey(const ValueKey('event-end'))).label,
      contains('End'),
    );
    expect(find.text('All-day event'), findsNothing);

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

  testWidgets('hidden all-day value is preserved when saving', (tester) async {
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

    expect(find.text('All-day event'), findsNothing);
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();

    expect(submitted?.isAllDay, isTrue);
  });

  testWidgets(
    'AT-007 authors complete timed event on compact and wide layouts',
    (
      tester,
    ) async {
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
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();
    expect(submitted?.image.alt, 'Updated event description');

    final remove = find.byKey(const Key('event-remove-image'));
    await tester.scrollUntilVisible(
      remove,
      250,
      scrollable: find.byType(Scrollable).last,
    );
    await tester.tap(remove);
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();
    expect(submitted?.image, isA<RemovedBusinessImageDraft>());
  });

  testWidgets('event image uploads only when the event is submitted', (
    tester,
  ) async {
    final prepared = _preparedImage();
    var uploadCount = 0;
    BusinessEventDraft? submitted;
    await tester.pumpWidget(
      _app(
        EventEditorDialog(
          initialDraft: _draft(),
          pickImage: (onPreviewReady) async {
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

    expect(uploadCount, 0);
    expect(find.byKey(const Key('event-preview-image')), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();

    expect(uploadCount, 1);
    final submittedImage = submitted!.image as UploadedBusinessImageDraft;
    expect(submittedImage.cid, 'bafy-event-image');
    expect(submittedImage.aspectRatio?.width, 1);
    expect(submittedImage.aspectRatio?.height, 1);
  });

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
          pickImage: (_) async => prepared,
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
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();

    expect(uploadCount, 1);
    expect(submitCount, 0);
    expect(find.byKey(const Key('event-preview-image')), findsOneWidget);
    expect(find.text('The image could not be uploaded. Try again.'), findsOne);

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
          pickImage: (_) async => prepared,
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

    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();
    expect(uploadCount, 1);
    expect(submitCount, 1);

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
    await tester.scrollUntilVisible(
      find.text('Other: Independent role'),
      250,
      scrollable: find.byType(Scrollable).last,
    );
    await tester.tap(find.text('Other: Independent role'));
    await tester.pump();
    await tester.scrollUntilVisible(
      find.text('Vendor'),
      250,
      scrollable: find.byType(Scrollable).last,
    );
    await tester.tap(find.text('Vendor'));
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

Widget _app(Widget child) => ProviderScope(
  overrides: [
    businessTimeZoneServiceProvider.overrideWithValue(
      BusinessTimeZoneService.initialized(),
    ),
  ],
  child: MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: child,
  ),
);

PreparedProfileImage _preparedImage() {
  final bytes = Uint8List.fromList(
    base64Decode(
      'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwC'
      'AAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII=',
    ),
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
