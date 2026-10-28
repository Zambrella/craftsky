import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/composer/post_edit_origin.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:material_ui/material_ui.dart';

import '../../fakes/recording_messenger.dart';
import '../../test_support/session_registry_storage.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  for (final scenario in _ExternalScenario.values) {
    final thumbnail = scenario == _ExternalScenario.replaceThumbnail;
    final replace = scenario == _ExternalScenario.replace || thumbnail;
    final newCard =
        scenario == _ExternalScenario.add ||
        scenario == _ExternalScenario.unchanged;
    final remove = scenario == _ExternalScenario.remove;
    final unchanged =
        scenario == _ExternalScenario.unchanged ||
        scenario == _ExternalScenario.retain ||
        scenario == _ExternalScenario.retainAtUri ||
        scenario == _ExternalScenario.retainAtUriCaption;
    testWidgets(
      'AT-002 link previews use normal retention and replacement controls '
      '(scenario=${scenario.name})',
      (tester) async {
        final registry = SessionRegistry.empty().upsertAndActivate(
          token: 'alice-token',
          did: 'did:plc:alice',
          handle: 'alice.example',
        );
        final retainedUri =
            scenario == _ExternalScenario.retainAtUri ||
                scenario == _ExternalScenario.retainAtUriCaption
            ? 'at://did:plc:other/social.craftsky.feed.post/target'
            : 'https://example.com/old';
        final post = Post(
          uri: 'at://did:plc:alice/social.craftsky.feed.post/external',
          cid: 'bafybase',
          rkey: 'external',
          text: remove || unchanged
              ? 'A useful pattern https://example.com/old '
              : 'A useful pattern',
          tags: const [],
          createdAt: DateTime.utc(2020),
          indexedAt: DateTime.utc(2020),
          author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
          likeCount: 0,
          repostCount: 0,
          replyCount: 0,
          viewerHasLiked: false,
          viewerHasReposted: false,
          viewerHasSaved: false,
          sponsored: false,
          external: newCard
              ? null
              : PostExternal(
                  uri: retainedUri,
                  title: 'Old title',
                  description: 'Old description',
                ),
        );
        final seed = PostEditSeed(
          post: post,
          baseCid: post.cid,
          recordRevision: '3aaaaaaaaaaa2',
          editable: {
            'text': post.text,
            'sponsored': false,
            'langs': ['en'],
            if (!newCard)
              'embed': {
                r'$type': 'app.bsky.embed.external',
                'external': {
                  'uri': retainedUri,
                  'title': 'Old title',
                  'description': 'Old description',
                  'thumb': {
                    r'$type': 'blob',
                    'ref': {r'$link': 'bafythumb'},
                    'mimeType': 'image/jpeg',
                    'size': 100,
                  },
                },
              },
          },
        );
        final previews = _ExternalPreviewRepository(thumbnail: thumbnail);
        final uploads = _ExternalThumbnailClient();
        Map<String, Object?>? submitted;
        final completed = Completer<Post?>();
        await tester.pumpWidget(
          ProviderScope(
            retry: (_, _) => null,
            overrides: [
              linkPreviewRepositoryProvider.overrideWithValue(previews),
              postApiClientProvider.overrideWithValue(uploads),
              secureSessionRegistryStorageProvider.overrideWithValue(
                InMemorySessionRegistryStorage(registry),
              ),
              activeLanguagePreferencesProvider.overrideWith(
                (ref) => const LanguagePreferences(
                  primaryLanguage: 'en',
                  contentLanguages: ['en'],
                ),
              ),
              postRepositoryProvider.overrideWithValue(
                FakePostRepository(
                  onEdit:
                      (
                        did,
                        key, {
                        required changes,
                        required expectedCid,
                        required operationKey,
                      }) async {
                        submitted = changes.toMap();
                        return PostEditResult(
                          post: post.copyWith(cid: 'bafyaccepted'),
                          acceptedCid: post.copyWith(cid: 'bafyaccepted').cid,
                          changed: true,
                          editedAt: DateTime.utc(2026),
                        );
                      },
                ),
              ),
            ],
            child: MessengerScope(
              messenger: RecordingMessenger(),
              child: MaterialApp(
                theme: AppTheme.lightThemeData,
                localizationsDelegates: appLocalizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                home: Builder(
                  builder: (context) => Scaffold(
                    body: ElevatedButton(
                      onPressed: () async => completed.complete(
                        await showPostComposerSheet(
                          context,
                          publishedEdit: PostEditOrigin(
                            seed: seed,
                            ownership: registry.activeLease!,
                          ),
                        ),
                      ),
                      child: const Text('Open edit'),
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
        await tester.tap(find.text('Open edit'));
        await tester.pumpAndSettle();
        expect(find.byKey(const Key('published-external-title')), findsNothing);
        if (!newCard) {
          expect(
            find.byKey(const Key('link-preview-carousel')),
            findsOneWidget,
          );
          expect(find.text('Old title'), findsOneWidget);
          if (remove || replace) {
            await tester.ensureVisible(find.byTooltip('Dismiss link previews'));
            await tester.tap(find.byTooltip('Dismiss link previews'));
            await tester.pumpAndSettle();
          }
        }
        expect(tester.takeException(), isNull);
        if (scenario == _ExternalScenario.retainAtUriCaption) {
          await tester.enterText(
            find.byType(TextField).first,
            'An updated caption',
          );
          await tester.pumpAndSettle();
        }
        if (replace || (newCard && !unchanged)) {
          await tester.enterText(
            find.byType(TextField).first,
            'A useful pattern https://example.com/new ',
          );
          await tester.pumpAndSettle(const Duration(milliseconds: 500));
          expect(find.text('Fetched title'), findsOneWidget);
        }
        FocusManager.instance.primaryFocus?.unfocus();
        await tester.pumpAndSettle();
        await tester.tap(find.widgetWithText(ChunkyButton, 'Save changes'));
        await tester.pumpAndSettle();
        if (scenario == _ExternalScenario.retainAtUriCaption) {
          expect(submitted!['text'], 'An updated caption');
          expect(submitted!.containsKey('embed'), isFalse);
        } else if (unchanged) {
          expect(submitted, isEmpty);
        } else if (remove) {
          expect(submitted, {'embed': null});
        } else {
          expect(
            submitted!['text'],
            'A useful pattern https://example.com/new',
          );
          expect(submitted!['embed'], {
            r'$type': 'app.bsky.embed.external',
            'external': {
              'uri': 'https://example.com/new',
              'title': 'Fetched title',
              'description': 'Fetched description',
              if (thumbnail)
                'thumb': {
                  r'$type': 'blob',
                  'ref': {r'$link': 'bafynewthumb'},
                  'mimeType': 'image/png',
                  'size': previews.bytes.length,
                },
            },
          });
        }
        expect(previews.calls, replace || (newCard && !unchanged) ? 1 : 0);
        expect(uploads.calls, thumbnail ? 1 : 0);
        expect(completed.isCompleted, isTrue);
      },
    );
  }
}

final class _ExternalPreviewRepository implements LinkPreviewRepository {
  new({required this.thumbnail});
  final bool thumbnail;
  final bytes = Uint8List.fromList(
    img.encodePng(img.Image(width: 1, height: 1)),
  );
  int calls = 0;
  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) async {
    calls++;
    return LinkPreview(
      url: url,
      title: 'Fetched title',
      description: 'Fetched description',
      thumbnail: thumbnail
          ? LinkPreviewThumbnail(
              bytes: bytes,
              mimeType: 'image/png',
              width: 1,
              height: 1,
            )
          : null,
    );
  }
}

final class _ExternalThumbnailClient extends Fake implements PostApiClient {
  int calls = 0;
  @override
  Future<UploadedImageBlob> uploadImage({
    required List<int> bytes,
    required String mimeType,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
    CancelToken? cancelToken,
  }) async {
    calls++;
    return UploadedImageBlob(
      blob: UploadedBlob(
        type: 'blob',
        ref: const UploadedBlobRef(link: 'bafynewthumb'),
        mimeType: mimeType,
        size: bytes.length,
      ),
      cid: 'bafynewthumb',
      mime: mimeType,
      size: bytes.length,
    );
  }
}

enum _ExternalScenario {
  retainAtUri,
  retainAtUriCaption,
  retain,
  replace,
  replaceThumbnail,
  add,
  unchanged,
  remove,
}
