import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/composer/post_edit_origin.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/video_upload_feature_provider.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/feed/widgets/published_post_editor.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/rich_text/data/mock_facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/facet_generator.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../../fakes/recording_messenger.dart';
import '../../test_support/post_fixtures.dart';
import '../../test_support/session_registry_storage.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  testWidgets(
    'AT-002 discarding a pending material edit preserves the published entry',
    (tester) async {
      PostEditChanges? saved;
      await _pumpMaterialEditor(tester, onSave: (changes) => saved = changes);
      await _openMaterialDetails(tester);
      await tester.tap(find.byKey(const Key('materials-edit-Wool')));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(EditableText).first, 'Merino wool');
      FocusManager.instance.primaryFocus?.unfocus();
      await tester.pumpAndSettle();
      await tester.tap(find.byType(BackButton).last);
      await tester.pumpAndSettle();
      expect(find.text('Discard material changes?'), findsOneWidget);
      await tester.tap(find.text('Discard'));
      await tester.pumpAndSettle();
      await _save(tester);
      expect(saved, isNotNull);
      expect(saved!.toMap(), isEmpty);
    },
  );

  testWidgets('AT-002 Keep editing retains pending material input until Add', (
    tester,
  ) async {
    PostEditChanges? saved;
    await _pumpMaterialEditor(tester, onSave: (changes) => saved = changes);
    await _openMaterialDetails(tester);
    await tester.tap(find.byKey(const Key('materials-edit-Wool')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(EditableText).first, 'Merino wool');
    FocusManager.instance.primaryFocus?.unfocus();
    await tester.pumpAndSettle();
    await tester.tap(find.byType(BackButton).last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Keep editing'));
    await tester.pumpAndSettle();
    expect(
      tester
          .widget<EditableText>(find.byType(EditableText).first)
          .controller
          .text,
      'Merino wool',
    );
    await tester.tap(find.byKey(const Key('materials-add-custom')));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(BackButton).last);
    await tester.pumpAndSettle();
    await _openMaterialDetails(tester);
    expect(find.byKey(const Key('materials-edit-Merino wool')), findsOneWidget);
    await tester.tap(find.byType(BackButton).last);
    await tester.pumpAndSettle();
    await _save(tester);
    expect(saved!.toMap(), {
      'project': {
        'common': {
          'materials': [
            {'existingIndex': 0, 'text': 'Merino wool'},
          ],
        },
      },
    });
  });

  testWidgets(
    'UT-003 leaving an unchanged material edit does not change the post',
    (tester) async {
      PostEditChanges? saved;
      await _pumpMaterialEditor(tester, onSave: (changes) => saved = changes);
      await _openMaterialDetails(tester);
      await tester.tap(find.byKey(const Key('materials-edit-Wool')));
      await tester.pumpAndSettle();
      FocusManager.instance.primaryFocus?.unfocus();
      await tester.pumpAndSettle();
      await tester.tap(find.byType(BackButton).last);
      await tester.pumpAndSettle();
      expect(find.text('Discard material changes?'), findsNothing);
      await _save(tester);
      expect(saved!.toMap(), isEmpty);
    },
  );

  testWidgets(
    'UT-002 explicit material removal still clears the published entry',
    (tester) async {
      PostEditChanges? saved;
      await _pumpMaterialEditor(tester, onSave: (changes) => saved = changes);
      await _openMaterialDetails(tester);
      await tester.tap(find.byKey(const Key('materials-remove-Wool')));
      await tester.pumpAndSettle();
      await tester.tap(find.byType(BackButton).last);
      await tester.pumpAndSettle();
      await _save(tester);
      expect(saved!.toMap(), {
        'project': {
          'common': {'materials': null},
        },
      });
    },
  );

  testWidgets(
    'UT-006 cancelling after a discarded material edit performs no write',
    (tester) async {
      var writes = 0;
      await _pumpMaterialEditor(tester, onSave: (_) => writes++);
      await _openMaterialDetails(tester);
      await tester.tap(find.byKey(const Key('materials-edit-Wool')));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(EditableText).first, 'Merino wool');
      FocusManager.instance.primaryFocus?.unfocus();
      await tester.pumpAndSettle();
      await tester.tap(find.byType(BackButton).last);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Discard'));
      await tester.pumpAndSettle();
      await tester.tap(find.byType(CloseButton).first);
      await tester.pumpAndSettle();
      expect(find.text('Open edit'), findsOneWidget);
      expect(writes, 0);
    },
  );

  for (final project in [false, true]) {
    testWidgets('UT-002 published editor clears the final language '
        '(project=$project)', (tester) async {
      final seed = _seed(project: project);
      PostEditChanges? saved;
      await _pumpEditor(
        tester,
        FakePostRepository(
          onFetchEdit: (did, key) async => seed,
          onEdit:
              (
                did,
                key, {
                required changes,
                required expectedCid,
                required operationKey,
              }) async {
                saved = changes;
                return PostEditResult(
                  post: seed.post,
                  acceptedCid: seed.baseCid,
                  changed: false,
                  projectionPending: false,
                );
              },
        ),
        seed: seed,
      );
      final control = find.byKey(const Key('composer-language-control'));
      await tester.ensureVisible(control);
      await tester.tap(control);
      await tester.pumpAndSettle();
      await tester.tap(find.text('English'));
      await tester.pumpAndSettle();
      expect(find.text('English'), findsNothing);
      await _save(tester);
      expect(saved!.toMap(), {'langs': <String>[]});
    });
  }

  for (final project in [false, true]) {
    for (final omitted in [false, true]) {
      for (final revert in [false, true]) {
        testWidgets(
          'UT-002 optional published language assignment and reversion '
          '(project=$project, omitted=$omitted, revert=$revert)',
          (tester) async {
            final base = _seed(project: project);
            final editable = {...base.editable}..remove('langs');
            if (!omitted) editable['langs'] = <String>[];
            final seed = PostEditSeed(
              post: base.post.copyWith(langs: const []),
              baseCid: base.baseCid,
              recordRevision: base.recordRevision,
              editable: editable,
            );
            PostEditChanges? saved;
            await _pumpEditor(
              tester,
              FakePostRepository(
                onFetchEdit: (did, key) async => seed,
                onEdit:
                    (
                      did,
                      key, {
                      required changes,
                      required expectedCid,
                      required operationKey,
                    }) async {
                      saved = changes;
                      return PostEditResult(
                        post: seed.post,
                        acceptedCid: seed.baseCid,
                        changed: false,
                        projectionPending: false,
                      );
                    },
              ),
              seed: seed,
            );
            final control = find.byKey(const Key('composer-language-control'));
            await tester.ensureVisible(control);
            await tester.tap(control);
            await tester.pumpAndSettle();
            await tester.tap(find.text('Add language'));
            await tester.pumpAndSettle();
            await tester.enterText(find.byType(TextField).last, 'English');
            await tester.pump();
            await tester.tap(find.text('English').last);
            await tester.pumpAndSettle();
            if (revert) {
              await tester.tap(control);
              await tester.pumpAndSettle();
              await tester.tap(find.text('English'));
              await tester.pumpAndSettle();
            }
            await _save(tester);
            expect(saved, isNotNull);
            expect(
              saved!.toMap(),
              revert
                  ? isEmpty
                  : {
                      'langs': ['en'],
                    },
            );
          },
        );
      }
    }
  }

  for (final project in [false, true]) {
    for (final omitted in [false, true]) {
      testWidgets('AT-002 absent and empty published languages stay unselected '
          '(project=$project, omitted=$omitted)', (tester) async {
        final semantics = tester.ensureSemantics();
        final base = _seed(project: project);
        final editable = {...base.editable}..remove('langs');
        if (!omitted) editable['langs'] = <String>[];
        final seed = PostEditSeed(
          post: base.post.copyWith(langs: const []),
          baseCid: base.baseCid,
          recordRevision: base.recordRevision,
          editable: editable,
        );
        PostEditChanges? saved;
        await _pumpEditor(
          tester,
          FakePostRepository(
            onFetchEdit: (did, key) async => seed,
            onEdit:
                (
                  did,
                  key, {
                  required changes,
                  required expectedCid,
                  required operationKey,
                }) async {
                  saved = changes;
                  return PostEditResult(
                    post: seed.post,
                    acceptedCid: seed.baseCid,
                    changed: false,
                    projectionPending: false,
                  );
                },
          ),
          seed: seed,
        );
        try {
          expect(
            tester.getSemantics(find.bySemanticsLabel('Post languages')).value,
            isEmpty,
          );
        } finally {
          semantics.dispose();
        }
        await _save(tester);
        expect(saved, isNotNull);
        expect(saved!.toMap(), isEmpty);
      });
    }
  }

  for (final project in [false, true]) {
    for (final cancel in [false, true]) {
      testWidgets('UT-006 edit save and discard preserve unrelated drafts '
          '(project=$project, cancel=$cancel)', (tester) async {
        final seed = _seed(project: project);
        final original = LocalPostDraft(
          id: 'unrelated-draft',
          owner: AccountKey('did:plc:alice'),
          kind: LocalPostDraftKind.standard,
          createdAt: DateTime.utc(2026),
          updatedAt: DateTime.utc(2026),
          revision: 4,
          content: const StandardDraftContent(
            text: 'Unrelated original',
            languages: ['fr'],
          ),
          schedule: DraftScheduleIntent.later(
            scheduledAtUtc: DateTime.utc(2030),
            savedOffsetMinutes: 60,
          ),
          media: const [],
        );
        final drafts = _UntouchedDraftRepository(original);
        var edits = 0;
        var creates = 0;
        await _pumpEditor(
          tester,
          FakePostRepository(
            onFetchEdit: (did, key) async => seed,
            onCreate: ({required text, images, reply}) async {
              creates++;
              return seed.post;
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
                  final post = seed.post.copyWith(
                    cid: 'bafyaccepted',
                    text: 'Working caption',
                  );
                  return PostEditResult(
                    post: post,
                    acceptedCid: post.cid,
                    changed: true,
                    projectionPending: false,
                    editedAt: DateTime.utc(2026),
                    editTimeSource: PostEditTimeSource.confirmed,
                  );
                },
          ),
          seed: seed,
          drafts: drafts,
        );
        await tester.ensureVisible(_bodyField(project));
        await tester.enterText(_bodyField(project), 'Working caption');
        FocusManager.instance.primaryFocus?.unfocus();
        await tester.pumpAndSettle();
        expect(find.text('Save draft'), findsNothing);
        if (cancel) {
          await tester.tap(find.byType(CloseButton).first);
          await tester.pumpAndSettle();
          await tester.tap(find.text('Discard'));
          await tester.pumpAndSettle();
        } else {
          await _save(tester);
        }
        expect(edits, cancel ? 0 : 1);
        expect(creates, 0);
        expect(drafts.saves, 0);
        expect(drafts.deletes, 0);
        expect(await drafts.get(original.id), same(original));
        expect(find.text('Open edit'), findsOneWidget);
      });
    }
  }

  for (final project in [false, true]) {
    testWidgets('AT-008 server validation names the field and keeps input '
        '(project=$project)', (tester) async {
      final seed = _seed(project: project);
      final messenger = RecordingMessenger();
      var reject = true;
      final keys = <String>[];
      await _pumpEditor(
        tester,
        FakePostRepository(
          onFetchEdit: (did, key) async => seed,
          onEdit:
              (
                did,
                key, {
                required changes,
                required expectedCid,
                required operationKey,
              }) async {
                keys.add(operationKey);
                expect(expectedCid, seed.baseCid);
                if (reject) {
                  throw const ApiBadRequest(
                    'invalid_post',
                    details: ApiFailureDetails(
                      statusCode: 422,
                      fields: {'text': 'PRIVATE_VALIDATION_CANARY'},
                    ),
                  );
                }
                final post = seed.post.copyWith(
                  cid: 'bafyaccepted',
                  text: 'Corrected caption',
                );
                return PostEditResult(
                  post: post,
                  acceptedCid: post.cid,
                  changed: true,
                  projectionPending: false,
                  editedAt: DateTime.utc(2026),
                  editTimeSource: PostEditTimeSource.confirmed,
                );
              },
        ),
        seed: seed,
        messenger: messenger,
      );
      await tester.ensureVisible(_bodyField(project));
      await tester.enterText(_bodyField(project), 'Keep this caption');
      await _save(tester);
      expect(
        tester.widget<TextField>(_bodyField(project)).controller?.text,
        'Keep this caption',
      );
      expect(
        messenger.calls.last.$2,
        'Check these fields and try again: Caption.',
      );
      reject = false;
      await tester.ensureVisible(_bodyField(project));
      await tester.enterText(_bodyField(project), 'Corrected caption');
      await _save(tester);
      expect(keys, hasLength(2));
      expect(keys[0], isNot(keys[1]));
      expect(find.text('Open edit'), findsOneWidget);
    });
  }

  testWidgets(
    'IT-009 unsupported embeds cannot be replaced by caption previews',
    (tester) async {
      final base = _seed();
      final post = base.post.copyWith(images: null);
      final seed = PostEditSeed(
        post: post,
        baseCid: post.cid,
        recordRevision: base.recordRevision,
        editable: {
          ...base.editable,
          'images': null,
          'unsupportedEmbed': true,
        },
      );
      final previews = _FixedPreviewRepository();
      await _pumpEditor(
        tester,
        FakePostRepository(onFetchEdit: (did, key) async => seed),
        seed: seed,
        previews: previews,
      );
      await tester.enterText(
        _bodyField(false),
        'Caption https://example.com/pattern ',
      );
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('link-preview-carousel')), findsNothing);
      expect(previews.calls, 0);
    },
  );

  testWidgets('AT-002 a published reply retains editable image attachments', (
    tester,
  ) async {
    final base = _seed();
    final target = postFixture(did: 'did:plc:parent', rkey: 'parent');
    final reference = PostRef(uri: target.uri, cid: target.cid);
    final post = base.post.copyWith(
      reply: PostReply(root: reference, parent: reference),
    );
    final seed = PostEditSeed(
      post: post,
      baseCid: post.cid,
      recordRevision: base.recordRevision,
      editable: base.editable,
    );
    await _pumpEditor(
      tester,
      FakePostRepository(
        onFetchEdit: (did, key) async => seed,
        onFetch: (did, key) async => target,
      ),
      seed: seed,
    );
    expect(find.byType(ComposerImageAttachmentSection), findsOneWidget);
  });

  testWidgets('AT-002 an editable general post can select a new video', (
    tester,
  ) async {
    final base = _seed();
    final post = base.post.copyWith(images: null);
    final seed = PostEditSeed(
      post: post,
      baseCid: post.cid,
      recordRevision: base.recordRevision,
      editable: {
        ...base.editable,
        'images': null,
      },
    );
    await _pumpEditor(
      tester,
      FakePostRepository(onFetchEdit: (did, key) async => seed),
      seed: seed,
      videoEnabled: true,
    );
    expect(find.text('Add photos or a video'), findsOneWidget);
  });

  testWidgets(
    'IT-009 editing shared project content retains an existing future craft',
    (tester) async {
      final base = _seed(project: true);
      const future = 'social.craftsky.feed.defs#futureCraft';
      final post = base.post.copyWith(
        project: base.post.project!.copyWith.common(craftType: future),
      );
      final seed = PostEditSeed(
        post: post,
        baseCid: post.cid,
        recordRevision: base.recordRevision,
        editable: {
          ...base.editable,
          'project': {
            'common': {'craftType': future, 'title': 'Linen shirt'},
          },
        },
      );
      Map<String, Object?>? submitted;
      await _pumpEditor(
        tester,
        FakePostRepository(
          onFetchEdit: (did, key) async => seed,
          onEdit:
              (
                did,
                key, {
                required changes,
                required expectedCid,
                required operationKey,
              }) async {
                submitted = changes.toMap();
                final accepted = post.copyWith(
                  cid: 'bafyaccepted',
                  text: 'Updated project caption',
                );
                return PostEditResult(
                  post: accepted,
                  acceptedCid: accepted.cid,
                  changed: true,
                  projectionPending: false,
                  editedAt: DateTime.utc(2026),
                  editTimeSource: PostEditTimeSource.confirmed,
                );
              },
        ),
        seed: seed,
      );
      expect(
        find.byKey(const Key('project-composer-craft-details-action')),
        findsNothing,
      );
      await tester.ensureVisible(_bodyField(true));
      await tester.enterText(_bodyField(true), 'Updated project caption');
      await _save(tester);
      expect(submitted?['text'], 'Updated project caption');
      expect(submitted?.containsKey('project'), isFalse);
      expect(find.text('Open edit'), findsOneWidget);
    },
  );
  for (final project in [false, true]) {
    for (final failure in [false, true]) {
      testWidgets(
        'AT-007 late save completion cannot expose input or '
        'feedback to another account (project=$project, failure=$failure)',
        (tester) async {
          final initial = _seed(project: project);
          final completion = Completer<PostEditResult>();
          final messenger = RecordingMessenger();
          var writes = 0;
          final container = await _pumpEditor(
            tester,
            FakePostRepository(
              onEdit:
                  (
                    did,
                    key, {
                    required changes,
                    required expectedCid,
                    required operationKey,
                  }) {
                    writes++;
                    return completion.future;
                  },
            ),
            seed: initial,
            direct: true,
            messenger: messenger,
          );
          final body = _bodyField(project);
          await tester.ensureVisible(body);
          await tester.enterText(body, 'ALICE_PRIVATE_UNSAVED');
          FocusManager.instance.primaryFocus?.unfocus();
          await tester.pumpAndSettle();
          await tester.tap(find.widgetWithText(ChunkyButton, 'Save changes'));
          await tester.pump();
          await tester.pump(const Duration(milliseconds: 100));
          expect(writes, 1);
          final feedbackBefore = messenger.calls.length;
          await container
              .read(sessionRegistryProvider.notifier)
              .upsertAndActivate(
                token: 'bob-token',
                did: 'did:plc:bob',
                handle: 'bob.example',
              );
          if (failure) {
            completion.completeError(const ApiNetworkError('late failure'));
          } else {
            final accepted = initial.post.copyWith(cid: 'bafyaccepted');
            completion.complete(
              PostEditResult(
                post: accepted,
                acceptedCid: accepted.cid,
                changed: true,
                projectionPending: false,
                editedAt: DateTime.utc(2026),
                editTimeSource: PostEditTimeSource.confirmed,
              ),
            );
          }
          await tester.pump();
          await tester.pump(const Duration(milliseconds: 500));
          expect(find.byType(TextField), findsNothing);
          expect(find.text('Open edit'), findsOneWidget);
          expect(messenger.calls, hasLength(feedbackBefore));
          expect(writes, 1);
        },
      );
    }
  }
  for (final project in [false, true]) {
    testWidgets(
      'AT-006 unresolved save freezes input and retries the '
      'exact attempt (project=$project)',
      (tester) async {
        final initial = _seed(project: project);
        final keys = <String>[];
        final bodies = <Map<String, Object?>>[];
        final messenger = RecordingMessenger();
        final facets = _CountingFacetGenerator();
        await _pumpEditor(
          tester,
          FakePostRepository(
            onFetchEdit: (did, key) async => initial,
            onEdit:
                (
                  did,
                  key, {
                  required changes,
                  required expectedCid,
                  required operationKey,
                }) async {
                  keys.add(operationKey);
                  bodies.add(changes.toMap());
                  if (keys.length <= 7) {
                    throw const PdsMutationAmbiguousException(
                      retryAfterSeconds: 1,
                    );
                  }
                  final accepted = initial.post.copyWith(
                    cid: 'bafyaccepted',
                    text: 'My uncertain correction',
                  );
                  return PostEditResult(
                    post: accepted,
                    acceptedCid: accepted.cid,
                    changed: true,
                    projectionPending: false,
                    editedAt: DateTime.utc(2026),
                    editTimeSource: PostEditTimeSource.confirmed,
                  );
                },
          ),
          seed: initial,
          facets: facets,
          messenger: messenger,
        );
        final body = _bodyField(project);
        await tester.ensureVisible(body);
        await tester.enterText(body, 'My uncertain correction');
        await _save(tester);
        expect(keys, hasLength(7));
        expect(messenger.calls.last.$2, contains('not confirmed'));
        final field = tester.widget<TextField>(body);
        expect(field.readOnly || field.enabled == false, isTrue);
        expect(field.controller?.text, 'My uncertain correction');
        await tester.tap(find.widgetWithText(ChunkyButton, 'Retry save'));
        await tester.pumpAndSettle();
        expect(keys, hasLength(8));
        expect(facets.calls, 1);
        expect(keys.toSet(), hasLength(1));
        expect(bodies, everyElement(bodies.first));
        expect(find.text('Open edit'), findsOneWidget);
      },
    );
  }
  for (final project in [false, true]) {
    testWidgets(
      'AT-005 deleted post preserves input and disables '
      'resubmission (project=$project)',
      (tester) async {
        var writes = 0;
        final messenger = RecordingMessenger();
        final initial = _seed(project: project);
        await _pumpEditor(
          tester,
          FakePostRepository(
            onFetchEdit: (did, key) async => initial,
            onEdit:
                (
                  did,
                  key, {
                  required changes,
                  required expectedCid,
                  required operationKey,
                }) async {
                  writes++;
                  throw const ApiBadRequest(
                    'post_not_found',
                    details: ApiFailureDetails(statusCode: 404),
                  );
                },
          ),
          seed: initial,
          messenger: messenger,
        );
        final body = _bodyField(project);
        await tester.ensureVisible(body);
        await tester.enterText(body, 'Keep my correction');
        await _save(tester);
        expect(writes, 1);
        expect(
          tester.widget<TextField>(body).controller?.text,
          'Keep my correction',
        );
        expect(messenger.calls.last.$2, contains('no longer available'));
        expect(
          tester
              .widget<ChunkyButton>(
                find.widgetWithText(ChunkyButton, 'Save changes'),
              )
              .onPressed,
          isNull,
        );
      },
    );
  }
  for (final project in [false, true]) {
    testWidgets(
      'AT-005 conflict review keeps input and fetches a fresh '
      'base (project=$project)',
      (tester) async {
        final initial = _seed(project: project);
        final currentPost = initial.post.copyWith(
          cid: 'bafycurrent',
          text: 'Other client version',
        );
        final current = PostEditSeed(
          post: currentPost,
          baseCid: currentPost.cid,
          recordRevision: '3aaaaaaaaaaa3',
          editable: {...initial.editable, 'text': currentPost.text},
        );
        var reads = 0;
        var writes = 0;
        final messenger = RecordingMessenger();
        await _pumpEditor(
          tester,
          FakePostRepository(
            onFetchEdit: (did, key) async => ++reads == 1 ? initial : current,
            onEdit:
                (
                  did,
                  key, {
                  required changes,
                  required expectedCid,
                  required operationKey,
                }) async {
                  writes++;
                  if (writes == 1) {
                    throw const ApiBadRequest(
                      'pds_record_conflict',
                      details: ApiFailureDetails(statusCode: 409),
                    );
                  }
                  expect(expectedCid, current.baseCid);
                  expect(changes.toMap()['text'], 'Reviewed correction');
                  final accepted = currentPost.copyWith(
                    cid: 'bafyaccepted',
                    text: 'Reviewed correction',
                  );
                  return PostEditResult(
                    post: accepted,
                    acceptedCid: accepted.cid,
                    changed: true,
                    projectionPending: false,
                    editedAt: DateTime.utc(2026),
                    editTimeSource: PostEditTimeSource.confirmed,
                  );
                },
          ),
          seed: initial,
          messenger: messenger,
        );
        final body = _bodyField(project);
        await tester.ensureVisible(body);
        await tester.enterText(body, 'My correction');
        await _save(tester);
        expect(writes, 1);
        expect(
          tester.widget<TextField>(body).controller?.text,
          'My correction',
        );
        expect(messenger.calls.last.$2, contains('changed'));
        final review = messenger.calls.last.$3;
        expect(review?.label, 'Review current version');
        review!.onPressed();
        await tester.pumpAndSettle();
        expect(
          tester.widget<TextField>(_bodyField(project)).controller?.text,
          'Other client version',
        );
        await tester.tap(find.byType(CloseButton).last);
        await tester.pumpAndSettle();
        expect(
          tester.widget<TextField>(_bodyField(project)).controller?.text,
          'My correction',
        );
        review.onPressed();
        await tester.pumpAndSettle();
        expect(reads, 3);
        await tester.ensureVisible(_bodyField(project));
        await tester.enterText(_bodyField(project), 'Reviewed correction');
        await _save(tester);
        expect(writes, 2);
        expect(find.text('Open edit'), findsOneWidget);
      },
    );
  }
  testWidgets(
    'AT-006 failed seed loading retries without opening an empty editor',
    (tester) async {
      var reads = 0;
      await _pumpEditor(
        tester,
        FakePostRepository(
          onFetchEdit: (did, key) async {
            reads++;
            if (reads == 1) throw const ApiNetworkError('offline');
            return _seed();
          },
        ),
      );
      expect(find.byType(TextField), findsNothing);
      expect(
        find.text('Could not load this post for editing. Please try again.'),
        findsOneWidget,
      );
      await tester.tap(find.text('Retry'));
      await tester.pumpAndSettle();
      expect(reads, 2);
      expect(
        tester.widget<TextField>(find.byType(TextField).first).controller?.text,
        'Original caption',
      );
      expect(find.text('Save changes'), findsOneWidget);
    },
  );

  testWidgets(
    'AT-007 account change discards a late seed without showing private input',
    (tester) async {
      final loading = Completer<PostEditSeed>();
      final container = await _pumpEditor(
        tester,
        FakePostRepository(onFetchEdit: (did, key) => loading.future),
        settle: false,
      );
      await container
          .read(sessionRegistryProvider.notifier)
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.example',
          );
      loading.complete(_seed());
      await tester.pumpAndSettle();
      expect(find.text('Open edit'), findsOneWidget);
      expect(find.byType(TextField), findsNothing);
      expect(find.text('Original caption'), findsNothing);
    },
  );
}

Future<void> _pumpMaterialEditor(
  WidgetTester tester, {
  required ValueChanged<PostEditChanges> onSave,
}) async {
  final base = _seed(project: true);
  final rawProject = Map<String, Object?>.from(
    base.editable['project']! as Map,
  );
  final common = Map<String, Object?>.from(rawProject['common']! as Map);
  common['materials'] = [
    {'text': 'Wool', 'existingIndex': 0},
  ];
  rawProject['common'] = common;
  final seed = PostEditSeed(
    post: base.post,
    baseCid: base.baseCid,
    recordRevision: base.recordRevision,
    editable: {...base.editable, 'project': rawProject},
  );
  await _pumpEditor(
    tester,
    FakePostRepository(
      onFetchEdit: (did, key) async => seed,
      onEdit:
          (
            did,
            key, {
            required changes,
            required expectedCid,
            required operationKey,
          }) async {
            onSave(changes);
            return PostEditResult(
              post: seed.post,
              acceptedCid: seed.baseCid,
              changed: false,
              projectionPending: false,
            );
          },
    ),
    seed: seed,
  );
}

Future<void> _openMaterialDetails(WidgetTester tester) async {
  final details = find.byKey(
    const Key('project-composer-common-details-action'),
  );
  await tester.ensureVisible(details);
  await tester.tap(details);
  await tester.pumpAndSettle();
}

PostEditSeed _seed({bool project = false}) {
  final base = postFixture(rkey: 'base', text: 'Original caption');
  final post = base.copyWith(
    project: project
        ? const Project(
            common: ProjectCommon(
              craftType: 'social.craftsky.feed.defs#sewing',
              title: 'Linen shirt',
            ),
          )
        : null,
    images: project
        ? [
            PostImage(
              cid: 'bafyimage',
              mime: 'image/jpeg',
              size: 100,
              alt: 'Linen shirt',
              fullsize: 'https://cdn.example/image.jpg',
            ),
          ]
        : null,
  );
  return PostEditSeed(
    post: post,
    baseCid: post.cid,
    recordRevision: '3aaaaaaaaaaa2',
    editable: {
      'text': post.text,
      'sponsored': false,
      'langs': ['en'],
      if (project)
        'project': {
          'common': {
            'craftType': 'social.craftsky.feed.defs#sewing',
            'title': 'Linen shirt',
          },
        },
      if (project)
        'images': [
          {
            'existingIndex': 0,
            'alt': 'Linen shirt',
            'image': {'mimeType': 'image/jpeg'},
          },
        ],
    },
  );
}

Future<ProviderContainer> _pumpEditor(
  WidgetTester tester,
  FakePostRepository repository, {
  bool settle = true,
  RecordingMessenger? messenger,
  PostEditSeed? seed,
  FacetGenerator? facets,
  bool direct = false,
  bool videoEnabled = false,
  LinkPreviewRepository? previews,
  LocalPostDraftRepository? drafts,
}) async {
  final registry = SessionRegistry.empty().upsertAndActivate(
    token: 'alice-token',
    did: 'did:plc:alice',
    handle: 'alice.example',
  );
  await tester.pumpWidget(
    ProviderScope(
      retry: (_, _) => null,
      overrides: [
        videoUploadsEnabledProvider.overrideWithValue(videoEnabled),
        if (drafts != null)
          accountLocalPostDraftRepositoryProvider.overrideWith(
            (ref, account) async => drafts,
          ),
        if (previews != null)
          linkPreviewRepositoryProvider.overrideWithValue(previews),
        secureSessionRegistryStorageProvider.overrideWithValue(
          InMemorySessionRegistryStorage(registry),
        ),
        postRepositoryProvider.overrideWithValue(repository),
        if (facets != null) facetGeneratorProvider.overrideWithValue(facets),
        pdsMutationDelayProvider.overrideWithValue((_) async {}),
        pdsMutationJitterProvider.overrideWithValue((_) => 0),
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
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
        messenger: messenger ?? RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: ElevatedButton(
                onPressed: () => direct
                    ? ((seed ?? _seed()).post.project == null
                          ? showPostComposerSheet(
                              context,
                              publishedEdit: PostEditOrigin(
                                seed: seed ?? _seed(),
                                ownership: registry.activeLease!,
                              ),
                            )
                          : showProjectComposerSheet(
                              context,
                              publishedEdit: PostEditOrigin(
                                seed: seed ?? _seed(),
                                ownership: registry.activeLease!,
                              ),
                            ))
                    : showPublishedPostEditor(
                        context,
                        post: (seed ?? _seed()).post,
                        ownership: registry.activeLease!,
                      ),
                child: const Text('Open edit'),
              ),
            ),
          ),
        ),
      ),
    ),
  );
  final container = ProviderScope.containerOf(
    tester.element(find.byType(MaterialApp)),
  );
  await container.read(sessionRegistryProvider.future);
  await tester.tap(find.text('Open edit'));
  if (settle) {
    await tester.pumpAndSettle();
  } else {
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
  }
  return container;
}

Finder _bodyField(bool project) => project
    ? find.descendant(
        of: find.byKey(const Key('project-composer-body-editor')),
        matching: find.byType(TextField),
      )
    : find.byType(TextField).first;

Future<void> _save(WidgetTester tester) async {
  FocusManager.instance.primaryFocus?.unfocus();
  await tester.pumpAndSettle();
  await tester.tap(find.widgetWithText(ChunkyButton, 'Save changes'));
  await tester.pumpAndSettle();
}

final class _CountingFacetGenerator extends Fake implements FacetGenerator {
  int calls = 0;
  @override
  Future<List<Map<String, dynamic>>> generate(
    String text, {
    bool includeMentions = true,
    bool includeLinks = true,
    bool includeTags = true,
  }) async {
    calls++;
    return [];
  }
}

final class _FixedPreviewRepository implements LinkPreviewRepository {
  int calls = 0;
  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) async {
    calls++;
    return LinkPreview(
      url: url,
      title: 'Fetched preview',
      description: 'A pattern',
    );
  }
}

final class _UntouchedDraftRepository implements LocalPostDraftRepository {
  new(this.original);
  final LocalPostDraft original;
  int saves = 0;
  int deletes = 0;
  @override
  Future<List<LocalPostDraft>> list() async => [original];
  @override
  Future<LocalPostDraft> get(String draftId) async => original;
  @override
  Future<LocalPostDraft> save(DraftWriteRequest request) async {
    saves++;
    return original;
  }

  @override
  Future<void> delete(String draftId) async {
    deletes++;
  }

  @override
  Future<Uint8List> readMedia(String draftId, String mediaId) async =>
      throw UnimplementedError();
}
