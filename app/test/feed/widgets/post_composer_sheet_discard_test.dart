import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  group('PostComposerSheet discard confirmation', () {
    testWidgets('IT-012 shared account guard cancels or closes dirty compose', (
      tester,
    ) async {
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'alice-token',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      await _openComposer(tester, registry: registry);
      await tester.enterText(find.byType(TextField).first, 'A cardigan WIP');
      await tester.pump();
      final container = ProviderScope.containerOf(
        tester.element(find.byType(PostComposerSheet)),
      );
      final owner = registry.activeLease!.session;

      final cancelled = container
          .read(unsavedWorkGuardProvider)
          .confirmLeave(owner);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Keep editing'));
      await tester.pumpAndSettle();
      expect(await cancelled, isFalse);
      expect(find.text('New post'), findsOneWidget);

      final confirmed = container
          .read(unsavedWorkGuardProvider)
          .confirmLeave(owner);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Discard'));
      await tester.pumpAndSettle();
      expect(await confirmed, isTrue);
      expect(find.text('Host'), findsOneWidget);
    });

    testWidgets('closes immediately when the composer is unchanged', (
      tester,
    ) async {
      await _openComposer(tester);

      await tester.tap(find.byType(CloseButton));
      await tester.pumpAndSettle();

      expect(find.text('Host'), findsOneWidget);
      expect(find.text('Discard draft?'), findsNothing);
    });

    testWidgets('close button confirms before discarding edits', (
      tester,
    ) async {
      await _openComposer(tester);
      await tester.enterText(find.byType(TextField).first, 'A cardigan WIP');
      await tester.pump();

      await tester.tap(find.byType(CloseButton));
      await tester.pumpAndSettle();

      expect(find.text('Discard draft?'), findsOneWidget);
      expect(find.text("Your draft won't be saved."), findsOneWidget);

      await tester.tap(find.text('Keep editing'));
      await tester.pumpAndSettle();

      expect(find.text('Discard draft?'), findsNothing);
      expect(find.text('New post'), findsOneWidget);

      await tester.tap(find.byType(CloseButton));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Discard'));
      await tester.pumpAndSettle();

      expect(find.text('Host'), findsOneWidget);
    });

    testWidgets('system back confirms before discarding edits', (tester) async {
      await _openComposer(tester);
      await tester.enterText(find.byType(TextField).first, 'A cardigan WIP');
      await tester.pump();

      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();

      expect(find.text('Discard draft?'), findsOneWidget);

      await tester.tap(find.text('Discard'));
      await tester.pumpAndSettle();

      expect(find.text('Host'), findsOneWidget);
    });
  });

  group('PostComposerSheet alt text warning', () {
    testWidgets('confirms before submitting images without alt text', (
      tester,
    ) async {
      var createCalls = 0;
      List<CreatePostImage>? submittedImages;
      final repo = FakePostRepository(
        onCreate: ({required text, reply, images}) async {
          createCalls += 1;
          submittedImages = images;
          return _post(text);
        },
      );

      await _openComposer(
        tester,
        composerId: 'composer',
        overrides: [
          composerImagesProvider('composer').overrideWithValue(
            const ComposerImagesState(
              images: [
                ComposerImageDraft(
                  id: 'image-1',
                  fileName: 'project.jpg',
                  mimeType: 'image/jpeg',
                  altText: '',
                  phase: ImageUploaded(
                    UploadedDraftImage(
                      cid: 'bafkimage',
                      mime: 'image/jpeg',
                      size: 123,
                    ),
                  ),
                ),
              ],
            ),
          ),
          postRepositoryProvider.overrideWithValue(repo),
        ],
      );
      await tester.enterText(find.byType(TextField).first, 'A cardigan WIP');
      await _pumpUntilPostEnabled(tester);

      await tester.tap(find.widgetWithText(TextButton, 'Post'));
      await tester.pumpAndSettle();

      expect(find.text('Some images do not have alt text'), findsOneWidget);
      expect(find.text('Do you wish to post anyway?'), findsOneWidget);
      expect(createCalls, 0);

      await tester.tap(find.text('Post anyway'));
      await tester.pumpAndSettle();

      expect(createCalls, 1);
      expect(submittedImages, hasLength(1));
      expect(submittedImages!.single.alt, isEmpty);
    });
  });
}

Future<void> _openComposer(
  WidgetTester tester, {
  List<dynamic> overrides = const [],
  String? composerId,
  SessionRegistry? registry,
}) async {
  final providerOverrides = List<dynamic>.from(overrides);
  if (registry != null) {
    providerOverrides.add(
      secureSessionRegistryStorageProvider.overrideWithValue(
        _RegistryStorage(registry),
      ),
    );
  }
  await tester.pumpWidget(
    ProviderScope(
      overrides: List.from(providerOverrides),
      child: MessengerScope(
        messenger: RecordingMessenger(),
        child: MaterialApp(
          theme: _testTheme,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) {
              return Scaffold(
                body: Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Text('Host'),
                      ElevatedButton(
                        onPressed: () {
                          unawaited(
                            Navigator.of(context).push<Post?>(
                              MaterialPageRoute<Post?>(
                                fullscreenDialog: true,
                                builder: (_) => PostComposerSheet(
                                  composerId: composerId,
                                ),
                              ),
                            ),
                          );
                        },
                        child: const Text('Open composer'),
                      ),
                    ],
                  ),
                ),
              );
            },
          ),
        ),
      ),
    ),
  );

  if (registry != null) {
    final container = ProviderScope.containerOf(
      tester.element(find.byType(MaterialApp)),
    );
    await container.read(sessionRegistryProvider.future);
  }

  await tester.tap(find.text('Open composer'));
  await tester.pumpAndSettle();
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.registry);

  SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async {
    this.registry = registry;
  }
}

Future<void> _pumpUntilPostEnabled(WidgetTester tester) async {
  for (var i = 0; i < 200; i += 1) {
    await tester.pump(const Duration(milliseconds: 20));
    final buttons = find.widgetWithText(TextButton, 'Post').evaluate();
    if (buttons.isEmpty) continue;
    final button = tester.widget<TextButton>(
      find.widgetWithText(TextButton, 'Post'),
    );
    if (button.onPressed != null) return;
  }
  fail('Timed out waiting for Post button to be enabled');
}

Post _post(String text) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
    cid: 'bafy123',
    rkey: '3lf2abc',
    text: text,
    tags: const [],
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    createdAt: DateTime(2026, 5, 22, 12),
    indexedAt: DateTime(2026, 5, 22, 12, 1),
    author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
  );
}

final ThemeData _testTheme =
    ThemeData.from(
      colorScheme: const ColorScheme.light(
        primary: BrandColors.cobalt,
        onSurface: BrandColors.ink,
        onSurfaceVariant: BrandColors.ink2,
        outline: BrandColors.ink3,
        outlineVariant: BrandColors.ink4,
        error: BrandColors.red,
      ),
    ).copyWith(
      scaffoldBackgroundColor: BrandColors.paper,
      extensions: const [
        SpacingTheme(),
        RadiusTheme(),
        DurationTheme(),
        BrandShadowTheme(),
        BrandSwatchTheme(),
        SemanticColorsTheme(
          error: BrandColors.red,
          warning: BrandColors.butter,
          success: BrandColors.moss,
          info: BrandColors.cobalt,
          errorSurface: BrandColors.redSoft,
          warningSurface: BrandColors.butter,
          successSurface: BrandColors.moss,
          infoSurface: BrandColors.cobaltSoft,
        ),
      ],
    );
