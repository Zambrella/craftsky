import 'dart:async';
import 'dart:convert';
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
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:logging/logging.dart';
import 'package:material_ui/material_ui.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../../fakes/recording_messenger.dart';
import '../../test_support/serialized_sentry_transport.dart';
import '../../test_support/session_registry_storage.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  for (final scenario in _ExternalScenario.values) {
    final refresh =
        scenario == _ExternalScenario.refresh ||
        scenario == _ExternalScenario.refreshThumbnail ||
        scenario == _ExternalScenario.refreshFailure;
    final thumbnail = scenario == _ExternalScenario.refreshThumbnail;
    final newCard =
        scenario == _ExternalScenario.add ||
        scenario == _ExternalScenario.unchanged;
    final remove = scenario == _ExternalScenario.remove;
    final unchanged = scenario == _ExternalScenario.unchanged;
    testWidgets(
      'AT-002 external metadata is prefilled and clears only changed leaves '
      '(scenario=${scenario.name})',
      (tester) async {
        final registry = SessionRegistry.empty().upsertAndActivate(
          token: 'alice-token',
          did: 'did:plc:alice',
          handle: 'alice.example',
        );
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
              : const PostExternal(
                  uri: 'https://example.com/old',
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
                  'uri': 'https://example.com/old',
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
        final failedRefresh = scenario == _ExternalScenario.refreshFailure;
        final reporter = _ExternalReporter();
        final transport = SerializedTransport();
        final lines = <String>[];
        if (failedRefresh) {
          await Sentry.init((options) {
            options
              ..dsn = 'https://public@example.invalid/1'
              ..transport = transport;
            configureDiagnosticOptions(options, enableLogs: false);
          });
          addTearDown(Sentry.close);
          final previous = Logger.root.level;
          Logger.root.level = Level.ALL;
          final emitter = DiagnosticEmitter(platformSink: lines.add);
          final subscription = Logger.root.onRecord.listen(emitter.emitLocal);
          addTearDown(subscription.cancel);
          addTearDown(() => Logger.root.level = previous);
        }
        final failure = StateError(
          'PRIVATE_PREVIEW_CANARY https://private.invalid/?token=PRIVATE_CAPABILITY_CANARY',
        );
        final stack = StackTrace.fromString(
          '#0 refreshPreview (package:craftsky_app/feed/preview.dart:12:3)',
        );
        final previews = _ExternalPreviewRepository(
          thumbnail: thumbnail,
          failure: failedRefresh ? failure : null,
          stack: stack,
        );
        final uploads = _ExternalThumbnailClient();
        Map<String, Object?>? submitted;
        final completed = Completer<Post?>();
        await tester.pumpWidget(
          ProviderScope(
            retry: (_, _) => null,
            overrides: [
              linkPreviewRepositoryProvider.overrideWithValue(previews),
              if (failedRefresh)
                errorReporterProvider.overrideWithValue(reporter),
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
                          projectionPending: false,
                          editedAt: DateTime.utc(2026),
                          editTimeSource: PostEditTimeSource.confirmed,
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
        if (!newCard) {
          expect(find.byType(ComposerImageAttachmentSection), findsNothing);
        }
        if (newCard && !unchanged) {
          await tester.enterText(
            find.byType(TextField).first,
            'A useful pattern https://example.com/new ',
          );
          await tester.pumpAndSettle(const Duration(milliseconds: 500));
          expect(find.text('Fetched title'), findsOneWidget);
        } else if (unchanged) {
          // Saving untouched caption URLs must not attach a new card.
        } else if (remove) {
          final dismiss = find.byKey(const Key('published-external-remove'));
          await tester.ensureVisible(dismiss);
          await tester.pumpAndSettle();
          await tester.tap(dismiss);
          await tester.pumpAndSettle();
          expect(find.byKey(const Key('link-preview-carousel')), findsNothing);
        } else {
          final uri = find.byKey(const Key('published-external-uri'));
          final title = find.byKey(const Key('published-external-title'));
          final description = find.byKey(
            const Key('published-external-description'),
          );
          expect(
            tester.widget<TextField>(uri).controller?.text,
            'https://example.com/old',
          );
          expect(tester.widget<TextField>(title).controller?.text, 'Old title');
          expect(
            tester.widget<TextField>(description).controller?.text,
            'Old description',
          );
          await tester.ensureVisible(uri);
          await tester.enterText(uri, 'https://example.com/new');
          if (refresh) {
            FocusManager.instance.primaryFocus?.unfocus();
            await tester.pumpAndSettle();
            final refreshButton = find.byKey(
              const Key('published-external-refresh'),
            );
            await tester.ensureVisible(refreshButton);
            await tester.pumpAndSettle();
            await tester.tap(refreshButton);
            await tester.pumpAndSettle();
            if (failedRefresh) {
              await Future.wait(reporter.pending);
              expect(reporter.errors, [same(failure)]);
              expect(reporter.stacks, [same(stack)]);
              expect(
                tester.widget<TextField>(title).controller?.text,
                'Old title',
              );
              expect(submitted, isNull);
              expect(uploads.calls, 0);
              final events = transport.payloads
                  .map(jsonDecode)
                  .whereType<Map<String, dynamic>>()
                  .where((event) => event.containsKey('exception'))
                  .toList();
              expect(events, hasLength(1));
              final serialized = [...lines, ...transport.payloads].join('\n');
              for (final positive in [
                'StateError',
                'preview.dart',
                'preview_refresh',
                post.uri.value,
                'did:plc:alice',
              ]) {
                expect(serialized, contains(positive));
              }
              for (final canary in [
                'PRIVATE_PREVIEW_CANARY',
                'PRIVATE_CAPABILITY_CANARY',
                'https://example.com/new',
              ]) {
                expect(serialized, isNot(contains(canary)));
              }
              await tester.runAsync(Sentry.close);
              return;
            }
            expect(
              tester.widget<TextField>(title).controller?.text,
              'Fetched title',
            );
            expect(
              tester.widget<TextField>(description).controller?.text,
              'Fetched description',
            );
          } else {
            await tester.ensureVisible(title);
            await tester.enterText(title, 'New title');
            await tester.ensureVisible(description);
            await tester.enterText(description, '');
            FocusManager.instance.primaryFocus?.unfocus();
            await tester.pumpAndSettle();
            await tester.ensureVisible(
              find.byKey(const Key('published-external-remove-thumbnail')),
            );
            await tester.pumpAndSettle();
            await tester.tap(
              find.byKey(const Key('published-external-remove-thumbnail')),
            );
          }
        }
        FocusManager.instance.primaryFocus?.unfocus();
        await tester.pumpAndSettle();
        await tester.tap(find.widgetWithText(ChunkyButton, 'Save changes'));
        await tester.pumpAndSettle();
        expect(
          submitted,
          unchanged
              ? <String, Object?>{}
              : remove
              ? {'embed': null}
              : {
                  if (newCard)
                    'text': 'A useful pattern https://example.com/new',
                  if (newCard) 'facets': anything,
                  'embed': {
                    if (newCard) r'$type': 'app.bsky.embed.external',
                    'external': {
                      'uri': 'https://example.com/new',
                      'title': refresh || newCard
                          ? 'Fetched title'
                          : 'New title',
                      'description': refresh || newCard
                          ? 'Fetched description'
                          : '',
                      if (!newCard || thumbnail)
                        'thumb': thumbnail
                            ? {
                                r'$type': 'blob',
                                'ref': {r'$link': 'bafynewthumb'},
                                'mimeType': 'image/png',
                                'size': previews.bytes.length,
                              }
                            : null,
                    },
                  },
                },
        );
        if (!remove) {
          expect(previews.calls, !unchanged && (refresh || newCard) ? 1 : 0);
        }
        expect(uploads.calls, thumbnail ? 1 : 0);
        expect(completed.isCompleted, isTrue);
      },
    );
  }
}

final class _ExternalPreviewRepository implements LinkPreviewRepository {
  new({required this.thumbnail, this.failure, this.stack});
  final bool thumbnail;
  final Object? failure;
  final StackTrace? stack;
  final bytes = Uint8List.fromList(
    img.encodePng(img.Image(width: 1, height: 1)),
  );
  int calls = 0;
  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) async {
    calls++;
    if (failure != null) Error.throwWithStackTrace(failure!, stack!);
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
  metadata,
  refresh,
  refreshThumbnail,
  add,
  unchanged,
  remove,
  refreshFailure,
}

final class _ExternalReporter implements ErrorReporter {
  final errors = <Object>[];
  final stacks = <StackTrace?>[];
  final pending = <Future<String?>>[];
  @override
  bool get enabled => true;
  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) {
    errors.add(error);
    stacks.add(stackTrace);
    final future = const SentryErrorReporter().captureException(
      error,
      context: context,
      stackTrace: stackTrace,
    );
    pending.add(future);
    return future;
  }
}
