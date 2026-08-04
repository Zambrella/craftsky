import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/composer/draft_schedule_restoration.dart';
import 'package:craftsky_app/drafts/composer/draft_submission_origin.dart';
import 'package:craftsky_app/drafts/composer/standard_draft_snapshot_adapter.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/draft_save_controller.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/drafts/providers/local_post_drafts_provider.dart';
import 'package:craftsky_app/drafts/widgets/draft_close_dialog.dart';
import 'package:craftsky_app/feed/composer/composer_media_uploader.dart';
import 'package:craftsky_app/feed/composer/composer_submission_coordinator.dart';
import 'package:craftsky_app/feed/composer/submission_screen_awake.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/feed/widgets/submission_blocking_overlay.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/post_language_selection.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/languages/widgets/post_language_selector.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/scheduled_posts/composer/schedule_capacity_state.dart';
import 'package:craftsky_app/scheduled_posts/composer/schedule_composer_state.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_post_repository_provider.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_posts_provider.dart';
import 'package:craftsky_app/scheduled_posts/services/scheduled_composer_media.dart';
import 'package:craftsky_app/scheduled_posts/widgets/schedule_choice_menu.dart';
import 'package:craftsky_app/scheduled_posts/widgets/schedule_time_picker.dart';
import 'package:craftsky_app/scheduled_posts/widgets/scheduled_post_capacity_warning.dart';
import 'package:craftsky_app/scheduled_posts/widgets/scheduled_staging_progress.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/shared/rich_text/widgets/facet_autocomplete_editor.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:uuid/uuid.dart';

Future<Post?> showPostComposerSheet(
  BuildContext context, {
  Post? replyTarget,
  Post? quoteTarget,
  ScheduledPostDetail? scheduledPost,
  ActiveAccountLease? scheduledOwner,
  LocalPostDraftSeed? draftSeed,
  ActiveAccountLease? draftOwner,
}) {
  return Navigator.of(context, rootNavigator: true).push<Post?>(
    MaterialPageRoute<Post?>(
      fullscreenDialog: true,
      builder: (_) => PostComposerSheet(
        replyTarget: replyTarget,
        quoteTarget: quoteTarget,
        scheduledPost: scheduledPost,
        scheduledOwner: scheduledOwner,
        draftSeed: draftSeed,
        draftOwner: draftOwner,
      ),
    ),
  );
}

class PostComposerSheet extends ConsumerStatefulWidget {
  const PostComposerSheet({
    super.key,
    this.replyTarget,
    this.quoteTarget,
    this.composerId,
    this.scheduledPost,
    this.scheduledOwner,
    this.draftSeed,
    this.draftOwner,
  }) : assert(
         replyTarget == null || quoteTarget == null,
         'replyTarget and quoteTarget are mutually exclusive',
       ),
       assert(
         scheduledPost == null || (replyTarget == null && quoteTarget == null),
         'Scheduled edits are top-level posts',
       ),
       assert(
         draftSeed == null ||
             (replyTarget == null &&
                 quoteTarget == null &&
                 scheduledPost == null),
         'Local drafts are top-level posts',
       );

  static const maxCharacters = 2000;

  final Post? replyTarget;
  final Post? quoteTarget;
  final String? composerId;
  final ScheduledPostDetail? scheduledPost;
  final ActiveAccountLease? scheduledOwner;
  final LocalPostDraftSeed? draftSeed;
  final ActiveAccountLease? draftOwner;

  @override
  ConsumerState<PostComposerSheet> createState() => _PostComposerSheetState();
}

class _PostComposerSheetState extends ConsumerState<PostComposerSheet> {
  final _controller = FacetTextEditingController();
  final _focusNode = FocusNode(debugLabel: 'postComposerText');
  late final String _composerId;
  String _initialText = '';
  String _text = '';
  AccountSessionLease? _unsavedOwner;
  UnsavedWorkRegistration? _unsavedRegistration;
  late final UnsavedWorkGuard _unsavedGuard;
  PostLanguageSelection? _languages;
  ScheduleChoice _scheduleChoice = ScheduleChoice.now;
  DateTime? _scheduledAtLocal;
  DateTime? _missedScheduledAtLocal;
  var _isScheduling = false;
  var _stagedImageCount = 0;
  var _stagedImageTotal = 0;
  var _isSavingSchedule = false;
  var _isLoadingScheduledMedia = false;
  var _scheduledMediaLoadFailed = false;
  ActiveAccountLease? _scheduledOwner;
  late final ComposerMediaUploader _mediaUploader;
  late final ComposerSubmissionCoordinator _submissionCoordinator;
  var _isSubmitting = false;
  var _isSavingDraft = false;
  var _submissionSucceeded = false;
  late final DraftSubmissionOrigin _origin;
  List<String>? _initialLanguages;
  ScheduleChoice _initialScheduleChoice = ScheduleChoice.now;
  DateTime? _initialScheduledAtLocal;

  @override
  void initState() {
    super.initState();
    _unsavedGuard = ref.read(unsavedWorkGuardProvider);
    _composerId = widget.composerId ?? const Uuid().v4();
    _origin = DraftSubmissionOrigin(widget.draftSeed?.draft);
    _mediaUploader = ComposerMediaUploader();
    _submissionCoordinator = ComposerSubmissionCoordinator(
      screenAwake: const WakelockSubmissionScreenAwake(),
    );
    _scheduledOwner = widget.scheduledOwner;
    if (widget.draftSeed case final seed?) {
      final content = seed.draft.content;
      if (content is StandardDraftContent) {
        _text = content.text;
        _initialText = content.text;
        _controller.text = content.text;
        if (content.languages.isNotEmpty) {
          _languages = PostLanguageSelection.fromValues(content.languages);
          _initialLanguages = List.of(content.languages);
        }
      }
      final restored = restoreDraftSchedule(
        seed.draft.schedule,
        now: DateTime.now(),
      );
      _scheduleChoice = restored.choice;
      _scheduledAtLocal = restored.scheduledAtLocal;
      _missedScheduledAtLocal = restored.needsExplanation
          ? seed.draft.schedule.scheduledAtUtc?.toLocal()
          : null;
      _initialScheduleChoice = _scheduleChoice;
      _initialScheduledAtLocal = _scheduledAtLocal;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          ref
              .read(composerImagesProvider(_composerId).notifier)
              .seedLocalDraft(seed);
        }
      });
    }
    if (widget.scheduledPost case final scheduled?) {
      _text = scheduled.payload['text'] as String? ?? '';
      _initialText = _text;
      _controller.text = _text;
      final langs = (scheduled.payload['langs'] as List<dynamic>? ?? const [])
          .cast<String>();
      if (langs.isNotEmpty) {
        _languages = PostLanguageSelection.fromValues(langs);
      }
      if (scheduled.status == ScheduledPostStatus.needsAttention) {
        _scheduleChoice = ScheduleChoice.now;
        _missedScheduledAtLocal = scheduled.scheduledAt.utc.toLocal();
      } else {
        _scheduleChoice = ScheduleChoice.later;
        _scheduledAtLocal = scheduled.scheduledAt.utc.toLocal();
      }
      if (_scheduledPayloadMedia.isNotEmpty) {
        _isLoadingScheduledMedia = true;
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted) unawaited(_hydrateScheduledMedia());
        });
      }
    }
    if (widget.replyTarget?.reply != null) {
      _text = '@${widget.replyTarget!.author.handle} ';
      _controller.text = _text;
      _controller.selection = TextSelection.collapsed(offset: _text.length);
    }
    _initialText = _text;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      _focusNode.requestFocus();
    });
  }

  @override
  void dispose() {
    _unsavedGuard.unregister(_unsavedRegistration);
    unawaited(_submissionCoordinator.dispose());
    _controller.dispose();
    _focusNode.dispose();
    _mediaUploader.disposeComposer(_composerId);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final createState = ref.watch(createPostProvider);
    final preferences = ref.watch(activeLanguagePreferencesProvider);
    _languages ??= PostLanguageSelection.fromPrimary(
      preferences.primaryLanguage,
    );
    _initialLanguages ??= List.of(_languages!.values);
    final imagesProvider = composerImagesProvider(_composerId);
    final imagesState = ref.watch(imagesProvider);
    final isReply = widget.replyTarget != null;
    final isQuote = widget.quoteTarget != null;
    final isSchedulable = !isReply && !isQuote;
    final activeLease = ref.watch(sessionRegistryProvider).value?.activeLease;
    if (widget.scheduledPost != null && _scheduledOwner == null) {
      _scheduledOwner = activeLease;
    }
    if (widget.scheduledPost != null &&
        _scheduledOwner != null &&
        activeLease != _scheduledOwner) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted && Navigator.of(context).canPop()) {
          Navigator.of(context).pop();
        }
      });
    }
    final account = activeLease?.session.account;
    if (account != null) {
      ref.listen(draftSaveControllerProvider(account), (_, _) {});
    }
    final scheduledCount = account == null
        ? 0
        : ref.watch(scheduledPostsProvider(account)).value?.items.length ?? 0;
    final capacity = ScheduleCapacityState.derive(
      scheduledCount: scheduledCount.clamp(0, 3),
      choice: _scheduleChoice,
      ownsExistingSlot: widget.scheduledPost != null,
    );
    final trimmedText = _text.trim();
    final tooLong = _text.length > PostComposerSheet.maxCharacters;
    final canSubmit =
        !createState.isLoading &&
        !_isSubmitting &&
        !_isScheduling &&
        !_isLoadingScheduledMedia &&
        !_scheduledMediaLoadFailed &&
        trimmedText.isNotEmpty &&
        !tooLong &&
        _languages != null &&
        imagesState.canSubmitImages() &&
        (_scheduleChoice == ScheduleChoice.now || capacity.scheduleEnabled);
    final submitLabel = _scheduleChoice == ScheduleChoice.later
        ? l10n.scheduledPostAction
        : isReply
        ? l10n.postComposeReplySubmit
        : l10n.postComposeSubmit;
    final hasDraft = _hasDraft(imagesState);
    final canSaveDraft = _canSaveDraft(imagesState, hasDraft: hasDraft);
    _ensureUnsavedWorkRegistration();

    ref
      ..listen(createPostProvider, (previous, next) {
        switch ((previous, next)) {
          case (AsyncLoading(), AsyncData(:final value?)):
            if (Navigator.of(context).canPop()) {
              Navigator.of(context).pop(value);
            }
            context.showInfo(l10n.postCreateSuccess);
            ref.read(createPostProvider.notifier).reset();
          case (AsyncLoading(), AsyncError()):
            context.showError(l10n.postCreateError);
            ref.read(createPostProvider.notifier).reset();
          case _:
            break;
        }
      })
      ..listen(imagesProvider, (previous, next) {
        final notice = next.notice;
        if (notice == null || previous?.notice?.id == notice.id) return;
        switch (notice) {
          case ImageSelectionLimitNotice(:final maxImages):
            context.showError(l10n.postComposeImageLimitError(maxImages));
          case UnsupportedImagesNotice(:final count):
            context.showError(l10n.postComposeUnsupportedImagesError(count));
          case ImagePickerFailedNotice():
            context.showError(l10n.postComposeImagePickerError);
        }
        ref.read(imagesProvider.notifier).clearNotice(notice.id);
      });

    return PopScope<Post?>(
      canPop: !_isSubmitting && (!hasDraft || createState.isLoading),
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        if (_isSubmitting) return;
        final shouldClose = await _confirmClose(imagesState);
        if (!shouldClose) return;
        if (!context.mounted) return;
        Navigator.of(context).pop();
      },
      child: Stack(
        fit: StackFit.expand,
        children: [
          Scaffold(
            backgroundColor: swatches.paper,
            appBar: AppBar(
              title: Text(
                isReply
                    ? l10n.postComposeReplyTitle
                    : isQuote
                    ? l10n.postQuoteAction
                    : l10n.postComposeTitle,
                style: theme.textTheme.titleLarge,
              ),
              actions: [
                if (widget.replyTarget == null &&
                    widget.quoteTarget == null &&
                    widget.scheduledPost == null)
                  TextButton(
                    onPressed: canSaveDraft
                        ? () => _saveDraft(imagesState)
                        : null,
                    child: Text(
                      widget.draftSeed == null
                          ? l10n.draftSaveAction
                          : l10n.draftSaveChangesAction,
                    ),
                  ),
              ],
            ),
            body: Stack(
              fit: StackFit.expand,
              children: [
                SafeArea(
                  top: false,
                  bottom: false,
                  child: SingleChildScrollView(
                    clipBehavior: Clip.none,
                    padding: EdgeInsets.fromLTRB(
                      spacing.sp4,
                      spacing.sp5,
                      spacing.sp4,
                      0,
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        if (widget.replyTarget case final replyTarget?) ...[
                          _ComposerTargetPreview(post: replyTarget),
                          SizedBox(height: spacing.sp4),
                        ],
                        if (widget.quoteTarget case final quoteTarget?) ...[
                          _ComposerTargetPreview(post: quoteTarget),
                          SizedBox(height: spacing.sp4),
                        ],
                        FacetAutocompleteEditor(
                          label: isReply
                              ? l10n.postComposeReplyHint
                              : l10n.postComposeHint,
                          hintText: isReply ? null : l10n.postComposeBodyHint,
                          controller: _controller,
                          focusNode: _focusNode,
                          minLines: isReply ? 5 : 3,
                          maxLines: 12,
                          textInputAction: TextInputAction.newline,
                          keyboardType: TextInputType.multiline,
                          enabled: !createState.isLoading,
                          errorText: tooLong ? l10n.postComposeTooLong : null,
                          helperText:
                              '${_text.length}/${PostComposerSheet.maxCharacters}',
                          helperAlignment: AlignmentDirectional.centerEnd,
                          onChanged: (value) => setState(() => _text = value),
                        ),
                        SizedBox(height: spacing.sp4),
                        PostLanguageSelector(
                          selection: _languages!,
                          enabled: !createState.isLoading,
                          onChanged: (value) =>
                              setState(() => _languages = value),
                        ),
                        if (isSchedulable) ...[
                          SizedBox(height: spacing.sp4),
                          Builder(
                            builder: (menuContext) => ListTile(
                              contentPadding: EdgeInsets.zero,
                              leading: const Icon(Icons.schedule_outlined),
                              title: Text(l10n.scheduledPostWhenTitle),
                              subtitle: Text(_whenLabel(context)),
                              trailing: const Icon(Icons.chevron_right),
                              enabled: !_isScheduling,
                              onTap: () => _chooseWhen(
                                menuContext,
                                scheduleEnabled: capacity.scheduleEnabled,
                              ),
                            ),
                          ),
                          if (capacity.showCapacityWarning)
                            const ScheduledPostCapacityWarning(),
                          if (capacity.showManageLink)
                            Align(
                              alignment: AlignmentDirectional.centerStart,
                              child: TextButton(
                                onPressed: () => const ScheduledPostsRoute()
                                    .push<void>(context),
                                child: Text(l10n.scheduledPostManageAction),
                              ),
                            ),
                          if (_missedScheduledAtLocal case final missed?)
                            Text(
                              l10n.scheduledPostMissedTime(
                                _localTimeLabel(context, missed),
                              ),
                            ),
                          if (widget.scheduledPost != null)
                            Align(
                              alignment: AlignmentDirectional.centerStart,
                              child: TextButton.icon(
                                onPressed: _isScheduling
                                    ? null
                                    : _deleteExistingSchedule,
                                icon: const Icon(Icons.delete_outline),
                                label: Text(l10n.scheduledPostsDeleteTooltip),
                              ),
                            ),
                          if (_isScheduling &&
                              (_stagedImageTotal > 0 || _isSavingSchedule)) ...[
                            SizedBox(height: spacing.sp3),
                            ScheduledStagingProgress(
                              completed: _stagedImageCount,
                              total: _stagedImageTotal,
                              creating: _isSavingSchedule,
                            ),
                          ],
                        ],
                        if (!isReply) ...[
                          SizedBox(height: spacing.sp6),
                          if (_isLoadingScheduledMedia)
                            const Center(child: CircularProgressIndicator())
                          else if (_scheduledMediaLoadFailed)
                            Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(l10n.scheduledPostsLoadError),
                                TextButton(
                                  onPressed: _hydrateScheduledMedia,
                                  child: Text(l10n.scheduledPostsRetryAction),
                                ),
                              ],
                            )
                          else
                            ComposerImageAttachmentSection(
                              imagesState: imagesState,
                              enabled: !createState.isLoading,
                              onAddImages: () =>
                                  ref.read(imagesProvider.notifier).addImages(),
                              onAltTextChanged: (imageId, value) => ref
                                  .read(imagesProvider.notifier)
                                  .setAltText(imageId, value),
                              onRemove: (imageId) => ref
                                  .read(imagesProvider.notifier)
                                  .remove(imageId),
                              onReplaceUnavailable: (imageId) => ref
                                  .read(imagesProvider.notifier)
                                  .replaceUnavailable(imageId),
                              onReorder: (fromIndex, toIndex) => ref
                                  .read(imagesProvider.notifier)
                                  .reorder(
                                    fromIndex: fromIndex,
                                    toIndex: toIndex,
                                  ),
                            ),
                        ],
                        SizedBox(
                          key: const Key('post-composer-bottom-safe-space'),
                          height:
                              spacing.sp9 +
                              MediaQuery.paddingOf(context).bottom,
                        ),
                      ],
                    ),
                  ),
                ),
                PositionedDirectional(
                  start: spacing.sp4,
                  end: spacing.sp4,
                  bottom: 0,
                  child: SafeArea(
                    top: false,
                    minimum: EdgeInsets.only(bottom: spacing.sp4),
                    child: _PostAction(
                      actionKey: const Key('post-composer-primary-action'),
                      isSaving: createState.isLoading || _isScheduling,
                      label: submitLabel,
                      onPressed: canSubmit
                          ? () => _submitPost(trimmedText: trimmedText)
                          : null,
                    ),
                  ),
                ),
              ],
            ),
          ),
          if (_isSubmitting)
            SubmissionBlockingOverlay(
              scheduling: _scheduleChoice == ScheduleChoice.later,
            ),
        ],
      ),
    );
  }

  bool _hasDraft(ComposerImagesState imagesState) {
    final seed = widget.draftSeed;
    final baselineMedia = seed?.draft.media ?? const [];
    final mediaChanged =
        imagesState.images.length != baselineMedia.length ||
        [
              for (final image in imagesState.images)
                '${image.id}:${image.altText}:${switch (image.phase) {
                  ImageReady(:final sha256) => sha256,
                  _ => 'unavailable',
                }}',
            ].join('|') !=
            [
              for (final media in baselineMedia)
                '${media.mediaId}:${media.altText}:${media.sha256}',
            ].join('|');
    return _text != _initialText ||
        !listEquals(_languages?.values, _initialLanguages) ||
        _scheduleChoice != _initialScheduleChoice ||
        _scheduledAtLocal != _initialScheduledAtLocal ||
        mediaChanged;
  }

  Future<void> _saveDraft(ComposerImagesState imagesState) async {
    final active = ref.read(sessionRegistryProvider).value?.activeLease;
    if (active == null ||
        (widget.draftOwner != null && active != widget.draftOwner)) {
      return;
    }
    setState(() => _isSavingDraft = true);
    try {
      final request = _draftWriteRequest(active.session.account, imagesState);
      final saved = await ref
          .read(draftSaveControllerProvider(active.session.account).notifier)
          .save(request);
      if (!mounted || saved == null) return;
      Navigator.of(context).pop();
      context.showInfo(AppLocalizations.of(context).draftSavedMessage);
    } on Object {
      if (mounted) {
        context.showError(AppLocalizations.of(context).draftSaveError);
      }
    } finally {
      if (mounted) setState(() => _isSavingDraft = false);
    }
  }

  DraftWriteRequest _draftWriteRequest(
    AccountKey owner,
    ComposerImagesState imagesState,
  ) {
    final existing = _origin.draft;
    final schedule = _scheduleChoice == ScheduleChoice.later
        ? DraftScheduleIntent.later(
            scheduledAtUtc: _scheduledAtLocal?.toUtc(),
            savedOffsetMinutes: _scheduledAtLocal?.timeZoneOffset.inMinutes,
          )
        : const DraftScheduleIntent.now();
    return const StandardDraftSnapshotAdapter().toWriteRequest(
      id: existing?.id ?? _composerId,
      owner: owner,
      text: _text,
      languages: _languages!.values,
      schedule: schedule,
      images: imagesState.images,
      existingRevision: existing?.revision,
      existingCreatedAt: existing?.createdAt,
    );
  }

  Future<bool> _confirmDiscard() {
    final l10n = AppLocalizations.of(context);
    return showCraftskyConfirmDialog(
      context,
      title: l10n.postComposeDiscardTitle,
      message: l10n.postComposeDiscardMessage,
      confirmLabel: l10n.postComposeDiscardConfirm,
      cancelLabel: l10n.postComposeDiscardCancel,
    );
  }

  bool _canSaveDraft(
    ComposerImagesState imagesState, {
    bool? hasDraft,
  }) =>
      widget.replyTarget == null &&
      widget.quoteTarget == null &&
      widget.scheduledPost == null &&
      (hasDraft ?? _hasDraft(imagesState)) &&
      imagesState.canSaveDraftMedia() &&
      !_isSavingDraft &&
      !_isSubmitting;

  Future<bool> _confirmClose(ComposerImagesState imagesState) async {
    final eligible =
        widget.replyTarget == null &&
        widget.quoteTarget == null &&
        widget.scheduledPost == null;
    if (!eligible) return _confirmDiscard();
    final choice = await showDraftCloseDialog(
      context,
      existingDraft: widget.draftSeed != null,
      canSave: _canSaveDraft(imagesState),
    );
    switch (choice) {
      case DraftCloseChoice.save:
        await _saveDraft(imagesState);
        return false;
      case DraftCloseChoice.discard:
        return true;
      case DraftCloseChoice.keepEditing:
        return false;
    }
  }

  void _ensureUnsavedWorkRegistration() {
    final owner = ref.read(sessionRegistryProvider).value?.activeLease?.session;
    if (owner == null || owner == _unsavedOwner) return;
    _unsavedOwner = owner;
    _unsavedRegistration = _unsavedGuard.replace(
      _unsavedRegistration,
      owner: owner,
      isDirty: () =>
          mounted && _hasDraft(ref.read(composerImagesProvider(_composerId))),
      confirmAndClose: () async {
        if (!mounted) return true;
        final imagesState = ref.read(composerImagesProvider(_composerId));
        final shouldClose = await _confirmClose(imagesState);
        if (!shouldClose || !mounted) return false;
        Navigator.of(context).pop();
        await Future<void>.delayed(Duration.zero);
        return true;
      },
    );
  }

  Future<void> _submitPost({required String trimmedText}) async {
    final imagesState = ref.read(composerImagesProvider(_composerId));
    if (widget.replyTarget == null && imagesState.hasImagesMissingAltText) {
      final l10n = AppLocalizations.of(context);
      final shouldPost = await showCraftskyConfirmDialog(
        context,
        title: l10n.postComposeMissingAltTitle,
        message: l10n.postComposeMissingAltMessage,
        confirmLabel: l10n.postComposeMissingAltConfirm,
        cancelLabel: l10n.postComposeMissingAltCancel,
      );
      if (!shouldPost || !mounted) return;
    }

    final facets = await _facetsForSubmission(trimmedText);
    if (!mounted) return;
    final submissionOwner = ref
        .read(sessionRegistryProvider)
        .value
        ?.activeLease;
    final needsUpload = imagesState.images.any(
      (image) => image.phase is ImageReady,
    );
    final uploadClient = needsUpload ? ref.read(postApiClientProvider) : null;

    await _runSubmission(submissionOwner, () async {
      if (widget.scheduledPost != null &&
          _scheduleChoice == ScheduleChoice.now) {
        await _publishExistingNow(
          trimmedText: trimmedText,
          facets: facets,
          imagesState: imagesState,
        );
        return;
      }

      if (_scheduleChoice == ScheduleChoice.later) {
        await _schedulePost(
          trimmedText: trimmedText,
          facets: facets,
          imagesState: imagesState,
        );
        return;
      }

      final images = widget.replyTarget == null
          ? await _mediaUploader.materializeImmediate(
              composerId: _composerId,
              images: imagesState.images,
              ownershipIsCurrent: () =>
                  _submissionOwnershipIsCurrent(submissionOwner),
              upload:
                  ({required bytes, required mimeType, required cancelToken}) =>
                      uploadClient!.uploadImage(
                        bytes: bytes,
                        mimeType: mimeType,
                        cancelToken: cancelToken,
                      ),
            )
          : null;
      final created = await ref
          .read(createPostProvider.notifier)
          .create(
            text: trimmedText,
            langs: _languages!.values,
            reply: _replyFor(widget.replyTarget),
            quote: _quoteFor(widget.quoteTarget),
            images: images,
            facets: facets.isEmpty ? null : facets,
            ownership: submissionOwner,
          );
      _submissionSucceeded = created != null;
    });
  }

  Future<void> _runSubmission(
    ActiveAccountLease? submissionOwner,
    Future<void> Function() operation,
  ) async {
    if (_submissionCoordinator.isRunning) return;
    _submissionSucceeded = false;
    await _submissionCoordinator.run(
      presentOverlay: () async {
        await WidgetsBinding.instance.endOfFrame;
        if (!mounted) throw StateError('composer disposed');
      },
      ownershipIsCurrent: () => _submissionOwnershipIsCurrent(submissionOwner),
      saveOriginSnapshot: _saveOriginSnapshot,
      operation: operation,
      didSucceed: () => _submissionSucceeded,
      deleteOriginAfterSuccess: _deleteOriginDraftAfterSuccess,
      onRunningChanged: ({required running}) {
        if (mounted) setState(() => _isSubmitting = running);
      },
      onFailure: (_) {
        if (mounted) {
          context.showError(AppLocalizations.of(context).postCreateError);
        }
      },
    );
  }

  bool _submissionOwnershipIsCurrent(ActiveAccountLease? owner) =>
      owner == null ||
      (ref.read(sessionRegistryProvider).value?.isCurrent(owner) ?? false);

  Future<void> _saveOriginSnapshot() async {
    final origin = _origin.draft;
    if (origin == null) return;
    final active = ref.read(sessionRegistryProvider).value?.activeLease;
    if (active == null ||
        active != widget.draftOwner ||
        active.session.account != origin.owner) {
      throw StateError('local-draft account changed');
    }
    final saved = await ref
        .read(draftSaveControllerProvider(origin.owner).notifier)
        .save(
          _draftWriteRequest(
            origin.owner,
            ref.read(composerImagesProvider(_composerId)),
          ),
        );
    if (saved == null) throw StateError('local-draft account changed');
    _origin.acceptSnapshot(saved);
  }

  Future<void> _deleteOriginDraftAfterSuccess() async {
    final origin = _origin.draft;
    if (origin == null) return;
    try {
      final repository = await ref.read(
        accountLocalPostDraftRepositoryProvider(origin.owner).future,
      );
      await repository.delete(origin.id);
      if (mounted) {
        await ref
            .read(localPostDraftsProvider(origin.owner).notifier)
            .refresh();
      }
    } on Object {
      if (mounted) {
        context.showError(AppLocalizations.of(context).draftCleanupError);
      }
    }
  }

  Future<List<Map<String, dynamic>>> _facetsForSubmission(String text) async {
    final stored = widget.scheduledPost?.payload['facets'];
    if (widget.scheduledPost != null &&
        text == _initialText.trim() &&
        stored is List<dynamic>) {
      return stored
          .whereType<Map<dynamic, dynamic>>()
          .map(
            (facet) => facet.map(
              (key, value) => MapEntry(key.toString(), value),
            ),
          )
          .toList(growable: false);
    }
    return ref.read(facetGeneratorProvider).generate(text);
  }

  String _whenLabel(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final scheduledAt = _scheduledAtLocal;
    if (_scheduleChoice == ScheduleChoice.now || scheduledAt == null) {
      return l10n.scheduledPostNow;
    }
    return _localTimeLabel(context, scheduledAt);
  }

  String _localTimeLabel(BuildContext context, DateTime value) {
    final l10n = AppLocalizations.of(context);
    final localizations = MaterialLocalizations.of(context);
    return l10n.scheduledPostLocalTime(
      localizations.formatMediumDate(value),
      localizations.formatTimeOfDay(TimeOfDay.fromDateTime(value)),
      value.timeZoneName,
      _offsetLabel(value.timeZoneOffset),
    );
  }

  Future<void> _chooseWhen(
    BuildContext menuContext, {
    required bool scheduleEnabled,
  }) async {
    final l10n = AppLocalizations.of(context);
    final choice = await showScheduleChoiceMenu(
      menuContext,
      selectedChoice: _scheduleChoice,
      scheduleEnabled: scheduleEnabled,
    );
    if (choice == null || !mounted) return;
    if (choice == ScheduleChoice.now) {
      setState(() {
        _scheduleChoice = choice;
        _scheduledAtLocal = null;
      });
      return;
    }
    var outOfRange = false;
    final selected = await pickScheduledLocalTime(
      context: context,
      now: DateTime.now(),
      onOutOfRange: (_) => outOfRange = true,
    );
    if (!mounted) return;
    if (outOfRange) {
      context.showError(l10n.scheduledPostTimeRangeError);
      return;
    }
    if (selected == null) return;
    setState(() {
      _scheduleChoice = ScheduleChoice.later;
      _scheduledAtLocal = selected;
    });
  }

  ActiveAccountLease? _captureScheduledOperationOwner() {
    final registry = ref.read(sessionRegistryProvider).value;
    final active = registry?.activeLease;
    final owner = widget.scheduledPost == null
        ? active
        : (_scheduledOwner ?? active);
    if (owner == null || registry?.isCurrent(owner) != true) return null;
    if (widget.scheduledPost != null) _scheduledOwner ??= owner;
    return owner;
  }

  bool _scheduledOperationIsCurrent(ActiveAccountLease owner) =>
      mounted &&
      (ref.read(sessionRegistryProvider).value?.isCurrent(owner) ?? false);

  Future<void> _schedulePost({
    required String trimmedText,
    required List<Map<String, dynamic>> facets,
    required ComposerImagesState imagesState,
  }) async {
    final owner = _captureScheduledOperationOwner();
    final scheduledAt = _scheduledAtLocal;
    if (owner == null || scheduledAt == null) return;
    final account = owner.session.account;
    setState(() {
      _isScheduling = true;
      _stagedImageCount = 0;
      _stagedImageTotal = imagesState.images
          .where((image) => image.phase is ImageUploaded)
          .length;
      _isSavingSchedule = _stagedImageTotal == 0;
    });
    try {
      final repository = await ref.read(
        accountScheduledPostRepositoryProvider(account).future,
      );
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      final media = await ref.read(scheduledComposerMediaMaterializerProvider)(
        imagesState.images,
        stageMedia: repository.stageMedia,
        onStaged: (_) {
          if (mounted && _scheduledOperationIsCurrent(owner)) {
            setState(() => _stagedImageCount++);
          }
        },
      );
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      final payload = <String, dynamic>{
        'kind': 'standard',
        'text': trimmedText,
        'langs': _languages!.values,
        'facets': ?facets.isEmpty ? null : facets,
        'media': media,
      };
      if (mounted) setState(() => _isSavingSchedule = true);
      final existing = widget.scheduledPost;
      if (existing == null) {
        await repository.create(
          operationId: _composerId,
          scheduledAt: scheduledAt.toUtc(),
          payload: payload,
        );
      } else {
        await repository.update(
          id: existing.id,
          scheduledAt: scheduledAt.toUtc(),
          payload: payload,
        );
      }
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      await ref.read(scheduledPostsProvider(account).notifier).refresh();
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      _submissionSucceeded = true;
      Navigator.of(context).pop();
      context.showInfo(AppLocalizations.of(context).scheduledPostSaved);
    } on Object {
      if (mounted && _scheduledOperationIsCurrent(owner)) {
        context.showError(
          AppLocalizations.of(context).scheduledPostSaveError,
        );
      }
    } finally {
      if (mounted) {
        setState(() {
          _isScheduling = false;
          _stagedImageCount = 0;
          _stagedImageTotal = 0;
          _isSavingSchedule = false;
        });
      }
    }
  }

  Future<void> _publishExistingNow({
    required String trimmedText,
    required List<Map<String, dynamic>> facets,
    required ComposerImagesState imagesState,
  }) async {
    final existing = widget.scheduledPost;
    final owner = _captureScheduledOperationOwner();
    if (existing == null || owner == null) return;
    final account = owner.session.account;
    setState(() {
      _isScheduling = true;
      _stagedImageCount = 0;
      _stagedImageTotal = imagesState.images
          .where((image) => image.phase is ImageUploaded)
          .length;
      _isSavingSchedule = _stagedImageTotal == 0;
    });
    try {
      final repository = await ref.read(
        accountScheduledPostRepositoryProvider(account).future,
      );
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      final media = await ref.read(scheduledComposerMediaMaterializerProvider)(
        imagesState.images,
        stageMedia: repository.stageMedia,
        onStaged: (_) {
          if (mounted && _scheduledOperationIsCurrent(owner)) {
            setState(() => _stagedImageCount++);
          }
        },
      );
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      if (mounted) setState(() => _isSavingSchedule = true);
      await repository.publishNow(
        id: existing.id,
        payload: {
          ...existing.payload,
          'text': trimmedText,
          'langs': _languages!.values,
          'facets': ?facets.isEmpty ? null : facets,
          'media': media,
        },
      );
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      await ref.read(scheduledPostsProvider(account).notifier).refresh();
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      _submissionSucceeded = true;
      Navigator.of(context).pop();
    } on Object {
      if (mounted && _scheduledOperationIsCurrent(owner)) {
        context.showError(AppLocalizations.of(context).scheduledPostNowError);
      }
    } finally {
      if (mounted) {
        setState(() {
          _isScheduling = false;
          _stagedImageCount = 0;
          _stagedImageTotal = 0;
          _isSavingSchedule = false;
        });
      }
    }
  }

  List<dynamic> get _scheduledPayloadMedia =>
      widget.scheduledPost?.payload['media'] as List<dynamic>? ?? const [];

  Future<void> _hydrateScheduledMedia() async {
    if (_scheduledPayloadMedia.isEmpty) return;
    setState(() {
      _isLoadingScheduledMedia = true;
      _scheduledMediaLoadFailed = false;
    });
    try {
      var owner = _captureScheduledOperationOwner();
      if (owner == null) {
        await ref.read(sessionRegistryProvider.future);
        if (!mounted) return;
        owner = _captureScheduledOperationOwner();
      }
      if (owner == null) throw StateError('account unavailable');
      final account = owner.session.account;
      final repository = await ref.read(
        accountScheduledPostRepositoryProvider(account).future,
      );
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      final images = await hydrateScheduledComposerMedia(
        _scheduledPayloadMedia,
        loadBytes: repository.mediaBytes,
      );
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      ref
          .read(composerImagesProvider(_composerId).notifier)
          .seedScheduledImages(images);
      setState(() => _isLoadingScheduledMedia = false);
    } on Object {
      if (!mounted) return;
      setState(() {
        _isLoadingScheduledMedia = false;
        _scheduledMediaLoadFailed = true;
      });
    }
  }

  Future<void> _deleteExistingSchedule() async {
    final existing = widget.scheduledPost;
    final owner = _captureScheduledOperationOwner();
    if (existing == null || owner == null) return;
    final account = owner.session.account;
    final l10n = AppLocalizations.of(context);
    final confirmed = await showCraftskyConfirmDialog(
      context,
      title: l10n.scheduledPostsDeleteTitle,
      message: l10n.scheduledPostsDeleteMessage,
      confirmLabel: l10n.scheduledPostsDeleteAction,
      cancelLabel: l10n.languageCancel,
    );
    if (!confirmed || !mounted || !_scheduledOperationIsCurrent(owner)) return;
    setState(() => _isScheduling = true);
    try {
      final repository = await ref.read(
        accountScheduledPostRepositoryProvider(account).future,
      );
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      await repository.delete(existing.id);
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      await ref.read(scheduledPostsProvider(account).notifier).refresh();
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      Navigator.of(context).pop();
    } on Object {
      if (mounted && _scheduledOperationIsCurrent(owner)) {
        context.showError(l10n.scheduledPostDeleteError);
      }
    } finally {
      if (mounted) setState(() => _isScheduling = false);
    }
  }
}

String _offsetLabel(Duration offset) {
  final minutes = offset.inMinutes.abs();
  final sign = offset.isNegative ? '-' : '+';
  return '$sign${(minutes ~/ 60).toString().padLeft(2, '0')}:'
      '${(minutes % 60).toString().padLeft(2, '0')}';
}

PostRef? _quoteFor(Post? target) {
  if (target == null) return null;
  return PostRef(uri: target.uri, cid: target.cid);
}

class _ComposerTargetPreview extends StatelessWidget {
  const _ComposerTargetPreview({required this.post});

  final Post post;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final displayName = post.author.displayName;
    return DecoratedBox(
      decoration: BoxDecoration(
        color: swatches.paper2,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: swatches.borderHair),
      ),
      child: Padding(
        padding: EdgeInsets.all(spacing.sp3),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (displayName != null && displayName.trim().isNotEmpty)
              Text(displayName, style: theme.textTheme.titleSmall),
            Text(
              '@${post.author.handle}',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
            ),
            SizedBox(height: spacing.sp2),
            Text(
              post.text,
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
              style: theme.textTheme.bodyMedium,
            ),
          ],
        ),
      ),
    );
  }
}

class _PostAction extends StatelessWidget {
  const _PostAction({
    required this.actionKey,
    required this.isSaving,
    required this.label,
    required this.onPressed,
  });

  final Key actionKey;
  final bool isSaving;
  final String label;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    return ChunkyButton(
      key: actionKey,
      onPressed: onPressed,
      child: isSaving ? const StitchProgressIndicator(size: 18) : Text(label),
    );
  }
}

PostReply? _replyFor(Post? target) {
  if (target == null) return null;

  return PostReply(
    root: target.reply?.root ?? PostRef(uri: target.uri, cid: target.cid),
    parent: PostRef(uri: target.uri, cid: target.cid),
  );
}
