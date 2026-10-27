import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/drafts/composer/composer_draft_session.dart';
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/composer/draft_schedule_restoration.dart';
import 'package:craftsky_app/drafts/composer/standard_draft_snapshot_adapter.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/draft_save_controller.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/drafts/providers/local_post_drafts_provider.dart';
import 'package:craftsky_app/drafts/widgets/draft_close_dialog.dart';
import 'package:craftsky_app/feed/composer/composer_media_uploader.dart';
import 'package:craftsky_app/feed/composer/composer_submission_coordinator.dart';
import 'package:craftsky_app/feed/composer/composer_video_publication.dart';
import 'package:craftsky_app/feed/composer/link_preview_candidate.dart';
import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/composer/post_edit_hydrator.dart';
import 'package:craftsky_app/feed/composer/post_edit_origin.dart';
import 'package:craftsky_app/feed/composer/post_edit_validation_message.dart';
import 'package:craftsky_app/feed/composer/published_post_embed_edit.dart';
import 'package:craftsky_app/feed/composer/submission_screen_awake.dart';
import 'package:craftsky_app/feed/composer/video_failure_message.dart';
import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/feed/providers/edit_post_provider.dart';
import 'package:craftsky_app/feed/providers/image_picker_existing_video.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/feed/providers/video_service_client_provider.dart';
import 'package:craftsky_app/feed/providers/video_upload_feature_provider.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/feed/widgets/composer_link_preview_carousel.dart';
import 'package:craftsky_app/feed/widgets/composer_metadata_controls.dart';
import 'package:craftsky_app/feed/widgets/composer_video_attachment_card.dart';
import 'package:craftsky_app/feed/widgets/published_external_edit_card.dart';
import 'package:craftsky_app/feed/widgets/published_post_edit_context.dart';
import 'package:craftsky_app/feed/widgets/published_post_editor.dart';
import 'package:craftsky_app/feed/widgets/submission_blocking_overlay.dart';
import 'package:craftsky_app/keys.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/post_language_selection.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/profile/models/profile_handle.dart';
import 'package:craftsky_app/profile/widgets/compact_profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/scheduled_posts/composer/schedule_capacity_state.dart';
import 'package:craftsky_app/scheduled_posts/composer/schedule_composer_state.dart';
import 'package:craftsky_app/scheduled_posts/composer/scheduled_composer_submission.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_post_repository_provider.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_posts_provider.dart';
import 'package:craftsky_app/scheduled_posts/services/scheduled_composer_media.dart';
import 'package:craftsky_app/scheduled_posts/widgets/schedule_choice_menu.dart';
import 'package:craftsky_app/scheduled_posts/widgets/schedule_time_picker.dart';
import 'package:craftsky_app/scheduled_posts/widgets/scheduled_post_capacity_warning.dart';
import 'package:craftsky_app/scheduled_posts/widgets/scheduled_staging_progress.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/shared/rich_text/widgets/facet_autocomplete_editor.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/shared/widgets/composer_bottom_action.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/publishing_progress_indicator.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:logging/logging.dart';
import 'package:material_ui/material_ui.dart';
import 'package:uuid/uuid.dart';

// TODO(Agent): Does this need to be a provider?
final linkPreviewUndoDurationProvider = Provider<Duration>(
  (_) => const Duration(seconds: 4),
);

Future<Post?> showPostComposerSheet(
  BuildContext context, {
  Post? replyTarget,
  Post? quoteTarget,
  ScheduledPostDetail? scheduledPost,
  ActiveAccountLease? scheduledOwner,
  LocalPostDraftSeed? draftSeed,
  ActiveAccountLease? draftOwner,
  PostEditOrigin? publishedEdit,
}) {
  return responsiveModalNavigator(context).push<Post?>(
    FullscreenModalRoute<Post?>(
      builder: (_) => PostComposerSheet(
        replyTarget: replyTarget,
        quoteTarget: quoteTarget,
        scheduledPost: scheduledPost,
        scheduledOwner: scheduledOwner,
        draftSeed: draftSeed,
        draftOwner: draftOwner,
        publishedEdit: publishedEdit,
      ),
    ),
  );
}

class PostComposerSheet extends ConsumerStatefulWidget {
  const new({
    super.key,
    this.replyTarget,
    this.quoteTarget,
    this.composerId,
    this.scheduledPost,
    this.scheduledOwner,
    this.draftSeed,
    this.draftOwner,
    this.publishedEdit,
    this.videoController,
    this.prepareVideoProof,
  }) : assert(
         publishedEdit == null ||
             (replyTarget == null &&
                 quoteTarget == null &&
                 scheduledPost == null &&
                 draftSeed == null),
         'Published edits have their own fixed origin',
       ),
       assert(
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
  final PostEditOrigin? publishedEdit;
  final ActiveAccountLease? draftOwner;
  final ComposerVideoController? videoController;
  final Future<CreatePostVideo> Function(LocalVideoSelection video)?
  prepareVideoProof;

  @override
  ConsumerState<PostComposerSheet> createState() => _PostComposerSheetState();
}

class _PostComposerSheetState extends ConsumerState<PostComposerSheet>
    with WidgetsBindingObserver {
  final _controller = FacetTextEditingController();
  final _focusNode = FocusNode(debugLabel: 'postComposerText');
  late final String _composerId;
  late final String? _existingExternalMediaId;
  String? _replacementExternalMediaId;
  List<int>? _replacementExternalBytes;
  String? _replacementExternalMimeType;
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
  late final ComposerVideoController _videoController;
  VideoPublicationCoordinator? _videoPublication;
  ActiveAccountLease? _videoPublicationOwner;
  VideoPublicationProgress? _videoProgress;
  Object? _videoFailure;
  var _usesStoredDraftVideo = false;
  var _isSubmitting = false;
  var _isSavingDraft = false;
  var _isDeletingSchedule = false;
  var _attemptedSubmit = false;
  var _sponsored = false;
  var _initialSponsored = false;
  var _submissionSucceeded = false;
  Post? _pendingPublishedDraftPost;
  late final ComposerDraftSession _origin;
  List<ComposerImageDraft> _publishedImages = const [];
  bool _isRefreshingExternal = false;
  bool _hasExternalReplacement = false;
  LinkPreviewThumbnail? _replacementExternalThumbnail;
  CancelToken? _externalRefreshCancel;
  late final PublishedPostEmbedEdit? _publishedEmbed;
  Future<void> Function()? _retryPublishedEdit;
  PostEditResult? _acceptedEditPending;
  List<String>? _initialLanguages;
  ScheduleChoice _initialScheduleChoice = ScheduleChoice.now;
  DateTime? _initialScheduledAtLocal;
  String? _lastPreviewSyncedText;
  AccountKey? _lastPreviewAccount;
  var _didSeedScheduledPreview = false;
  LinkPreviewThumbnail? _scheduledExternalThumbnail;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _unsavedGuard = ref.read(unsavedWorkGuardProvider);
    _composerId = widget.composerId ?? const Uuid().v4();
    _existingExternalMediaId = _existingExternalThumbMediaId(
      widget.scheduledPost,
    );
    _origin = ComposerDraftSession(widget.draftSeed?.draft);
    _publishedEmbed = widget.publishedEdit == null
        ? null
        : PublishedPostEmbedEdit(
            (widget.publishedEdit!.seed.editable['embed'] as Map?)
                ?.cast<String, Object?>(),
          );
    _mediaUploader = ComposerMediaUploader();
    _submissionCoordinator = ComposerSubmissionCoordinator(
      screenAwake: const WakelockSubmissionScreenAwake(),
    );
    _videoController =
        widget.videoController ??
        ComposerVideoController(
          picker: ImagePickerExistingVideo(),
          checkEligibility: () =>
              ref.read(postApiClientProvider).getVideoUploadLimits(),
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
        _sponsored = content.sponsored;
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
          unawaited(_restoreDraftVideo(seed));
        }
      });
    }
    if (widget.publishedEdit case final edit?) {
      _text = edit.seed.editable['text']! as String;
      _controller.text = _text;
      _sponsored = edit.seed.editable['sponsored']! as bool;
      final languages = (edit.seed.editable['langs'] as List? ?? const [])
          .cast<String>();
      _languages = PostLanguageSelection.forPublishedEdit(languages);
      _initialLanguages = List.of(languages);
      _publishedImages = hydratePublishedPostImages(edit.seed);
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          ref
              .read(composerImagesProvider(_composerId).notifier)
              .seedPublishedImages(_publishedImages);
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
      _sponsored = scheduled.payload['sponsored'] == true;
      if (scheduled.status == ScheduledPostStatus.needsAttention) {
        _scheduleChoice = ScheduleChoice.now;
        _missedScheduledAtLocal = scheduled.scheduledAt.utc.toLocal();
      } else {
        _scheduleChoice = ScheduleChoice.later;
        _scheduledAtLocal = scheduled.scheduledAt.utc.toLocal();
      }
      if (_scheduledPayloadMedia.isNotEmpty ||
          _existingExternalMediaId != null) {
        _isLoadingScheduledMedia = true;
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted) unawaited(_hydrateScheduledMedia());
        });
      }
    }
    if (widget.replyTarget?.reply != null) {
      final alias = ProfileHandle(widget.replyTarget!.author.handle).aliasInput;
      _text = alias == null ? '' : '@$alias ';
      _controller.text = _text;
      _controller.selection = TextSelection.collapsed(offset: _text.length);
    }
    _initialText = _text;
    _initialSponsored = _sponsored;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      _focusNode.requestFocus();
    });
  }

  @override
  void dispose() {
    _externalRefreshCancel?.cancel('editor disposed');
    WidgetsBinding.instance.removeObserver(this);
    _unsavedGuard.unregister(_unsavedRegistration);
    unawaited(_submissionCoordinator.dispose());
    _videoPublication?.cancel();
    _controller.dispose();
    _focusNode.dispose();
    _mediaUploader.disposeComposer(_composerId);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.resumed &&
        shouldCancelVideoPublicationOnLifecycleInterruption(
          _videoProgress?.stage,
        )) {
      _videoPublication?.cancel();
    }
  }

  @override
  Widget build(BuildContext context) {
    ref.watch(publishingAnimationFileProvider);
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final keyboardVisible = MediaQuery.viewInsetsOf(context).bottom > 0;
    final createState = ref.watch(createPostProvider);
    final editState = widget.publishedEdit == null
        ? null
        : ref.watch(editPostProvider);
    final controlsBusy =
        _isSubmitting ||
        _acceptedEditPending != null ||
        _isRefreshingExternal ||
        editState?.error is PdsMutationUnresolvedException;
    final editUnresolved = editState?.error is PdsMutationUnresolvedException;
    final editUnavailable = switch (editState?.error) {
      ApiBadRequest(code: 'post_not_found') => true,
      _ => false,
    };
    final videoUploadsEnabled = ref.watch(videoUploadsEnabledProvider);
    final preferences = ref.watch(activeLanguagePreferencesProvider);
    _languages ??= PostLanguageSelection.fromPrimary(
      preferences.primaryLanguage,
    );
    _initialLanguages ??= List.of(_languages!.values);
    final imagesProvider = composerImagesProvider(_composerId);
    final imagesState = ref.watch(imagesProvider);
    final selectedVideo = _videoController.selection;
    final isResponse =
        widget.replyTarget != null ||
        widget.publishedEdit?.seed.post.reply != null;
    final isComment = widget.replyTarget?.reply == null && isResponse;
    final isQuote =
        widget.quoteTarget != null ||
        widget.publishedEdit?.seed.post.quote != null;
    final isSchedulable =
        !isResponse && !isQuote && widget.publishedEdit == null;
    final registry = ref.watch(sessionRegistryProvider);
    final activeLease = registry.value?.activeLease;
    if (widget.publishedEdit case final edit?) {
      if (!registry.isLoading && activeLease != edit.ownership) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted && Navigator.of(context).canPop()) {
            Navigator.of(context).pop();
          }
        });
        return const SizedBox.shrink();
      }
    }
    if (_videoPublicationOwner != null &&
        activeLease != _videoPublicationOwner) {
      _videoPublication?.cancel();
    }
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
    final scheduleAccess = activeLease == null
        ? null
        : ref.watch(subscriptionAccessProvider(activeLease.session));
    final maySchedule = switch (scheduleAccess) {
      AsyncData(:final value) => value.allowsPlus,
      _ => false,
    };
    final previewProvider = account == null
        ? null
        : linkPreviewControllerProvider(_composerId, account);
    final previewState = previewProvider == null
        ? null
        : ref.watch(previewProvider);
    final previewController = previewProvider == null
        ? null
        : ref.read(previewProvider.notifier);
    final publishedEmbedIsFixed =
        _publishedEmbed?.hasExternal == true ||
        _publishedEmbed?.hasRetainedVideo == true ||
        widget.publishedEdit?.seed.editable['unsupportedEmbed'] == true;
    final selectedPreview = previewController?.selected;
    final availablePreviews = previewController?.available ?? const [];
    if (previewController != null) {
      final shouldSyncText =
          _lastPreviewSyncedText != _text || _lastPreviewAccount != account;
      if (shouldSyncText) {
        _lastPreviewSyncedText = _text;
        _lastPreviewAccount = account;
      }
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        if (!_didSeedScheduledPreview) {
          _didSeedScheduledPreview = true;
          final frozen = _scheduledExternalSelection(
            widget.scheduledPost,
            thumbnail: _scheduledExternalThumbnail,
          );
          if (frozen != null) previewController.seed(frozen);
        }
        if (shouldSyncText) {
          previewController.updateText(
            _previewText(_text),
          );
        }
        previewController.setSuppressed(
          value: imagesState.images.isNotEmpty || isQuote,
        );
      });
    }
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
    final tooLong = !RecordTextLimits.post.accepts(_text);
    final canSubmit =
        !editUnavailable &&
        (widget.publishedEdit == null ||
            activeLease == widget.publishedEdit!.ownership) &&
        !(editState?.isLoading ?? false) &&
        _acceptedEditPending == null &&
        !_isSavingDraft &&
        !createState.isLoading &&
        !controlsBusy &&
        !_isScheduling &&
        !_isLoadingScheduledMedia &&
        !_scheduledMediaLoadFailed &&
        _languages != null &&
        imagesState.isSubmitReady &&
        (selectedVideo == null || _scheduleChoice == ScheduleChoice.now) &&
        (_scheduleChoice == ScheduleChoice.now ||
            (capacity.scheduleEnabled && maySchedule));
    final bodyErrorText = switch ((_attemptedSubmit, trimmedText.isEmpty)) {
      (true, true) => l10n.postComposeBodyRequiredError,
      _ when tooLong => l10n.postComposeTooLong,
      _ => null,
    };
    final submitLabel = _acceptedEditPending != null
        ? l10n.postEditRefresh
        : editUnresolved
        ? l10n.postEditRetrySave
        : widget.publishedEdit != null
        ? l10n.postEditSaveChanges
        : _scheduleChoice == ScheduleChoice.later
        ? l10n.scheduledPostAction
        : isComment
        ? l10n.postComposeCommentSubmit
        : isResponse
        ? l10n.postComposeReplySubmit
        : l10n.postComposeSubmit;
    final hasDraft = _hasDraft(imagesState);
    final canSaveDraft = _canSaveDraft(imagesState, hasDraft: hasDraft);
    _ensureUnsavedWorkRegistration();

    ref
      ..listen(createPostProvider, (previous, next) {
        switch ((previous, next)) {
          case (AsyncLoading(), AsyncData(:final value?)):
            if (_origin.draft != null) {
              _pendingPublishedDraftPost = value;
            } else if (Navigator.of(context).canPop()) {
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
      canPop:
          !_isSavingDraft &&
          !_isDeletingSchedule &&
          !_isSubmitting &&
          (_acceptedEditPending != null ||
              _pendingPublishedDraftPost != null ||
              !hasDraft ||
              createState.isLoading),
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        if (_isSubmitting || _isSavingDraft || _isDeletingSchedule) return;
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
                isComment
                    ? l10n.postComposeCommentTitle
                    : isResponse
                    ? l10n.postComposeReplyTitle
                    : isQuote
                    ? l10n.postQuoteAction
                    : l10n.postComposeTitle,
                style: theme.textTheme.titleLarge,
              ),
              actions: [
                if (widget.replyTarget == null &&
                    widget.quoteTarget == null &&
                    widget.scheduledPost == null &&
                    widget.publishedEdit == null)
                  TextButton(
                    onPressed: canSaveDraft
                        ? () => _saveDraft(imagesState)
                        : null,
                    child: ActionProgress(
                      isLoading: _isSavingDraft,
                      child: Text(
                        widget.draftSeed == null
                            ? l10n.draftSaveAction
                            : l10n.draftSaveChangesAction,
                      ),
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
                        if (widget.publishedEdit case final edit?)
                          PublishedPostEditContext(post: edit.seed.post),
                        if (widget.replyTarget case final replyTarget?) ...[
                          _ComposerTargetPreview(post: replyTarget),
                          SizedBox(height: spacing.sp4),
                        ],
                        if (widget.quoteTarget case final quoteTarget?) ...[
                          _ComposerTargetPreview(post: quoteTarget),
                          SizedBox(height: spacing.sp4),
                        ],
                        FacetAutocompleteEditor(
                          key: keys.feed.composerBody,
                          maxLength: RecordTextLimits.post.graphemes,
                          maxBytes: RecordTextLimits.post.bytes,
                          label: isComment
                              ? l10n.postComposeCommentHint
                              : isResponse
                              ? l10n.postComposeReplyHint
                              : l10n.postComposeHint,
                          hintText: isResponse
                              ? null
                              : l10n.postComposeBodyHint,
                          controller: _controller,
                          focusNode: _focusNode,
                          required: true,
                          requiredLabel: l10n.postComposeRequiredLabel,
                          minLines: isResponse ? 5 : 3,
                          maxLines: 12,
                          textInputAction: TextInputAction.newline,
                          keyboardType: TextInputType.multiline,
                          enabled:
                              !createState.isLoading &&
                              !_isSavingDraft &&
                              !_isScheduling &&
                              !controlsBusy,
                          errorText: bodyErrorText,
                          helperAlignment: AlignmentDirectional.centerEnd,
                          onChanged: (value) {
                            setState(() => _text = value);
                            _lastPreviewSyncedText = value;
                            _lastPreviewAccount = account;
                            previewController?.updateText(_previewText(value));
                          },
                        ),
                        ComposerMetadataControls(
                          languages: _languages!,
                          onLanguagesChanged:
                              createState.isLoading ||
                                  _isSavingDraft ||
                                  _isScheduling ||
                                  controlsBusy
                              ? null
                              : (value) => setState(() => _languages = value),
                          sponsored: _sponsored,
                          onSponsoredChanged:
                              !createState.isLoading &&
                                  !_isSavingDraft &&
                                  !controlsBusy &&
                                  !_isScheduling
                              ? (value) => setState(() => _sponsored = value)
                              : null,
                          showSponsored: !isResponse,
                          scheduledAtLocal: _scheduledAtLocal,
                          showSchedule: isSchedulable,
                          scheduleAccess: scheduleAccess,
                          onScheduleAccessRetry: activeLease == null
                              ? null
                              : () => ref.invalidate(
                                  subscriptionAccessProvider(
                                    activeLease.session,
                                  ),
                                ),
                          onSchedulePressed:
                              isSchedulable &&
                                  !_isScheduling &&
                                  !_isSavingDraft &&
                                  !controlsBusy
                              ? (menuContext) => _chooseWhen(
                                  menuContext,
                                  scheduleEnabled: capacity.scheduleEnabled,
                                )
                              : null,
                        ),
                        if (isSchedulable) ...[
                          if (capacity.showCapacityWarning)
                            const ScheduledPostCapacityWarning(),
                          if (capacity.showManageLink)
                            Align(
                              alignment: AlignmentDirectional.centerStart,
                              child: TextButton(
                                onPressed: () =>
                                    const ScheduledPostsRoute().go(context),
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
                                onPressed:
                                    _isScheduling ||
                                        _isSavingDraft ||
                                        controlsBusy
                                    ? null
                                    : _deleteExistingSchedule,
                                icon: ActionProgress(
                                  isLoading: _isDeletingSchedule,
                                  child: const Icon(CraftskyIconsBold.delete),
                                ),
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
                        if (_publishedEmbed?.hasExternal == true)
                          PublishedExternalEditCard(
                            edit: _publishedEmbed!,
                            enabled:
                                !controlsBusy &&
                                !_isSavingDraft &&
                                !_isScheduling,
                            thumbnailUrl: _hasExternalReplacement
                                ? null
                                : widget
                                      .publishedEdit
                                      ?.seed
                                      .post
                                      .external
                                      ?.thumb
                                      ?.url,
                            thumbnailBytes:
                                _replacementExternalThumbnail?.bytes,
                            onRefresh: _refreshPublishedExternal,
                            onRemove: () {
                              previewController?.dismiss();
                              setState(() {
                                _publishedEmbed.remove();
                                _replacementExternalThumbnail = null;
                              });
                            },
                            onRemoveThumbnail: () => setState(() {
                              _publishedEmbed.removeExternalThumbnail();
                              _replacementExternalThumbnail = null;
                            }),
                            onChanged: () => setState(() {}),
                          ),
                        if (!publishedEmbedIsFixed &&
                            selectedVideo == null &&
                            previewController != null &&
                            previewState != null &&
                            !previewState.dismissed &&
                            !previewState.suppressed) ...[
                          ComposerLinkPreviewCarousel(
                            selected: selectedPreview,
                            current: selectedPreview == null
                                ? 0
                                : availablePreviews.indexWhere(
                                        (item) =>
                                            item.candidate.identity ==
                                            selectedPreview.candidate.identity,
                                      ) +
                                      1,
                            total: availablePreviews.length,
                            loading: previewState.inFlightIdentity != null,
                            onPrevious: previewController.selectPrevious,
                            onNext: previewController.selectNext,
                            onDismiss: () => _dismissLinkPreviews(
                              previewController,
                              l10n,
                            ),
                          ),
                        ],
                        if (!isResponse || widget.publishedEdit != null) ...[
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
                          else if (_publishedEmbed?.hasRetainedVideo == true &&
                              selectedVideo == null)
                            ComposerVideoAttachmentCard.published(
                              altText: _publishedEmbed!.videoAlt,
                              enabled:
                                  !controlsBusy &&
                                  !_isSavingDraft &&
                                  !_isScheduling,
                              onAltTextChanged: (value) => setState(
                                () => _publishedEmbed.videoAlt = value,
                              ),
                              onReplace: () =>
                                  unawaited(_selectVideo(replace: true)),
                              onRemove: () =>
                                  setState(() => _publishedEmbed.remove()),
                            )
                          else if (selectedVideo != null)
                            ComposerVideoAttachmentCard(
                              selection: selectedVideo,
                              enabled:
                                  !createState.isLoading &&
                                  !controlsBusy &&
                                  !_isSavingDraft &&
                                  !_isScheduling,
                              onAltTextChanged: (value) {
                                setState(
                                  () => _videoController.setAltText(value),
                                );
                              },
                              onReplace: () =>
                                  unawaited(_selectVideo(replace: true)),
                              onRemove: () {
                                setState(() {
                                  _usesStoredDraftVideo = false;
                                  _videoController.remove();
                                });
                              },
                            )
                          else if (_publishedEmbed?.hasExternal != true)
                            ComposerImageAttachmentSection(
                              key: keys.composerImages.section('composer'),
                              useDetailsRoute: true,
                              imagesProvider: imagesProvider,
                              imagesState: imagesState,
                              enabled:
                                  !createState.isLoading &&
                                  !_isSavingDraft &&
                                  !_isScheduling &&
                                  !controlsBusy,
                              onAddImages: () =>
                                  ref.read(imagesProvider.notifier).addImages(),
                              onTakePhoto: () =>
                                  ref.read(imagesProvider.notifier).takePhoto(),
                              onAltTextChanged: (imageId, value) => ref
                                  .read(imagesProvider.notifier)
                                  .setAltText(imageId, value),
                              onRemove: (imageId) => ref
                                  .read(imagesProvider.notifier)
                                  .remove(imageId),
                              onReplaceUnavailable: (imageId, source) => ref
                                  .read(imagesProvider.notifier)
                                  .replaceUnavailable(imageId, source: source),
                              onReorder: (fromIndex, toIndex) => ref
                                  .read(imagesProvider.notifier)
                                  .reorder(
                                    fromIndex: fromIndex,
                                    toIndex: toIndex,
                                  ),
                              supportsVideo:
                                  videoUploadsEnabled &&
                                  !isResponse &&
                                  !isQuote &&
                                  widget
                                          .publishedEdit
                                          ?.seed
                                          .editable['unsupportedEmbed'] !=
                                      true &&
                                  _scheduleChoice == ScheduleChoice.now &&
                                  widget.scheduledPost == null,
                              onAddVideo: _selectVideo,
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
                    child: Padding(
                      padding: EdgeInsets.only(bottom: spacing.sp4),
                      child: ComposerBottomAction(
                        keyboardVisible: keyboardVisible,
                        doneKey: const Key('post-composer-done'),
                        actionKey: keys.feed.submit,
                        label: submitLabel,
                        onPressed: _acceptedEditPending != null
                            ? (!_isSubmitting &&
                                      activeLease ==
                                          widget.publishedEdit?.ownership
                                  ? _refreshPublishedAccepted
                                  : null)
                            : editUnresolved
                            ? (!_isSubmitting &&
                                      _submissionOwnershipIsCurrent(
                                        widget.publishedEdit?.ownership,
                                      )
                                  ? _retryPublishedEdit
                                  : null)
                            : canSubmit
                            ? () => _submitPost(trimmedText: trimmedText)
                            : null,
                        child:
                            createState.isLoading ||
                                (_isScheduling && !_isDeletingSchedule)
                            ? const StitchProgressIndicator(size: 18)
                            : null,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
          if (_isSubmitting)
            SubmissionBlockingOverlay(
              scheduling: _scheduleChoice == ScheduleChoice.later,
              kind: widget.publishedEdit != null
                  ? (_acceptedEditPending != null
                        ? SubmissionKind.refreshPost
                        : SubmissionKind.edit)
                  : isComment
                  ? SubmissionKind.comment
                  : isResponse
                  ? SubmissionKind.reply
                  : SubmissionKind.post,
              videoProgress: _videoProgress,
              onCancelVideo: _videoPublication?.cancel,
            ),
        ],
      ),
    );
  }

  String _previewText(String text) =>
      widget.publishedEdit != null &&
          (text == _initialText ||
              _publishedEmbed?.hasExternal == true ||
              _publishedEmbed?.hasRetainedVideo == true ||
              widget.publishedEdit!.seed.editable['unsupportedEmbed'] == true)
      ? ''
      : text;

  void _dismissLinkPreviews(
    LinkPreviewController controller,
    AppLocalizations l10n,
  ) {
    final dismissGeneration = controller.dismiss();
    final messenger = ScaffoldMessenger.of(context);
    final semantic = Theme.of(context).extension<SemanticColorsTheme>()!;
    unawaited(
      messenger
          .showSnackBar(
            SnackBar(
              behavior: SnackBarBehavior.floating,
              dismissDirection: DismissDirection.horizontal,
              backgroundColor: semantic.infoSurface,
              content: CraftskySnackBarContent(
                severity: MessageSeverity.info,
                message: l10n.linkPreviewHidden,
                action: MessageAction(
                  label: l10n.linkPreviewUndo,
                  onPressed: () {
                    controller.undoDismiss();
                    messenger.hideCurrentSnackBar();
                  },
                ),
              ),
              duration: ref.read(linkPreviewUndoDurationProvider),
              persist: false,
            ),
          )
          .closed
          .then((_) => controller.expireUndo(dismissGeneration)),
    );
  }

  bool _hasDraft(ComposerImagesState imagesState) {
    if (_acceptedEditPending != null) return false;
    if (widget.publishedEdit != null) {
      return (_publishedEmbed?.changes.isNotEmpty ?? false) ||
          _text != _initialText ||
          !listEquals(_languages?.values, _initialLanguages) ||
          _sponsored != _initialSponsored ||
          _publishedImagesChanged(imagesState) ||
          _videoController.selection != null;
    }
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
        _sponsored != _initialSponsored ||
        mediaChanged ||
        _videoController.selection != null;
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
      sponsored: _sponsored,
      schedule: schedule,
      images: imagesState.images,
      video: _draftVideoWrite(),
      existingRevision: existing?.revision,
      existingCreatedAt: existing?.createdAt,
    );
  }

  DraftVideoWrite? _draftVideoWrite() => _origin.videoWrite(
    selection: _videoController.selection,
    useStoredVideo: _usesStoredDraftVideo,
  );

  Future<void> _restoreDraftVideo(LocalPostDraftSeed seed) async {
    final descriptor = seed.draft.video;
    final owner = widget.draftOwner;
    if (descriptor == null || owner == null) return;
    try {
      final repository = await ref.read(
        accountLocalPostDraftRepositoryProvider(
          owner.session.account,
        ).future,
      );
      final video = await _origin.restoreVideo(seed, repository);
      if (!mounted || video == null) return;
      setState(() {
        _videoController.restoredSelection = video;
        _usesStoredDraftVideo = true;
      });
    } on Object {
      // Damaged drafts remain editable after the missing video is replaced.
    }
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
      widget.publishedEdit == null &&
      (hasDraft ?? _hasDraft(imagesState)) &&
      imagesState.canSaveDraftMedia() &&
      !_isSavingDraft &&
      !_isSubmitting;

  Future<bool> _confirmClose(ComposerImagesState imagesState) async {
    final eligible =
        widget.replyTarget == null &&
        widget.quoteTarget == null &&
        widget.scheduledPost == null &&
        widget.publishedEdit == null;
    if (!eligible) return await _confirmDiscard();
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

  Future<void> _selectVideo({bool replace = false}) async {
    try {
      final limits = replace
          ? await _videoController.replace()
          : await _videoController.selectExisting();
      if (!mounted) return;
      if (limits != null && !limits.canUpload) {
        context.showError(
          videoLimitsMessage(AppLocalizations.of(context), limits),
        );
        return;
      }
      final quota = limits == null
          ? null
          : videoQuotaMessage(AppLocalizations.of(context), limits);
      if (quota != null) {
        context.showInfo(quota);
      }
      setState(() => _usesStoredDraftVideo = false);
    } on Object {
      if (mounted) {
        context.showError(AppLocalizations.of(context).postCreateError);
      }
    }
  }

  Future<void> _publishVideo({
    required LocalVideoSelection selectedVideo,
    required ActiveAccountLease? owner,
    required String trimmedText,
    required List<Map<String, dynamic>> facets,
  }) async {
    if (owner == null) {
      throw DiagnosticStateError('Video publication requires an owner');
    }
    if (widget.prepareVideoProof case final prepare?) {
      final video = await prepare(selectedVideo);
      final created = await ref
          .read(createPostProvider.notifier)
          .create(
            text: trimmedText,
            langs: _languages!.values,
            sponsored: _sponsored,
            video: video,
            facets: facets.isEmpty ? null : facets,
            ownership: owner,
          );
      _submissionSucceeded = created != null;
      return;
    }
    final coordinator = composerVideoPublication(
      owner: owner,
      selection: selectedVideo,
      api: ref.read(postApiClientProvider),
      service: ref.read(videoServiceClientProvider),
      publish: (video, {required allowBlobRecovery}) async {
        final created = await ref
            .read(createPostProvider.notifier)
            .create(
              text: trimmedText,
              langs: _languages!.values,
              sponsored: _sponsored,
              video: video,
              facets: facets.isEmpty ? null : facets,
              ownership: owner,
              allowVideoBlobRecovery: allowBlobRecovery,
            );
        if (created == null) throw const VideoPublicationException(null);
        _submissionSucceeded = true;
      },
      onProgress: (progress) {
        if (mounted) setState(() => _videoProgress = progress);
      },
    );
    _videoPublication = coordinator;
    _videoPublicationOwner = owner;
    try {
      await coordinator.publish(
        altText: selectedVideo.altText,
        aspectRatio: (selectedVideo.width, selectedVideo.height),
      );
    } finally {
      if (identical(_videoPublication, coordinator)) {
        _videoPublication = null;
        _videoPublicationOwner = null;
        if (mounted) setState(() => _videoProgress = null);
      }
    }
  }

  bool _publishedImagesChanged(ComposerImagesState images) => !listEquals(
    images.images.map((image) => (image.id, image.altText)).toList(),
    _publishedImages.map((image) => (image.id, image.altText)).toList(),
  );

  Future<void> _refreshPublishedExternal() async {
    final edit = widget.publishedEdit;
    final media = _publishedEmbed;
    if (edit == null || media == null || _isRefreshingExternal) return;
    final operation = ActiveAccountOperation.fromWidget(ref, edit.ownership);
    final reporter = GuardedErrorReporter(ref.read(errorReporterProvider));
    final cancel = CancelToken();
    _externalRefreshCancel = cancel;
    setState(() => _isRefreshingExternal = true);
    try {
      operation.requireCurrent();
      final candidate = LinkPreviewCandidate.parse(media.externalUri);
      final preview = await ref
          .read(linkPreviewRepositoryProvider)
          .fetch(candidate.transportUri, cancel);
      operation.requireCurrent();
      if (!mounted || !media.hasExternal) return;
      setState(() {
        media
          ..updateExternal(
            uri: candidate.navigationUri(preview.url).toString(),
            title: preview.title,
            description: preview.description,
          )
          ..removeExternalThumbnail();
        _hasExternalReplacement = true;
        _replacementExternalThumbnail = preview.thumbnail;
      });
    } on Object catch (error, stack) {
      final mapped = AppErrorMapper.map(
        error,
        source: 'post_mutation',
        fallbackClassification: 'post.edit.preview.failed',
      );
      final diagnostic = ReportContext(
        feature: 'PostEditComposer',
        operation: 'post.edit.preview',
        classification: mapped.sentryClassification,
        safeDiagnostics: {
          ...mapped.safeDiagnostics,
          'failureStage': 'preview_refresh',
        },
        workflow: PublicRecordContext(
          actorDid: edit.ownership.session.account.did.value,
          recordUri: edit.seed.post.uri.value,
        ),
      );
      Logger('PostEditComposer').warning(
        DiagnosticMessage('post edit preview failed', context: diagnostic),
        error,
        stack,
      );
      if (mapped.reportable) {
        unawaited(
          reporter.captureException(
            error,
            stackTrace: stack,
            context: diagnostic,
          ),
        );
      }
      if (mounted && operation.isCurrent) {
        context.showError(AppLocalizations.of(context).postEditError);
      }
    } finally {
      if (identical(_externalRefreshCancel, cancel)) {
        _externalRefreshCancel = null;
      }
      if (mounted) setState(() => _isRefreshingExternal = false);
    }
  }

  Future<void> _savePublishedPostEdit(
    PostEditOrigin edit,
    ComposerImagesState images,
    String trimmedText,
  ) async {
    FocusScope.of(context).unfocus();
    await _runSubmission(edit.ownership, () async {
      final operation = ActiveAccountOperation.fromWidget(ref, edit.ownership)
        ..requireCurrent();
      final changes = <String, Object?>{...?_publishedEmbed?.changes};
      SelectedLinkPreview? newExternal;
      if (images.images.isEmpty &&
          _videoController.selection == null &&
          _publishedEmbed?.hasRetainedVideo != true &&
          _publishedEmbed?.hasExternal != true &&
          edit.seed.editable['unsupportedEmbed'] != true &&
          edit.seed.post.quote == null) {
        newExternal = ref
            .read(
              linkPreviewControllerProvider(
                _composerId,
                edit.ownership.session.account,
              ).notifier,
            )
            .snapshotForSubmit();
      }

      if (_text != _initialText) {
        changes['text'] = trimmedText;
        final facets = await _facetsForSubmission(trimmedText);
        operation.requireCurrent();
        changes['facets'] = facets.isEmpty ? null : facets;
      }
      if (_sponsored != _initialSponsored) {
        changes['sponsored'] = _sponsored;
      }
      if (!listEquals(_languages?.values, _initialLanguages)) {
        changes['langs'] = _languages!.values;
      }
      if (_publishedImagesChanged(images)) {
        final needsUpload = images.images.any(
          (image) => image.phase is ImageReady,
        );
        final client = needsUpload ? ref.read(postApiClientProvider) : null;
        changes['images'] = await _mediaUploader.materializePostEdit(
          composerId: _composerId,
          images: images.images,
          ownershipIsCurrent: () => operation.isCurrent,
          upload: ({required bytes, required mimeType, required cancelToken}) =>
              client!.uploadImage(
                bytes: bytes,
                mimeType: mimeType,
                cancelToken: cancelToken,
              ),
        );
        operation.requireCurrent();
      }
      if (_hasExternalReplacement &&
          _publishedEmbed?.hasExternal == true &&
          _videoController.selection == null) {
        final media = _publishedEmbed!;
        final selection = SelectedLinkPreview(
          candidate: LinkPreviewCandidate.parse(media.externalUri),
          preview: LinkPreview(
            url: Uri.parse(media.externalUri),
            title: media.externalTitle,
            description: media.externalDescription,
            thumbnail: _replacementExternalThumbnail,
          ),
        );
        final client = _replacementExternalThumbnail == null
            ? null
            : ref.read(postApiClientProvider);
        final external = await _mediaUploader.materializeImmediateExternal(
          composerId: _composerId,
          selection: selection,
          ownershipIsCurrent: () => operation.isCurrent,
          upload: ({required bytes, required mimeType, required cancelToken}) =>
              client!.uploadImage(
                bytes: bytes,
                mimeType: mimeType,
                cancelToken: cancelToken,
              ),
        );
        operation.requireCurrent();
        changes['embed'] = {
          'external': {...external.toMap(), 'thumb': external.thumb?.toMap()},
        };
      }
      if (newExternal != null) {
        final client = newExternal.preview.thumbnail == null
            ? null
            : ref.read(postApiClientProvider);
        final external = await _mediaUploader.materializeImmediateExternal(
          composerId: _composerId,
          selection: newExternal,
          ownershipIsCurrent: () => operation.isCurrent,
          upload: ({required bytes, required mimeType, required cancelToken}) =>
              client!.uploadImage(
                bytes: bytes,
                mimeType: mimeType,
                cancelToken: cancelToken,
              ),
        );
        operation.requireCurrent();
        changes['embed'] = {
          r'$type': 'app.bsky.embed.external',
          'external': external.toMap(),
        };
      }
      Future<void> submit(
        CreatePostVideo? video, {
        bool allowBlobRecovery = false,
      }) async {
        if (video != null) {
          changes['embed'] = {
            r'$type': 'app.bsky.embed.video',
            'video': video.blob.toMap(),
            'jobId': video.jobId,
            'alt': video.alt,
            if (video.aspectRatio != null)
              'aspectRatio': video.aspectRatio!.toMap(),
          };
          if (edit.seed.post.images?.isNotEmpty ?? false) {
            changes['images'] = null;
          }
        }
        final frozenChanges = PostEditChanges(changes);
        Future<void> sendFrozen({bool recoverBlob = false}) async {
          operation.requireCurrent();
          final result = await ref
              .read(editPostProvider.notifier)
              .save(
                seed: edit.seed,
                changes: frozenChanges,
                ownership: edit.ownership,
                allowVideoBlobRecovery: recoverBlob,
              );
          operation.requireCurrent();
          if (result == null) {
            throw const ComposerSubmissionFailure();
          }
          _submissionSucceeded = true;
          var visible = result.post;
          if (visible == null) {
            try {
              visible = await ref
                  .read(editPostProvider.notifier)
                  .refreshAccepted(
                    seed: edit.seed,
                    accepted: result,
                    ownership: edit.ownership,
                  );
            } on Object {
              // Acceptance is already final; a read failure cannot cause a
              // second write or present the edit as rejected.
            }
            operation.requireCurrent();
          }
          if (!mounted) return;
          if (visible case final post?) {
            _pendingPublishedDraftPost = post;
            context.showInfo(AppLocalizations.of(context).postEditSuccess);
          } else {
            setState(() => _acceptedEditPending = result);
            context.showInfo(AppLocalizations.of(context).postEditPending);
          }
        }

        _retryPublishedEdit = () => _runSubmission(edit.ownership, sendFrozen);
        await sendFrozen(recoverBlob: allowBlobRecovery);
      }

      final selectedVideo = _videoController.selection;
      if (selectedVideo == null) {
        await submit(null);
      } else if (widget.prepareVideoProof case final prepare?) {
        final video = await prepare(selectedVideo);
        operation.requireCurrent();
        await submit(video);
      } else {
        final coordinator = composerVideoPublication(
          owner: edit.ownership,
          selection: selectedVideo,
          api: ref.read(postApiClientProvider),
          service: ref.read(videoServiceClientProvider),
          publish: submit,
          onProgress: (progress) {
            if (mounted) setState(() => _videoProgress = progress);
          },
        );
        _videoPublication = coordinator;
        _videoPublicationOwner = edit.ownership;
        try {
          await coordinator.publish(
            altText: selectedVideo.altText,
            aspectRatio: (selectedVideo.width, selectedVideo.height),
          );
        } finally {
          if (identical(_videoPublication, coordinator)) {
            _videoPublication = null;
            _videoPublicationOwner = null;
            if (mounted) setState(() => _videoProgress = null);
          }
        }
      }
    });
  }

  Future<void> _reviewCurrentPublishedPost() async {
    final edit = widget.publishedEdit;
    if (edit == null) return;
    final operation = ActiveAccountOperation.fromWidget(ref, edit.ownership);
    if (!operation.isCurrent) return;
    final current = await showPublishedPostEditor(
      context,
      post: edit.seed.post,
      ownership: edit.ownership,
    );
    if (!mounted || !operation.isCurrent || current == null) return;
    setState(() => _pendingPublishedDraftPost = current);
    await WidgetsBinding.instance.endOfFrame;
    if (mounted && operation.isCurrent && Navigator.of(context).canPop()) {
      Navigator.of(context).pop(current);
    }
  }

  Future<void> _refreshPublishedAccepted() async {
    final edit = widget.publishedEdit;
    final accepted = _acceptedEditPending;
    if (edit == null || accepted == null) return;
    await _runSubmission(edit.ownership, () async {
      final operation = ActiveAccountOperation.fromWidget(ref, edit.ownership)
        ..requireCurrent();
      final post = await ref
          .read(editPostProvider.notifier)
          .refreshAccepted(
            seed: edit.seed,
            accepted: accepted,
            ownership: edit.ownership,
          );
      operation.requireCurrent();
      if (!mounted) return;
      if (post == null) {
        context.showInfo(AppLocalizations.of(context).postEditPending);
        return;
      }
      setState(() => _acceptedEditPending = null);
      _submissionSucceeded = true;
      _pendingPublishedDraftPost = post;
      context.showInfo(AppLocalizations.of(context).postEditSuccess);
    });
  }

  Future<void> _submitPost({required String trimmedText}) async {
    final imagesState = ref.read(composerImagesProvider(_composerId));
    setState(() => _attemptedSubmit = true);
    if (trimmedText.isEmpty ||
        !RecordTextLimits.post.accepts(_text) ||
        !imagesState.canSubmitImages()) {
      if (trimmedText.isEmpty || !RecordTextLimits.post.accepts(_text)) {
        _focusNode.requestFocus();
        if (_focusNode.context case final bodyContext?) {
          await Scrollable.ensureVisible(bodyContext);
        }
      }
      return;
    }
    if (widget.publishedEdit case final edit?) {
      await _savePublishedPostEdit(edit, imagesState, trimmedText);
      return;
    }
    // Release the editor's input connection before async work or route removal.
    FocusScope.of(context).unfocus();
    final activeLease = ref.read(sessionRegistryProvider).value?.activeLease;
    SelectedLinkPreview? previewSelection;
    var scheduledExternalDisposition = _ScheduledExternalDisposition.remove;
    if (activeLease != null && widget.quoteTarget == null) {
      final provider = linkPreviewControllerProvider(
        _composerId,
        activeLease.session.account,
      );
      final previewController = ref.read(provider.notifier);
      if (imagesState.images.isEmpty && !previewController.dismissed) {
        previewSelection = previewController.snapshotForSubmit();
      } else if (widget.scheduledPost != null) {
        previewController.snapshotForSubmit();
      }
      if (previewSelection != null) {
        final existing = widget.scheduledPost?.payload['external'];
        scheduledExternalDisposition =
            existing is Map<dynamic, dynamic> &&
                _selectionMatchesFrozenExternal(previewSelection, existing)
            ? _ScheduledExternalDisposition.preserve
            : _ScheduledExternalDisposition.attach;
      }
    }
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
    final submissionOwner = activeLease;
    if (_videoController.selection case final selectedVideo?) {
      await _runSubmission(
        submissionOwner,
        () => _publishVideo(
          selectedVideo: selectedVideo,
          owner: submissionOwner,
          trimmedText: trimmedText,
          facets: facets,
        ),
      );
      return;
    }
    final needsUpload =
        imagesState.images.any(
          (image) => image.phase is ImageReady,
        ) ||
        previewSelection?.preview.thumbnail != null;
    final uploadClient = needsUpload ? ref.read(postApiClientProvider) : null;

    await _runSubmission(submissionOwner, () async {
      if (widget.scheduledPost != null &&
          _scheduleChoice == ScheduleChoice.now) {
        await _publishExistingNow(
          trimmedText: trimmedText,
          facets: facets,
          imagesState: imagesState,
          previewSelection: previewSelection,
          externalDisposition: scheduledExternalDisposition,
        );
        return;
      }

      if (_scheduleChoice == ScheduleChoice.later) {
        await _schedulePost(
          trimmedText: trimmedText,
          facets: facets,
          imagesState: imagesState,
          previewSelection: previewSelection,
          externalDisposition: scheduledExternalDisposition,
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
      final external = previewSelection == null
          ? null
          : await _mediaUploader.materializeImmediateExternal(
              composerId: _composerId,
              selection: previewSelection,
              ownershipIsCurrent: () =>
                  _submissionOwnershipIsCurrent(submissionOwner),
              upload:
                  ({required bytes, required mimeType, required cancelToken}) =>
                      uploadClient!.uploadImage(
                        bytes: bytes,
                        mimeType: mimeType,
                        cancelToken: cancelToken,
                      ),
            );
      final created = await ref
          .read(createPostProvider.notifier)
          .create(
            text: trimmedText,
            langs: _languages!.values,
            sponsored: widget.replyTarget == null && _sponsored,
            reply: _replyFor(widget.replyTarget),
            quote: _quoteFor(widget.quoteTarget),
            images: images,
            external: external,
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
    _pendingPublishedDraftPost = null;
    _videoFailure = null;
    final accountOperation = ActiveAccountOperation.withGuard(
      ownership: submissionOwner,
      isCurrent: () => _submissionOwnershipIsCurrent(submissionOwner),
    );
    await _submissionCoordinator.run(
      presentOverlay: () async {
        await WidgetsBinding.instance.endOfFrame;
        if (!mounted) throw DiagnosticStateError('composer disposed');
      },
      ownershipIsCurrent: () => accountOperation.isCurrent,
      saveOriginSnapshot: _saveOriginSnapshot,
      operation: () async {
        try {
          await operation();
        } on Object catch (error) {
          if (_videoController.selection != null) _videoFailure = error;
          rethrow;
        }
      },
      didSucceed: () => _submissionSucceeded,
      deleteOriginAfterSuccess: _deleteOriginDraftAfterSuccess,
      onRunningChanged: ({required running}) {
        if (mounted) setState(() => _isSubmitting = running);
      },
      onFailure: (_) {
        if (mounted &&
            (widget.publishedEdit == null ||
                _submissionOwnershipIsCurrent(submissionOwner))) {
          final l10n = AppLocalizations.of(context);
          final validationMessage = widget.publishedEdit == null
              ? null
              : postEditValidationMessage(
                  l10n,
                  ref.read(editPostProvider).error,
                );
          final unavailable =
              widget.publishedEdit != null &&
              switch (ref.read(editPostProvider).error) {
                ApiBadRequest(code: 'post_not_found') => true,
                _ => false,
              };
          final conflict =
              widget.publishedEdit != null &&
              switch (ref.read(editPostProvider).error) {
                ApiBadRequest(code: 'pds_record_conflict') => true,
                _ => false,
              };
          final unresolved =
              widget.publishedEdit != null &&
              ref.read(editPostProvider).error
                  is PdsMutationUnresolvedException;
          final retryOperation = unresolved && _retryPublishedEdit != null
              ? MessageAction(
                  label: l10n.postEditRetrySave,
                  onPressed: () => unawaited(_retryPublishedEdit!()),
                )
              : conflict
              ? MessageAction(
                  label: l10n.postEditReviewCurrent,
                  onPressed: () => unawaited(_reviewCurrentPublishedPost()),
                )
              : _videoController.selection == null
              ? null
              : MessageAction(
                  label: l10n.retryButton,
                  onPressed: () => unawaited(
                    _runSubmission(submissionOwner, operation),
                  ),
                );
          context.showError(
            unresolved
                ? l10n.postEditUnresolved
                : unavailable
                ? l10n.postEditUnavailable
                : conflict
                ? l10n.postEditConflict
                : validationMessage ??
                      (retryOperation == null
                          ? (widget.publishedEdit == null
                                ? l10n.postCreateError
                                : _acceptedEditPending != null
                                ? l10n.postEditRefreshError
                                : l10n.postEditError)
                          : videoPublicationFailureMessage(
                              l10n,
                              _videoFailure,
                            )),
            action: retryOperation,
          );
        }
      },
    );
    final published = _pendingPublishedDraftPost;
    if (published != null) await WidgetsBinding.instance.endOfFrame;
    if (mounted &&
        published != null &&
        Navigator.of(context).canPop() &&
        (widget.publishedEdit == null ||
            _submissionOwnershipIsCurrent(submissionOwner))) {
      Navigator.of(context).pop(published);
    }
  }

  bool _submissionOwnershipIsCurrent(ActiveAccountLease? owner) =>
      owner == null ||
      (ref.read(sessionRegistryProvider).value?.isCurrent(owner) ?? false);

  Future<void> _saveOriginSnapshot() => _origin.saveOriginSnapshot(
    expectedOwner: widget.draftOwner,
    activeOwner: ref.read(sessionRegistryProvider).value?.activeLease,
    save: (owner) {
      return ref
          .read(draftSaveControllerProvider(owner).notifier)
          .save(
            _draftWriteRequest(
              owner,
              ref.read(composerImagesProvider(_composerId)),
            ),
          );
    },
  );

  Future<void> _deleteOriginDraftAfterSuccess() async {
    final origin = _origin.draft;
    if (origin == null) return;
    try {
      await _origin.deleteAfterSuccess(
        repositoryFor: (owner) => ref.read(
          accountLocalPostDraftRepositoryProvider(owner).future,
        ),
        refresh: (owner) async {
          if (mounted) {
            await ref.read(localPostDraftsProvider(owner).notifier).refresh();
          }
        },
      );
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
    return await ref.read(facetGeneratorProvider).generate(text);
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

  ScheduledComposerSubmission _scheduledSubmission(ActiveAccountLease owner) {
    final account = owner.session.account;
    return ScheduledComposerSubmission(
      operation: ActiveAccountOperation.withGuard(
        ownership: owner,
        isCurrent: () => _scheduledOperationIsCurrent(owner),
      ),
      repository: () => ref.read(
        accountScheduledPostRepositoryProvider(account).future,
      ),
      refresh: () =>
          ref.read(scheduledPostsProvider(account).notifier).refresh(),
      materializeMedia: ref.read(scheduledComposerMediaMaterializerProvider),
    );
  }

  void _scheduleProgress(ScheduledComposerProgress progress) {
    if (!mounted) return;
    setState(() {
      _stagedImageCount = progress.staged;
      _stagedImageTotal = progress.total;
      _isSavingSchedule = progress.saving;
    });
  }

  Future<void> _runScheduledWrite(
    ActiveAccountLease owner, {
    required Future<void> Function(ScheduledComposerSubmission) write,
    required String errorMessage,
    String? successMessage,
  }) async {
    setState(() => _isScheduling = true);
    try {
      await write(_scheduledSubmission(owner));
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      _submissionSucceeded = true;
      Navigator.of(context).pop();
      if (successMessage != null) context.showInfo(successMessage);
    } on Object {
      if (mounted && _scheduledOperationIsCurrent(owner)) {
        context.showError(errorMessage);
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

  Future<Map<String, dynamic>> _scheduledPostPayload({
    required String trimmedText,
    required List<Map<String, dynamic>> facets,
    required List<Map<String, dynamic>> media,
    required SelectedLinkPreview? previewSelection,
    required _ScheduledExternalDisposition externalDisposition,
    required ScheduledMediaStager stageMedia,
  }) async {
    final external = await _scheduledExternalForSave(
      previewSelection,
      disposition: externalDisposition,
      stageMedia: stageMedia,
    );
    return {
      'kind': 'standard',
      'text': trimmedText,
      'langs': _languages!.values,
      'sponsored': _sponsored,
      'facets': ?facets.isEmpty ? null : facets,
      'media': media,
      'external': ?external,
    };
  }

  Future<void> _schedulePost({
    required String trimmedText,
    required List<Map<String, dynamic>> facets,
    required ComposerImagesState imagesState,
    required SelectedLinkPreview? previewSelection,
    required _ScheduledExternalDisposition externalDisposition,
  }) async {
    final owner = _captureScheduledOperationOwner();
    final scheduledAt = _scheduledAtLocal;
    if (owner == null || scheduledAt == null) return;
    final l10n = AppLocalizations.of(context);
    await _runScheduledWrite(
      owner,
      errorMessage: l10n.scheduledPostSaveError,
      successMessage: l10n.scheduledPostSaved,
      write: (submission) => submission.save(
        operationId: _composerId,
        scheduledAt: scheduledAt,
        existing: widget.scheduledPost,
        images: imagesState.images,
        onProgress: _scheduleProgress,
        payload: (repository, media) => _scheduledPostPayload(
          trimmedText: trimmedText,
          facets: facets,
          media: media,
          previewSelection: previewSelection,
          externalDisposition: externalDisposition,
          stageMedia: repository.stageMedia,
        ),
      ),
    );
  }

  Future<void> _publishExistingNow({
    required String trimmedText,
    required List<Map<String, dynamic>> facets,
    required ComposerImagesState imagesState,
    required SelectedLinkPreview? previewSelection,
    required _ScheduledExternalDisposition externalDisposition,
  }) async {
    final existing = widget.scheduledPost;
    final owner = _captureScheduledOperationOwner();
    if (existing == null || owner == null) return;
    await _runScheduledWrite(
      owner,
      errorMessage: AppLocalizations.of(context).scheduledPostNowError,
      write: (submission) => submission.publishNow(
        existing: existing,
        images: imagesState.images,
        onProgress: _scheduleProgress,
        payload: (repository, media) => _scheduledPostPayload(
          trimmedText: trimmedText,
          facets: facets,
          media: media,
          previewSelection: previewSelection,
          externalDisposition: externalDisposition,
          stageMedia: repository.stageMedia,
        ),
      ),
    );
  }

  Future<Map<String, dynamic>?> _scheduledExternalForSave(
    SelectedLinkPreview? selection, {
    required _ScheduledExternalDisposition disposition,
    required ScheduledMediaStager stageMedia,
  }) async {
    switch (disposition) {
      case _ScheduledExternalDisposition.remove:
        return null;
      case _ScheduledExternalDisposition.preserve:
        final existing = widget.scheduledPost?.payload['external'];
        if (existing is! Map<dynamic, dynamic>) {
          throw DiagnosticStateError('scheduled external is unavailable');
        }
        return Map<String, dynamic>.from(existing);
      case _ScheduledExternalDisposition.attach:
        if (selection == null) {
          throw DiagnosticStateError(
            'scheduled external selection is unavailable',
          );
        }
        return (await materializeScheduledExternal(
          selection,
          mediaId: widget.scheduledPost == null
              ? _replacementMediaIdFor(selection)
              : const Uuid().v4(),
          stageMedia: stageMedia,
        )).toMap();
    }
  }

  String _replacementMediaIdFor(SelectedLinkPreview selection) {
    final thumbnail = selection.preview.thumbnail;
    if (thumbnail == null) return const Uuid().v4();
    if (_replacementExternalMediaId == null ||
        _replacementExternalMimeType != thumbnail.mimeType ||
        !listEquals(_replacementExternalBytes, thumbnail.bytes)) {
      _replacementExternalMediaId = const Uuid().v4();
      _replacementExternalBytes = List.unmodifiable(thumbnail.bytes);
      _replacementExternalMimeType = thumbnail.mimeType;
    }
    return _replacementExternalMediaId!;
  }

  List<dynamic> get _scheduledPayloadMedia =>
      widget.scheduledPost?.payload['media'] as List<dynamic>? ?? const [];

  Future<void> _hydrateScheduledMedia() async {
    if (_scheduledPayloadMedia.isEmpty && _existingExternalMediaId == null) {
      return;
    }
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
      if (owner == null) throw DiagnosticStateError('account unavailable');
      final account = owner.session.account;
      final repository = await ref.read(
        accountScheduledPostRepositoryProvider(account).future,
      );
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      final images = _scheduledPayloadMedia.isEmpty
          ? const <ComposerImageDraft>[]
          : await hydrateScheduledComposerMedia(
              _scheduledPayloadMedia,
              loadBytes: repository.mediaBytes,
            );
      final externalThumbnail = _existingExternalMediaId == null
          ? null
          : await hydrateScheduledExternalThumbnail(
              _existingExternalMediaId,
              loadBytes: repository.mediaBytes,
            );
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      if (images.isNotEmpty) {
        ref
            .read(composerImagesProvider(_composerId).notifier)
            .seedScheduledImages(images);
      }
      setState(() {
        _scheduledExternalThumbnail = externalThumbnail;
        _isLoadingScheduledMedia = false;
      });
      final frozen = _scheduledExternalSelection(
        widget.scheduledPost,
        thumbnail: externalThumbnail,
      );
      if (frozen != null) {
        ref
            .read(
              linkPreviewControllerProvider(_composerId, account).notifier,
            )
            .refreshSeed(frozen);
      }
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
    final l10n = AppLocalizations.of(context);
    final confirmed = await showCraftskyConfirmDialog(
      context,
      title: l10n.scheduledPostsDeleteTitle,
      message: l10n.scheduledPostsDeleteMessage,
      confirmLabel: l10n.scheduledPostsDeleteAction,
      cancelLabel: l10n.languageCancel,
    );
    if (!confirmed || !mounted || !_scheduledOperationIsCurrent(owner)) return;
    setState(() {
      _isScheduling = true;
      _isDeletingSchedule = true;
    });
    try {
      await _scheduledSubmission(owner).delete(existing.id);
      if (!mounted || !_scheduledOperationIsCurrent(owner)) return;
      Navigator.of(context).pop();
    } on Object {
      if (mounted && _scheduledOperationIsCurrent(owner)) {
        context.showError(l10n.scheduledPostDeleteError);
      }
    } finally {
      if (mounted) {
        setState(() {
          _isScheduling = false;
          _isDeletingSchedule = false;
        });
      }
    }
  }
}

enum _ScheduledExternalDisposition { preserve, attach, remove }

String? _existingExternalThumbMediaId(ScheduledPostDetail? post) {
  final external = post?.payload['external'];
  if (external is! Map<dynamic, dynamic>) return null;
  final id = external['thumbMediaId'];
  return id is String ? id : null;
}

SelectedLinkPreview? _scheduledExternalSelection(
  ScheduledPostDetail? post, {
  LinkPreviewThumbnail? thumbnail,
}) {
  final external = post?.payload['external'];
  if (external is! Map<dynamic, dynamic>) return null;
  try {
    final frozen = ScheduledPostExternal.fromMap(
      Map<String, dynamic>.from(external),
    );
    return SelectedLinkPreview(
      candidate: LinkPreviewCandidate.parse(frozen.sourceUri),
      preview: LinkPreview(
        url: Uri.parse(frozen.uri),
        title: frozen.title,
        description: frozen.description,
        thumbnail: thumbnail,
      ),
    );
  } on FormatException {
    return null;
  }
}

bool _selectionMatchesFrozenExternal(
  SelectedLinkPreview selection,
  Map<dynamic, dynamic> existing,
) {
  final frozen = ScheduledPostExternal.fromMap(
    Map<String, dynamic>.from(existing),
  );
  return frozen.sourceUri == selection.candidate.identity.toString() &&
      frozen.uri == selection.navigationUri.toString() &&
      frozen.title == selection.preview.title &&
      frozen.description == selection.preview.description &&
      ((frozen.thumbMediaId == null && selection.preview.thumbnail == null) ||
          (frozen.thumbMediaId != null && selection.preview.thumbnail != null));
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
  const new({required this.post});

  final Post post;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final l10n = AppLocalizations.of(context);
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
            Row(
              children: [
                ProfileAvatar(
                  seed: ProfileHandle(post.author.handle).displayLabel(
                    displayName: displayName,
                    unavailableLabel: l10n.handleUnavailable,
                  ),
                  avatarUrl: post.author.avatar,
                  size: ProfileAvatarSize.small,
                  customisation: post.author.customisation,
                ),
                SizedBox(width: spacing.sp3),
                Expanded(
                  child: CompactProfileIdentity(
                    handle: post.author.handle.toString(),
                    displayName: displayName,
                    crafts: post.author.crafts,
                  ),
                ),
              ],
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

PostReply? _replyFor(Post? target) {
  if (target == null) return null;

  return PostReply(
    root: target.reply?.root ?? PostRef(uri: target.uri, cid: target.cid),
    parent: PostRef(uri: target.uri, cid: target.cid),
  );
}
