import 'dart:async';
import 'dart:ui' show Tristate;

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/settings/pages/profile_customisation_page.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/image_cache_fakes.dart';
import '../fakes/recording_messenger.dart';
import '../profile/fakes/fake_profile_repository.dart';

final _defaultProfile = Profile(
  did: 'did:plc:alice',
  handle: 'alice.example',
  crafts: const [],
);

FakeProfileRepository _repository({
  Future<Profile> Function()? onFetchMe,
  Future<ProfileCustomisation> Function(ProfileCustomisation)? onSave,
}) => FakeProfileRepository(
  onFetchMe: onFetchMe ?? () async => _defaultProfile,
  onUpdateCustomisation: onSave ?? (value) async => value,
);

Widget _app({
  required FakeProfileRepository repository,
  required Widget home,
  ThemeData? theme,
  TextScaler textScaler = TextScaler.noScaling,
}) => ProviderScope(
  overrides: [profileRepositoryProvider.overrideWithValue(repository)],
  child: MessengerScope(
    messenger: RecordingMessenger(),
    child: MaterialApp(
      theme: theme ?? AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      builder: (context, child) => MediaQuery(
        data: MediaQuery.of(context).copyWith(textScaler: textScaler),
        child: child!,
      ),
      home: home,
    ),
  ),
);

Future<void> _pumpRoutedPage(
  WidgetTester tester, {
  FakeProfileRepository? repository,
}) async {
  await tester.pumpWidget(
    _app(
      repository: repository ?? _repository(),
      home: Builder(
        builder: (context) => Scaffold(
          body: TextButton(
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute<void>(
                builder: (_) => const ProfileCustomisationPage(),
              ),
            ),
            child: const Text('Open customisation'),
          ),
        ),
      ),
    ),
  );
  await tester.tap(find.text('Open customisation'));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('edits a live local preview and saves all three choices', (
    tester,
  ) async {
    ProfileCustomisation? submitted;
    final messenger = RecordingMessenger();
    final imageCache = FakeBaseCacheManager();
    final repository = FakeProfileRepository(
      onFetchMe: () async => Profile(
        did: 'did:plc:alice',
        handle: 'alice.example',
        displayName: 'Alice',
        avatar: 'https://example.test/alice.jpg',
        crafts: [],
      ),
      onUpdateCustomisation: (value) async {
        submitted = value;
        return value;
      },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repository),
          profileImageCacheManagerProvider.overrideWith((ref) => imageCache),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfileCustomisationPage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Colour'), findsOneWidget);
    expect(find.text('Profile border'), findsOneWidget);
    expect(find.text('Profile background'), findsOneWidget);
    expect(find.text('Green'), findsOneWidget);
    expect(find.text('Lime'), findsNothing);
    expect(
      tester
          .widget<CachedNetworkImage>(find.byType(CachedNetworkImage))
          .imageUrl,
      'https://example.test/alice.jpg',
    );
    expect(
      tester.widget<ProfileAvatar>(find.byType(ProfileAvatar)).showShadow,
      isFalse,
    );
    await tester.tap(find.text('Teal'));
    await tester.tap(find.text('Thick'));
    await tester.tap(find.text('Crosshatch'));
    await tester.pump();

    final avatar = tester.widget<Container>(
      find.byKey(const Key('profile-avatar-border')),
    );
    final border = (avatar.decoration! as BoxDecoration).border! as Border;
    expect(border.top.color, const Color(0xFF007663));
    expect(border.top.width, 8);
    expect(
      find.byKey(const Key('profile-header-background-texture')),
      findsOneWidget,
    );

    await tester.scrollUntilVisible(
      find.text('Save'),
      300,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(
      submitted,
      const ProfileCustomisation(
        colour: 'teal',
        border: 'thick',
        background: 'x2',
      ),
    );
    expect(messenger.calls.last.$2, 'Profile customisation saved');
  });

  testWidgets('retains the draft and shows exact feedback when save fails', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    final repository = FakeProfileRepository(
      onFetchMe: () async => Profile(
        did: 'did:plc:alice',
        handle: 'alice.example',
        crafts: [],
      ),
      onUpdateCustomisation: (_) async => throw StateError('save failed'),
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repository),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfileCustomisationPage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Rose'));
    await tester.scrollUntilVisible(
      find.text('Save'),
      300,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(ChoiceChip, 'Rose'), findsOneWidget);
    expect(
      messenger.calls.last.$2,
      "Couldn't save your profile customisation.",
    );
    expect(find.text('Save'), findsOneWidget);
  });

  testWidgets('IR-005 clean, reverted, and saved pages pop without a prompt', (
    tester,
  ) async {
    await _pumpRoutedPage(tester);
    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.text('Open customisation'), findsOneWidget);
    expect(find.text('Discard customisation changes?'), findsNothing);

    await tester.tap(find.text('Open customisation'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Rose'));
    await tester.tap(find.text('Cobalt'));
    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.text('Open customisation'), findsOneWidget);
    expect(find.text('Discard customisation changes?'), findsNothing);

    await tester.tap(find.text('Open customisation'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Rose'));
    await tester.scrollUntilVisible(
      find.text('Save'),
      300,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.text('Open customisation'), findsOneWidget);
    expect(find.text('Discard customisation changes?'), findsNothing);
  });

  testWidgets('IR-005 dirty Back uses the branded discard confirmation', (
    tester,
  ) async {
    await _pumpRoutedPage(tester);
    await tester.tap(find.text('Rose'));
    await tester.pump();

    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.text('Discard customisation changes?'), findsOneWidget);
    expect(
      find.text("Your customisation changes won't be saved."),
      findsOneWidget,
    );
    await tester.tap(find.text('Keep editing'));
    await tester.pumpAndSettle();
    expect(find.text('Colour'), findsOneWidget);

    await tester.pageBack();
    await tester.pumpAndSettle();
    await tester.tap(find.text('Discard'));
    await tester.pumpAndSettle();
    expect(find.text('Open customisation'), findsOneWidget);
  });

  testWidgets('IR-005 pending Save suppresses duplicate widget activation', (
    tester,
  ) async {
    final save = Completer<ProfileCustomisation>();
    var calls = 0;
    await tester.pumpWidget(
      _app(
        repository: _repository(
          onSave: (value) {
            calls++;
            return save.future;
          },
        ),
        home: const ProfileCustomisationPage(),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Teal'));
    await tester.scrollUntilVisible(
      find.text('Save'),
      300,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(find.text('Save'));
    await tester.tap(find.text('Save'));
    await tester.pump();

    expect(calls, 1);
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    save.complete(const ProfileCustomisation(colour: 'teal'));
    await tester.pumpAndSettle();
  });

  testWidgets('IR-005 initial load error retries into the editor', (
    tester,
  ) async {
    var fetches = 0;
    await tester.pumpWidget(
      _app(
        repository: _repository(
          onFetchMe: () async {
            fetches++;
            if (fetches == 1) throw StateError('load failed');
            return _defaultProfile;
          },
        ),
        home: const ProfileCustomisationPage(),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.text("Couldn't load your profile customisation."),
      findsOneWidget,
    );
    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();
    expect(find.text('Colour'), findsOneWidget);
    expect(fetches, 2);
  });

  testWidgets('IR-005 choices expose selection and ordered focus structure', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await tester.pumpWidget(
      _app(
        repository: _repository(),
        home: const ProfileCustomisationPage(),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      tester
          .getSemantics(find.widgetWithText(ChoiceChip, 'Cobalt'))
          .flagsCollection
          .isSelected,
      Tristate.isTrue,
    );
    expect(
      tester
          .getSemantics(find.widgetWithText(ChoiceChip, 'Orchid'))
          .flagsCollection
          .isSelected,
      Tristate.isFalse,
    );
    expect(
      tester
          .widgetList<FocusTraversalOrder>(find.byType(FocusTraversalOrder))
          .map((widget) => (widget.order as NumericFocusOrder).order),
      orderedEquals(<double>[
        10,
        11,
        12,
        13,
        14,
        15,
        20,
        21,
        22,
        30,
        31,
        32,
        33,
        34,
        35,
        36,
      ]),
    );
    await tester.scrollUntilVisible(
      find.text('Save'),
      300,
      scrollable: find.byType(Scrollable).first,
    );
    final saveOrder = tester.widget<FocusTraversalOrder>(
      find.ancestor(
        of: find.widgetWithText(FilledButton, 'Save'),
        matching: find.byType(FocusTraversalOrder),
      ),
    );
    expect((saveOrder.order as NumericFocusOrder).order, 40);
    semantics.dispose();
  });

  for (final themeName in <String>['light', 'dark']) {
    testWidgets('IR-005 supports large text in the $themeName theme', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 640);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(
        _app(
          repository: _repository(),
          theme: themeName == 'light'
              ? AppTheme.lightThemeData
              : AppTheme.darkThemeData,
          textScaler: const TextScaler.linear(2),
          home: const ProfileCustomisationPage(),
        ),
      );
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.text('Save'),
        300,
        scrollable: find.byType(Scrollable).first,
      );

      expect(find.text('Save'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }
}
