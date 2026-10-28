import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/composer/post_edit_origin.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/video_service_client_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/feed/widgets/published_post_editor.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/providers/preferred_project_crafts_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/rich_text/data/mock_facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../../fakes/recording_messenger.dart';
import '../../test_support/session_registry_storage.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  for (final (
        pending,
        lag,
        projectEdit,
        retainedVideo,
        replaceVideo,
        realUpload,
        timeoutRecovery,
      )
      in [
        (false, false, false, false, false, false, false),
        (true, false, false, false, false, false, false),
        (true, true, false, false, false, false, false),
        (false, false, true, false, false, false, false),
        (true, false, true, false, false, false, false),
        (true, true, true, false, false, false, false),
        (false, false, false, true, false, false, false),
        (false, false, true, true, false, false, false),
        (false, false, false, true, true, false, false),
        (false, false, true, true, true, false, false),
        (false, false, false, true, true, true, false),
        (false, false, true, true, true, true, false),
        (true, true, false, true, false, false, false),
        (true, true, true, true, false, false, false),
        (true, true, false, true, true, false, false),
        (true, true, true, true, true, true, false),
        (false, false, false, false, false, false, true),
        (false, false, true, false, false, false, true),
      ]) {
    for (final missingBlob in realUpload ? [false, true] : [false]) {
      testWidgets(
        'AT-001 published post (project=$projectEdit, video=$retainedVideo, '
        'replace=$replaceVideo, realUpload=$realUpload) opens prefilled '
        'and saves through edit '
        '(pending=$pending, lag=$lag, missingBlob=$missingBlob, '
        'timeout=$timeoutRecovery)',
        (tester) async {
          final registry = SessionRegistry.empty().upsertAndActivate(
            token: 'alice-token',
            did: 'did:plc:alice',
            handle: 'alice.example',
          );
          final post = Post(
            uri: 'at://did:plc:alice/social.craftsky.feed.post/base',
            cid: 'bafybase',
            rkey: 'base',
            text: 'Original caption',
            langs: const ['en'],
            tags: const [],
            createdAt: DateTime.utc(2020),
            indexedAt: DateTime.utc(2020),
            author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
            likeCount: 7,
            repostCount: 2,
            replyCount: 1,
            viewerHasLiked: false,
            viewerHasReposted: false,
            viewerHasSaved: true,
            sponsored: false,
            reply: !pending && !projectEdit && !retainedVideo
                ? PostReply(
                    root: PostRef(
                      uri: 'at://did:plc:bob/social.craftsky.feed.post/parent',
                      cid: 'bafyparent',
                    ),
                    parent: PostRef(
                      uri: 'at://did:plc:bob/social.craftsky.feed.post/parent',
                      cid: 'bafyparent',
                    ),
                  )
                : null,
            quote: !pending && projectEdit && !retainedVideo
                ? PostRef(
                    uri: 'at://did:plc:bob/social.craftsky.feed.post/quoted',
                    cid: 'bafyquoted',
                  )
                : null,
            quoteView: !pending && projectEdit && !retainedVideo
                ? QuoteView(
                    state: 'visible',
                    post: QuotePreviewPost(
                      uri: 'at://did:plc:bob/social.craftsky.feed.post/quoted',
                      cid: 'bafyquoted',
                      text: 'Fixed quoted context',
                      author: PostAuthor(
                        did: 'did:plc:bob',
                        handle: 'bob.example',
                      ),
                      createdAt: DateTime.utc(2020),
                      sponsored: false,
                    ),
                  )
                : null,
            project: projectEdit
                ? const Project(
                    common: ProjectCommon(
                      craftType: 'social.craftsky.feed.defs#sewing',
                      title: 'Linen shirt',
                    ),
                  )
                : null,
            video: retainedVideo
                ? PostVideo(
                    cid: 'bafyvideo',
                    mime: 'video/mp4',
                    size: 100,
                    alt: 'Original video alt',
                  )
                : null,
            images: projectEdit && !retainedVideo
                ? [
                    PostImage(
                      cid: 'bafyimage',
                      mime: 'image/jpeg',
                      size: 200,
                      alt: 'Linen shirt',
                      fullsize: 'https://cdn.example/image.jpeg',
                    ),
                  ]
                : null,
          );
          final seed = PostEditSeed(
            post: post,
            baseCid: post.cid,
            recordRevision: '3aaaaaaaaaaa2',
            editable: {
              'text': post.text,
              'sponsored': false,
              'langs': ['en'],
              if (projectEdit && pending && lag)
                'unsupportedProjectDetails': true,
              if (projectEdit)
                'project': {
                  'common': {
                    'craftType': 'social.craftsky.feed.defs#sewing',
                    'title': 'Linen shirt',
                  },
                },
              if (retainedVideo)
                'embed': {
                  r'$type': 'app.bsky.embed.video',
                  'video': {
                    r'$type': 'blob',
                    'ref': {r'$link': 'bafyvideo'},
                    'mimeType': 'video/mp4',
                    'size': 100,
                  },
                  'alt': 'Original video alt',
                },
              if (projectEdit && !retainedVideo)
                'images': [
                  {
                    'existingIndex': 0,
                    'alt': 'Linen shirt',
                    'image': {
                      r'$type': 'blob',
                      'ref': {r'$link': 'bafyimage'},
                      'mimeType': 'image/jpeg',
                      'size': 200,
                    },
                  },
                ],
            },
          );
          final origin = PostEditOrigin(
            seed: seed,
            ownership: registry.activeLease!,
          );
          var edits = 0;
          final editKeys = <String>[];
          var recovered = false;
          var creates = 0;
          var reads = 0;
          final repository = FakePostRepository(
            onFetch: (did, key) async {
              if (did == post.author.did && key == post.rkey && edits > 0) {
                reads++;
                if (lag) return await Completer<Post>().future;
                return post.copyWith(
                  cid: 'bafyaccepted',
                  text: 'Changed caption',
                  recordRevision: '3aaaaaaaaaaa3',
                );
              }
              return post.copyWith(
                uri: 'at://$did/social.craftsky.feed.post/$key',
                rkey: key,
                text: 'Original parent context',
                reply: null,
                quote: null,
                quoteView: null,
              );
            },
            onFetchEdit: (did, key) async => seed,
            onCreate: ({required text, reply, images}) async {
              creates++;
              return post;
            },
            onEdit:
                (
                  did,
                  key, {
                  required changes,
                  required expectedCid,
                  required operationKey,
                }) async {
                  edits++;
                  if (timeoutRecovery && !recovered) {
                    editKeys.add(operationKey);
                    throw const PdsMutationAmbiguousException(
                      retryAfterSeconds: 1,
                    );
                  }
                  editKeys.add(operationKey);
                  expect(find.text('Saving changes…'), findsOneWidget);
                  expect(expectedCid, post.cid);
                  expect(changes.toMap()['text'], 'Changed caption');
                  expect(changes.toMap().containsKey('createdAt'), isFalse);
                  if (projectEdit) {
                    expect(changes.toMap()['project'], {
                      'common': {'title': 'Updated shirt'},
                    });
                  }
                  expect(changes.toMap().containsKey('images'), isFalse);
                  if (retainedVideo) {
                    if (replaceVideo) {
                      final embed = changes.toMap()['embed']! as Map;
                      expect(
                        embed['jobId'],
                        missingBlob && edits > 1 ? 'job-recovered' : 'job-new',
                      );
                      expect((embed['video'] as Map)['ref'], {
                        r'$link': missingBlob && edits > 1
                            ? 'bafyrecoveredvideo'
                            : 'bafynewvideo',
                      });
                      expect(embed['alt'], 'Updated video alt');
                    } else {
                      expect(changes.toMap()['embed'], {
                        'alt': 'Updated video alt',
                      });
                    }
                  }
                  if (missingBlob && edits == 1) {
                    throw const ApiBadRequest(
                      'video_blob_missing',
                      details: ApiFailureDetails(
                        appViewError: 'video_blob_missing',
                        statusCode: 502,
                      ),
                    );
                  }
                  return PostEditResult(
                    acceptedCid: Cid.parse('bafyaccepted'),
                    changed: true,
                    post: pending
                        ? null
                        : post.copyWith(
                            cid: 'bafyaccepted',
                            text: 'Changed caption',
                          ),
                    editedAt: DateTime.utc(2026, 10, 26),
                  );
                },
          );
          final adapter = _CompletedEditVideoAdapter();
          final videoDio = Dio()..httpClientAdapter = adapter;
          addTearDown(() => videoDio.close(force: true));
          var proofs = 0;
          final videos = ComposerVideoController(picker: _NoopEditVideoPicker())
            ..restoredSelection = LocalVideoSelection(
              displayName: 'replacement.mp4',
              mimeType: 'video/mp4',
              byteLength: 12,
              duration: const Duration(seconds: 2),
              headerBytes: Uint8List(0),
              openRead: () => Stream.value(Uint8List(12)),
              altText: 'Original video alt',
            );
          Future<CreatePostVideo> prepareProof(
            LocalVideoSelection selected,
          ) async {
            proofs++;
            return CreatePostVideo(
              jobId: 'job-new',
              blob: const CreatePostVideoBlob(
                cid: 'bafynewvideo',
                mimeType: 'video/mp4',
                size: 12,
              ),
              alt: selected.altText,
            );
          }

          final completed = Completer<Post?>();
          final messenger = RecordingMessenger();
          await tester.pumpWidget(
            ProviderScope(
              retry: (_, _) => null,
              overrides: [
                pdsMutationDelayProvider.overrideWithValue((_) async {}),
                pdsMutationJitterProvider.overrideWithValue((_) => 0),
                secureSessionRegistryStorageProvider.overrideWithValue(
                  InMemorySessionRegistryStorage(registry),
                ),
                postRepositoryProvider.overrideWithValue(repository),
                postApiClientProvider.overrideWithValue(_EditVideoApiClient()),
                videoServiceClientProvider.overrideWithValue(
                  VideoServiceClient.forTesting(
                    uploadEndpoint:
                        VideoServiceConfiguration.bluesky().uploadEndpoint,
                    dio: videoDio,
                  ),
                ),
                preferredProjectCraftsProvider.overrideWithValue(
                  ProjectOptionCatalogs.craftTypesForPreferences(const []),
                ),
                defaultProjectCraftProvider.overrideWith(
                  (ref) => 'social.craftsky.feed.defs#quilting',
                ),
                activeLanguagePreferencesProvider.overrideWith(
                  (ref) => const LanguagePreferences(
                    primaryLanguage: 'fr',
                    contentLanguages: ['fr'],
                  ),
                ),
                accountSuggestionRepositoryProvider.overrideWithValue(
                  const MockAccountSuggestionRepository(accounts: []),
                ),
                hashtagSuggestionRepositoryProvider.overrideWithValue(
                  const MockHashtagSuggestionRepository(hashtags: []),
                ),
              ],
              child: MessengerScope(
                messenger: messenger,
                child: MaterialApp(
                  theme: AppTheme.lightThemeData,
                  localizationsDelegates: appLocalizationsDelegates,
                  supportedLocales: AppLocalizations.supportedLocales,
                  home: Builder(
                    builder: (context) => Scaffold(
                      body: ElevatedButton(
                        onPressed: () async {
                          completed.complete(
                            replaceVideo
                                ? await Navigator.of(context).push<Post>(
                                    MaterialPageRoute(
                                      builder: (_) => projectEdit
                                          ? ProjectComposerSheet(
                                              publishedEdit: origin,
                                              videoController: videos,
                                              prepareVideoProof: realUpload
                                                  ? null
                                                  : prepareProof,
                                            )
                                          : PostComposerSheet(
                                              publishedEdit: origin,
                                              videoController: videos,
                                              prepareVideoProof: realUpload
                                                  ? null
                                                  : prepareProof,
                                            ),
                                    ),
                                  )
                                : await showPublishedPostEditor(
                                    context,
                                    post: post,
                                    ownership: registry.activeLease!,
                                  ),
                          );
                        },
                        child: const Text('Edit fixture'),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          );
          await tester.tap(find.text('Edit fixture'));
          await tester.pumpAndSettle();
          final field = projectEdit
              ? find.descendant(
                  of: find.byKey(const Key('project-composer-body-editor')),
                  matching: find.byType(TextField),
                )
              : find.byType(TextField).first;
          expect(
            tester.widget<TextField>(field).controller?.text,
            'Original caption',
          );
          expect(find.text('Save changes'), findsOneWidget);
          if (projectEdit && pending && lag) {
            expect(
              find.byKey(const Key('project-composer-craft-details-action')),
              findsNothing,
            );
          }
          if (!pending && !retainedVideo) {
            expect(
              find.text(
                projectEdit
                    ? 'Fixed quoted context'
                    : 'Original parent context',
              ),
              findsOneWidget,
            );
          }
          expect(
            find.byKey(const Key('composer-schedule-control')),
            findsNothing,
          );
          expect(find.text('Save draft'), findsNothing);
          if (retainedVideo) {
            final alt = find.byKey(const Key('composer-video-alt-text'));
            expect(
              tester.widget<TextField>(alt).controller?.text,
              'Original video alt',
            );
            await tester.ensureVisible(alt);
            await tester.enterText(alt, 'Updated video alt');
          }
          if (projectEdit) {
            final title = find.byKey(const Key('project-title-input'));
            expect(
              tester.widget<TextField>(title).controller?.text,
              'Linen shirt',
            );
            await tester.ensureVisible(title);
            await tester.enterText(title, 'Updated shirt');
          }
          await tester.ensureVisible(field);
          await tester.enterText(field, 'Changed caption');
          FocusManager.instance.primaryFocus?.unfocus();
          await tester.pumpAndSettle();
          await tester.tap(find.widgetWithText(ChunkyButton, 'Save changes'));
          await tester.pumpAndSettle();
          if (timeoutRecovery) {
            expect(edits, 7);
            expect(completed.isCompleted, isFalse);
            expect(
              tester.widget<TextField>(field).controller?.text,
              'Changed caption',
            );
            final frozenField = tester.widget<TextField>(field);
            expect(
              frozenField.readOnly || frozenField.enabled == false,
              isTrue,
            );
            expect(
              messenger.calls.last.$2,
              'Saving is not confirmed. Retry to check the same save.',
            );
            expect(
              find.text(
                'Saving is not confirmed. Retry to check the same save.',
              ),
              findsOneWidget,
            );
            final retry = find.widgetWithText(ChunkyButton, 'Retry save');
            expect(tester.widget<ChunkyButton>(retry).onPressed, isNotNull);
            recovered = true;
            await tester.tap(retry);
            await tester.pumpAndSettle();
          }
          expect(find.text('Refresh post'), findsNothing);
          if (pending) expect(reads, 1);
          expect(
            edits,
            timeoutRecovery
                ? 8
                : missingBlob
                ? 2
                : 1,
          );
          expect(editKeys.toSet(), hasLength(timeoutRecovery ? 1 : edits));
          expect(proofs, replaceVideo && !realUpload ? 1 : 0);
          expect(adapter.calls, realUpload ? (missingBlob ? 2 : 1) : 0);
          expect(creates, 0);
          expect(completed.isCompleted, isTrue);
          final completedPost = await completed.future;
          expect(completedPost?.text, 'Changed caption');
          if (pending) {
            expect(completedPost?.cid.value, 'bafyaccepted');
            expect(completedPost?.createdAt, post.createdAt);
            expect(completedPost?.editedAt, DateTime.utc(2026, 10, 26));
            if (projectEdit) {
              expect(completedPost?.project?.common.title, 'Updated shirt');
            }
            if (retainedVideo) {
              expect(completedPost?.video?.alt, 'Updated video alt');
              expect(
                completedPost?.video?.cid.value,
                replaceVideo
                    ? (missingBlob ? 'bafyrecoveredvideo' : 'bafynewvideo')
                    : 'bafyvideo',
              );
            }
          }
          expect(
            messenger.calls.any((call) => call.$2 == 'Changes saved'),
            isTrue,
          );
        },
      );
    }
  }
}

final class _NoopEditVideoPicker implements ExistingVideoPicker {
  @override
  Future<LocalVideoSelection?> pickExisting() async => null;
}

final class _EditVideoApiClient extends PostApiClient {
  new() : super(Dio());
  @override
  Future<VideoUploadLimits> getVideoUploadLimits() async =>
      const VideoUploadLimits(canUpload: true);
  @override
  Future<VideoUploadAuthorization> authorizeVideoUpload() async =>
      VideoUploadAuthorization.fromMap({
        'token': 'synthetic-video-token',
        'expiresAt': '2030-01-01T00:00:00Z',
      });
}

final class _CompletedEditVideoAdapter implements HttpClientAdapter {
  int calls = 0;
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    calls++;
    expect(options.headers['authorization'], 'Bearer synthetic-video-token');
    final bytes = await requestStream!.expand((chunk) => chunk).toList();
    expect(bytes.length, calls == 1 ? 12 : greaterThan(12));
    return ResponseBody.fromString(
      jsonEncode({
        'jobStatus': {
          'jobId': calls == 1 ? 'job-new' : 'job-recovered',
          'state': 'JOB_STATE_COMPLETED',
          'blob': {
            r'$type': 'blob',
            'ref': {
              r'$link': calls == 1 ? 'bafynewvideo' : 'bafyrecoveredvideo',
            },
            'mimeType': 'video/mp4',
            'size': 12,
          },
        },
      }),
      200,
      headers: {
        Headers.contentTypeHeader: ['application/json'],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}
