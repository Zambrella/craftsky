import 'dart:async';
import 'dart:ui' show Tristate;

import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/pages/event_editor_dialog.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
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

  testWidgets('AT-007 failed image replacement preserves stored blob', (
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
          pickImage: (_) async => throw Exception('upload failed'),
          onSubmit: (draft) async {
            submitted = draft;
            return true;
          },
        ),
      ),
    );

    await tester.ensureVisible(find.text('Replace image'));
    await tester.tap(find.text('Replace image'));
    await tester.pumpAndSettle();
    expect(find.text('The image could not be uploaded. Try again.'), findsOne);
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();

    expect(submitted?.image.toJson(), image.toJson());
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
    await tester.tap(find.text('Other: Independent role'));
    await tester.tap(find.text('Vendor'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('event-submit')));
    await tester.pumpAndSettle();

    expect(submitted?.roles, ['vendor']);
  });
}

BusinessEventDraft _draft({BusinessImageDraft? image}) => BusinessEventDraft(
  name: 'Fibre fair',
  startsAt: DateTime(2026, 9, 5, 10),
  endsAt: DateTime(2026, 9, 5, 12),
  roles: const ['vendor'],
  mode: 'in-person',
  status: 'scheduled',
  timeZone: 'Europe/London',
  isAllDay: false,
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
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: child,
  ),
);
