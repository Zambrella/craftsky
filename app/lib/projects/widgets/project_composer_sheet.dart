import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/drafts/composer/composer_draft_session.dart';
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/composer/draft_schedule_restoration.dart';
import 'package:craftsky_app/drafts/composer/save_composer_snapshot.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/draft_save_controller.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/drafts/providers/local_post_drafts_provider.dart';
import 'package:craftsky_app/drafts/widgets/draft_close_dialog.dart';
import 'package:craftsky_app/feed/composer/composer_media_uploader.dart';
import 'package:craftsky_app/feed/composer/composer_submission_coordinator.dart';
import 'package:craftsky_app/feed/composer/composer_video_publication.dart';
import 'package:craftsky_app/feed/composer/post_edit_changes.dart';
import 'package:craftsky_app/feed/composer/post_edit_hydrator.dart';
import 'package:craftsky_app/feed/composer/post_edit_origin.dart';
import 'package:craftsky_app/feed/composer/post_edit_validation_message.dart';
import 'package:craftsky_app/feed/composer/published_post_edit_submission.dart';
import 'package:craftsky_app/feed/composer/published_post_embed_edit.dart';
import 'package:craftsky_app/feed/composer/submission_screen_awake.dart';
import 'package:craftsky_app/feed/composer/video_failure_message.dart';
import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/feed/providers/edit_post_provider.dart';
import 'package:craftsky_app/feed/providers/image_picker_existing_video.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/feed/providers/video_service_client_provider.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/feed/widgets/composer_metadata_controls.dart';
import 'package:craftsky_app/feed/widgets/composer_video_attachment_card.dart';
import 'package:craftsky_app/feed/widgets/composer_video_details_page.dart';
import 'package:craftsky_app/feed/widgets/published_post_edit_context.dart';
import 'package:craftsky_app/feed/widgets/published_post_editor.dart';
import 'package:craftsky_app/feed/widgets/submission_blocking_overlay.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/post_language_selection.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/composer/project_composer_draft_state.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/composer/project_composer_hydrator.dart';
import 'package:craftsky_app/projects/composer/project_composer_payload.dart';
import 'package:craftsky_app/projects/composer/project_composer_submit_adapter.dart';
import 'package:craftsky_app/projects/composer/project_draft_snapshot_adapter.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_detail_fields.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/providers/preferred_project_crafts_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_detail_page.dart';
import 'package:craftsky_app/projects/widgets/project_craft_fields.dart';
import 'package:craftsky_app/projects/widgets/project_facet_field.dart';
import 'package:craftsky_app/projects/widgets/project_pattern_editor.dart';
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
import 'package:craftsky_app/shared/image/basic_image_editor.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/shared/rich_text/facet_autocomplete_controller.dart';
import 'package:craftsky_app/shared/rich_text/hashtag_budget.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/shared/rich_text/widgets/facet_autocomplete_editor.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/shared/widgets/composer_bottom_action.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_text_field.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/publishing_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:material_ui/material_ui.dart';
import 'package:uuid/uuid.dart';

Future<Post?> showProjectComposerSheet(
  BuildContext context, {
  ScheduledPostDetail? scheduledPost,
  ActiveAccountLease? scheduledOwner,
  PostEditOrigin? publishedEdit,
  LocalPostDraftSeed? draftSeed,
  ActiveAccountLease? draftOwner,
}) {
  return responsiveModalNavigator(context).push<Post?>(
    FullscreenModalRoute<Post?>(
      builder: (_) => ProjectComposerSheet(
        scheduledPost: scheduledPost,
        scheduledOwner: scheduledOwner,
        publishedEdit: publishedEdit,
        draftSeed: draftSeed,
        draftOwner: draftOwner,
      ),
    ),
  );
}

String _projectOffsetLabel(Duration offset) {
  final minutes = offset.inMinutes.abs();
  final sign = offset.isNegative ? '-' : '+';
  return '$sign${(minutes ~/ 60).toString().padLeft(2, '0')}:'
      '${(minutes % 60).toString().padLeft(2, '0')}';
}

class ProjectComposerSheet extends ConsumerStatefulWidget {
  const new({
    super.key,
    this.composerId,
    this.scheduledPost,
    this.scheduledOwner,
    this.publishedEdit,
    this.draftSeed,
    this.draftOwner,
    this.videoController,
    this.prepareVideoProof,
  }) : assert(
         publishedEdit == null || (scheduledPost == null && draftSeed == null),
         'Published, scheduled and draft origins are mutually exclusive',
       ),
       assert(
         scheduledPost == null || draftSeed == null,
         'Scheduled and local draft edits are mutually exclusive',
       );

  static const maxCharacters = 2000;

  final String? composerId;
  final ScheduledPostDetail? scheduledPost;
  final ActiveAccountLease? scheduledOwner;
  final PostEditOrigin? publishedEdit;
  final LocalPostDraftSeed? draftSeed;
  final ActiveAccountLease? draftOwner;
  final ComposerVideoController? videoController;
  final Future<CreatePostVideo> Function(LocalVideoSelection video)?
  prepareVideoProof;

  @override
  ConsumerState<ProjectComposerSheet> createState() =>
      _ProjectComposerSheetState();
}

class _ProjectComposerSheetState extends ConsumerState<ProjectComposerSheet>
    with WidgetsBindingObserver {
  late final Map<String, dynamic> _initialFormValues;
  late final Project? _existingScheduledProject = _readScheduledProject();
  bool _craftChosen = false;
  late final Map<String, dynamic> _detailFormValues;
  static const List<String> _patternDetailFieldNames = [
    ProjectComposerFields.patternName,
    ProjectComposerFields.patternDesigner,
    ProjectComposerFields.patternPublisher,
    ProjectComposerFields.patternUrl,
    ProjectComposerFields.patternDifficulty,
    ProjectComposerFields.patternSelfDrafted,
  ];
  final _extraPatternKeys = <UniqueKey>[];
  final _formKey = GlobalKey<FormBuilderState>();
  final _patternDetailsFormKey = GlobalKey<FormBuilderState>();
  final _craftDetailsFormKey = GlobalKey<FormBuilderState>();
  final _scrollController = ScrollController();
  final _bodyController = FacetTextEditingController();
  final _bodyFocusNode = FocusNode(debugLabel: 'projectComposerBody');
  final _primaryActionFocusNode = FocusNode(
    debugLabel: 'projectComposerPrimaryAction',
  );
  final _patternDetailsFocusNode = FocusNode(
    debugLabel: 'projectPatternDetailsAction',
  );
  final _craftDetailsFocusNode = FocusNode(
    debugLabel: 'projectCraftDetailsAction',
  );
  final _patternNameController = FacetTextEditingController();
  final _patternDesignerController = FacetTextEditingController();
  final _patternPublisherController = FacetTextEditingController();
  final _patternNameFocusNode = FocusNode(debugLabel: 'projectPatternName');
  final _patternDesignerFocusNode = FocusNode(
    debugLabel: 'projectPatternDesigner',
  );
  final _patternPublisherFocusNode = FocusNode(
    debugLabel: 'projectPatternPublisher',
  );
  late final String _composerId;
  String _bodyText = '';
  String? _activeCraftType;
  bool _attemptedSubmit = false;
  String? _formValidationError;
  int? _lastImageNoticeId;
  AccountSessionLease? _unsavedOwner;
  UnsavedWorkRegistration? _unsavedRegistration;
  late final UnsavedWorkGuard _unsavedGuard;
  PostLanguageSelection? _languages;
  ScheduleChoice _scheduleChoice = ScheduleChoice.now;
  DateTime? _scheduledAtLocal;
  DateTime? _missedScheduledAtLocal;
  bool _isScheduling = false;
  bool _isDeletingSchedule = false;
  int _stagedImageCount = 0;
  int _stagedImageTotal = 0;
  bool _isSavingSchedule = false;
  bool _isLoadingScheduledMedia = false;
  bool _scheduledMediaLoadFailed = false;
  ActiveAccountLease? _scheduledOwner;
  late final ComposerMediaUploader _mediaUploader;
  late final ComposerSubmissionCoordinator _submissionCoordinator;
  late final ComposerVideoController _videoController;
  VideoPublicationCoordinator? _videoPublication;
  ActiveAccountLease? _videoPublicationOwner;
  VideoPublicationProgress? _videoProgress;
  Object? _videoFailure;
  var _usesStoredDraftVideo = false;
  bool _isSubmitting = false;
  bool _isSavingDraft = false;
  bool _sponsored = false;
  bool _initialSponsored = false;
  bool _submissionSucceeded = false;
  Post? _pendingPublishedDraftPost;
  late final ComposerDraftSession _origin;
  late final PublishedPostEmbedEdit? _publishedEmbed;
  Future<void> Function()? _retryPublishedEdit;
  List<ComposerImageDraft> _publishedImages = const [];
  String _initialBodyText = '';
  List<String>? _initialLanguages;
  ScheduleChoice _initialScheduleChoice = ScheduleChoice.now;
  DateTime? _initialScheduledAtLocal;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _origin = ComposerDraftSession(widget.draftSeed?.draft);
    _publishedEmbed = widget.publishedEdit == null
        ? null
        : PublishedPostEmbedEdit(
            (widget.publishedEdit!.seed.editable['embed'] as Map?)
                ?.cast<String, Object?>(),
          );
    _scheduledOwner = widget.scheduledOwner;
    final draftContent = widget.draftSeed?.draft.content;
    _initialFormValues = widget.publishedEdit != null
        ? hydratePublishedProjectComposer(widget.publishedEdit!.seed)
        : draftContent is ProjectDraftContent
        ? const ProjectDraftSnapshotAdapter().decodeKnownFields(
            Map<String, dynamic>.from(draftContent.knownProjectFieldValues),
          )
        : hydrateScheduledProjectComposer(
            _existingScheduledProject?.toMap(),
          );
    if (widget.draftSeed == null &&
        widget.scheduledPost == null &&
        widget.publishedEdit == null) {
      final defaultCraft = ref.read(defaultProjectCraftProvider);
      if (defaultCraft != null) {
        _initialFormValues[ProjectComposerFields.craftType] = defaultCraft;
      }
      ref.listenManual(defaultProjectCraftProvider, (_, craft) {
        if (!mounted ||
            craft == null ||
            _craftChosen ||
            _activeCraftType != null) {
          return;
        }
        // Apply a late profile load once, keeping the form pristine.
        _initialFormValues[ProjectComposerFields.craftType] = craft;
        _formKey.currentState?.fields[ProjectComposerFields.craftType]
            ?.didChange(craft);
        setState(() => _activeCraftType = craft);
      });
    }
    _detailFormValues = {
      for (final name in [
        ..._patternDetailFieldNames,
        ProjectComposerFields.patterns,
        ProjectComposerFields.additionalCrafts,
        if (_initialFormValues[ProjectComposerFields.craftType]
            case final String craft) ...[
          ProjectComposerFields.projectTypeFor(craft),
          ProjectComposerFields.projectSubtypeFor(craft),
          ProjectDetailFields.formName(craft),
        ],
      ])
        if (_initialFormValues.containsKey(name))
          name: _initialFormValues[name],
    };
    _extraPatternKeys.addAll(_patternEntries.skip(1).map((_) => UniqueKey()));
    _patternNameController.text = _patternDisplayText(
      _initialFormValues[ProjectComposerFields.patternName],
    );
    _patternDesignerController.text =
        _initialFormValues[ProjectComposerFields.patternDesigner] as String? ??
        '';
    _patternPublisherController.text =
        _initialFormValues[ProjectComposerFields.patternPublisher] as String? ??
        '';
    _activeCraftType =
        _initialFormValues[ProjectComposerFields.craftType] as String?;
    _unsavedGuard = ref.read(unsavedWorkGuardProvider);
    _composerId = widget.composerId ?? const Uuid().v4();
    _mediaUploader = ComposerMediaUploader();
    _submissionCoordinator = ComposerSubmissionCoordinator(
      screenAwake: const WakelockSubmissionScreenAwake(),
      reporter: ref.read(errorReporterProvider),
    );
    _videoController =
        widget.videoController ??
        ComposerVideoController(
          picker: ImagePickerExistingVideo(),
          checkEligibility: () =>
              ref.read(postApiClientProvider).getVideoUploadLimits(),
        );
    if (widget.publishedEdit case final edit?) {
      _bodyText = edit.seed.editable['text']! as String;
      _initialBodyText = _bodyText;
      _bodyController.text = _bodyText;
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
    if (draftContent is ProjectDraftContent) {
      _bodyText = draftContent.body;
      _initialBodyText = draftContent.body;
      _bodyController.text = draftContent.body;
      if (draftContent.languages.isNotEmpty) {
        _languages = PostLanguageSelection.fromValues(draftContent.languages);
        _initialLanguages = List.of(draftContent.languages);
      }
      _sponsored = draftContent.sponsored;
      final restored = restoreDraftSchedule(
        widget.draftSeed!.draft.schedule,
        now: DateTime.now(),
      );
      _scheduleChoice = restored.choice;
      _scheduledAtLocal = restored.scheduledAtLocal;
      _missedScheduledAtLocal = restored.needsExplanation
          ? widget.draftSeed!.draft.schedule.scheduledAtUtc?.toLocal()
          : null;
      _initialScheduleChoice = _scheduleChoice;
      _initialScheduledAtLocal = _scheduledAtLocal;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          ref
              .read(composerImagesProvider(_composerId).notifier)
              .seedLocalDraft(widget.draftSeed!);
          unawaited(_restoreDraftVideo(widget.draftSeed!));
        }
      });
    }
    if (widget.scheduledPost case final scheduled?) {
      _bodyText = scheduled.payload['text'] as String? ?? '';
      _bodyController.text = _bodyText;
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
      if (_scheduledPayloadMedia.isNotEmpty) {
        _isLoadingScheduledMedia = true;
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted) unawaited(_hydrateScheduledMedia());
        });
      }
    }
    _initialSponsored = _sponsored;
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _unsavedGuard.unregister(_unsavedRegistration);
    unawaited(_submissionCoordinator.dispose());
    _videoPublication?.cancel();
    _scrollController.dispose();
    _bodyController.dispose();
    _bodyFocusNode.dispose();
    _primaryActionFocusNode.dispose();
    _patternDetailsFocusNode.dispose();
    _craftDetailsFocusNode.dispose();
    _patternNameController.dispose();
    _patternDesignerController.dispose();
    _patternPublisherController.dispose();
    _patternNameFocusNode.dispose();
    _patternDesignerFocusNode.dispose();
    _patternPublisherFocusNode.dispose();
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
    final editUnresolved = editState?.error is PdsMutationUnresolvedException;
    final editUnavailable = switch (editState?.error) {
      ApiBadRequest(code: 'post_not_found') => true,
      _ => false,
    };

    final preferences = ref.watch(activeLanguagePreferencesProvider);
    _languages ??= PostLanguageSelection.fromPrimary(
      preferences.primaryLanguage,
    );
    _initialLanguages ??= List.of(_languages!.values);
    final imagesProvider = composerImagesProvider(_composerId);
    final imagesState = ref.watch(imagesProvider);
    final selectedVideo = _videoController.selection;
    final activeLease = ref.watch(sessionRegistryProvider).value?.activeLease;
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
    if (widget.publishedEdit case final edit?) {
      if (ref.read(sessionRegistryProvider).hasValue &&
          activeLease != edit.ownership) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted && Navigator.of(context).canPop()) {
            Navigator.of(context).pop();
          }
        });
        return const SizedBox.shrink();
      }
    }
    final account = activeLease?.session.account;
    final scheduleAccess = activeLease == null
        ? const AsyncLoading<SubscriptionAccess>()
        : ref.watch(subscriptionAccessProvider(activeLease.session));
    final maySchedule = switch (scheduleAccess) {
      AsyncData(:final value) => value.allowsPlus,
      _ => false,
    };
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
    final controlsEnabled =
        !editUnresolved &&
        !_isSavingDraft &&
        !createState.isLoading &&
        !_isSubmitting &&
        !_isScheduling &&
        !_isLoadingScheduledMedia &&
        !_scheduledMediaLoadFailed;
    final trimmedBody = _bodyText.trim();
    final tooLong = !RecordTextLimits.post.accepts(_bodyText);
    final canSubmit =
        !editUnresolved &&
        !editUnavailable &&
        (widget.publishedEdit == null ||
            activeLease == widget.publishedEdit!.ownership) &&
        !_isSavingDraft &&
        !_isScheduling &&
        !createState.isLoading &&
        !_isSubmitting &&
        !_isLoadingScheduledMedia &&
        !_scheduledMediaLoadFailed &&
        _languages != null &&
        (_publishedEmbed?.hasRetainedVideo == true ||
            selectedVideo != null ||
            imagesState.canSubmitImages()) &&
        (selectedVideo == null || _scheduleChoice == ScheduleChoice.now) &&
        (_scheduleChoice == ScheduleChoice.now ||
            (capacity.scheduleEnabled && maySchedule));
    final bodyErrorText = switch ((_attemptedSubmit, trimmedBody.isEmpty)) {
      (true, true) => l10n.projectComposerBodyRequiredError,
      _ when tooLong => l10n.postComposeTooLong,
      _ => null,
    };
    final formValues = _combinedFormValues();
    final photoErrorText =
        _attemptedSubmit &&
            imagesState.images.isEmpty &&
            selectedVideo == null &&
            _publishedEmbed?.hasRetainedVideo != true
        ? l10n.projectComposerPhotoRequiredError
        : null;
    final hasDraft =
        ProjectComposerDraftState.hasDraft(
          bodyText: _bodyText,
          initialBodyText: _initialBodyText,
          imageCount: _draftMediaChanged(imagesState) || _draftVideoChanged()
              ? 1
              : 0,
          formValues: formValues,
          initialFormValues: _initialFormValues,
        ) ||
        !listEquals(_languages?.values, _initialLanguages) ||
        _scheduleChoice != _initialScheduleChoice ||
        _scheduledAtLocal != _initialScheduledAtLocal ||
        _sponsored != _initialSponsored;
    final canSaveDraft =
        widget.scheduledPost == null &&
        widget.publishedEdit == null &&
        hasDraft &&
        imagesState.canSaveDraftMedia() &&
        !_isSavingDraft &&
        !_isSubmitting;
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
        _consumeImageNotice(
          l10n: l10n,
          notice: next.notice,
          clearNotice: (noticeId) =>
              ref.read(imagesProvider.notifier).clearNotice(noticeId),
        );
      });
    if (imagesState.notice case final notice?) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        _consumeImageNotice(
          l10n: l10n,
          notice: notice,
          clearNotice: (noticeId) =>
              ref.read(imagesProvider.notifier).clearNotice(noticeId),
        );
      });
    }

    return PopScope<Post?>(
      canPop:
          !_isSavingDraft &&
          !_isDeletingSchedule &&
          !_isSubmitting &&
          (_pendingPublishedDraftPost != null ||
              !hasDraft ||
              createState.isLoading),
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        if (_isSubmitting || _isSavingDraft || _isDeletingSchedule) return;
        final shouldClose = await _confirmProjectClose(imagesState);
        if (!shouldClose || !context.mounted) return;
        Navigator.of(context).pop();
      },
      child: Stack(
        fit: StackFit.expand,
        children: [
          Scaffold(
            backgroundColor: swatches.paper,
            appBar: AppBar(
              title: Text(
                l10n.projectComposerTitle,
                style: theme.textTheme.titleLarge,
              ),
              actions: [
                if (widget.scheduledPost == null &&
                    widget.publishedEdit == null)
                  TextButton(
                    onPressed: canSaveDraft
                        ? () => _saveProjectDraft(imagesState)
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
            body: GestureDetector(
              behavior: HitTestBehavior.translucent,
              onTap: () => FocusManager.instance.primaryFocus?.unfocus(),
              child: Stack(
                fit: StackFit.expand,
                children: [
                  SafeArea(
                    top: false,
                    bottom: false,
                    child: SingleChildScrollView(
                      controller: _scrollController,
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
                          if (editUnresolved) ...[
                            Semantics(
                              liveRegion: true,
                              child: Text(l10n.postEditUnresolved),
                            ),
                            SizedBox(height: spacing.sp4),
                          ],
                          FormBuilder(
                            key: _formKey,
                            initialValue: _initialFormValues,
                            onChanged: () {
                              if (mounted) setState(() {});
                            },
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.stretch,
                              children: [
                                _projectFields(
                                  l10n: l10n,
                                  spacing: spacing,
                                  imagesState: imagesState,
                                  selectedVideo: selectedVideo,

                                  controlsEnabled: controlsEnabled,
                                  photoErrorText: photoErrorText,
                                  onAddImages: () => ref
                                      .read(imagesProvider.notifier)
                                      .addImages(),
                                  onTakePhoto: () => ref
                                      .read(imagesProvider.notifier)
                                      .takePhoto(),
                                  onAddVideo: _selectVideo,
                                  onAltTextChanged: (imageId, value) => ref
                                      .read(imagesProvider.notifier)
                                      .setAltText(imageId, value),
                                  onRemoveImage: (imageId) => ref
                                      .read(imagesProvider.notifier)
                                      .remove(imageId),
                                  onReplaceUnavailable: (imageId, source) => ref
                                      .read(imagesProvider.notifier)
                                      .replaceUnavailable(
                                        imageId,
                                        source: source,
                                      ),
                                  onReorderImages: (fromIndex, toIndex) => ref
                                      .read(imagesProvider.notifier)
                                      .reorder(
                                        fromIndex: fromIndex,
                                        toIndex: toIndex,
                                      ),
                                ),
                                SizedBox(height: spacing.sp4),
                                _descriptionField(
                                  l10n: l10n,
                                  controlsEnabled: controlsEnabled,
                                  bodyErrorText: bodyErrorText,
                                ),
                                ComposerMetadataControls(
                                  languages: _languages!,
                                  onLanguagesChanged: controlsEnabled
                                      ? (value) =>
                                            setState(() => _languages = value)
                                      : null,
                                  sponsored: _sponsored,
                                  onSponsoredChanged: controlsEnabled
                                      ? (value) =>
                                            setState(() => _sponsored = value)
                                      : null,
                                  showSchedule: widget.publishedEdit == null,
                                  scheduledAtLocal: _scheduledAtLocal,
                                  scheduleAccess: scheduleAccess,
                                  onScheduleAccessRetry: activeLease == null
                                      ? null
                                      : () => ref.invalidate(
                                          subscriptionAccessProvider(
                                            activeLease.session,
                                          ),
                                        ),
                                  onSchedulePressed: controlsEnabled
                                      ? (menuContext) => _chooseWhen(
                                          menuContext,
                                          scheduleEnabled:
                                              capacity.scheduleEnabled,
                                        )
                                      : null,
                                ),
                                if (capacity.showCapacityWarning)
                                  const ScheduledPostCapacityWarning(),
                                if (capacity.showManageLink)
                                  Align(
                                    alignment: AlignmentDirectional.centerStart,
                                    child: TextButton(
                                      onPressed: () =>
                                          const ScheduledPostsRoute().go(
                                            context,
                                          ),
                                      child: Text(
                                        l10n.scheduledPostManageAction,
                                      ),
                                    ),
                                  ),
                                if (_missedScheduledAtLocal case final missed?)
                                  Text(
                                    l10n.scheduledPostMissedTime(
                                      _projectLocalTimeLabel(context, missed),
                                    ),
                                  ),
                                if (_isScheduling &&
                                    (_stagedImageTotal > 0 ||
                                        _isSavingSchedule)) ...[
                                  SizedBox(height: spacing.sp3),
                                  ScheduledStagingProgress(
                                    completed: _stagedImageCount,
                                    total: _stagedImageTotal,
                                    creating: _isSavingSchedule,
                                  ),
                                ],
                                if (widget.scheduledPost != null)
                                  Align(
                                    alignment: AlignmentDirectional.centerStart,
                                    child: TextButton.icon(
                                      onPressed:
                                          _isScheduling ||
                                              _isSavingDraft ||
                                              _isSubmitting
                                          ? null
                                          : _deleteExistingSchedule,
                                      icon: ActionProgress(
                                        isLoading: _isDeletingSchedule,
                                        child: const Icon(
                                          CraftskyIconsBold.delete,
                                        ),
                                      ),
                                      label: Text(
                                        l10n.scheduledPostsDeleteTooltip,
                                      ),
                                    ),
                                  ),
                              ],
                            ),
                          ),
                          SizedBox(height: spacing.sp4),
                          SizedBox(
                            key: const Key(
                              'project-composer-bottom-safe-space',
                            ),
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
                          doneKey: const Key('project-composer-done'),
                          actionKey: const Key(
                            'project-composer-primary-action',
                          ),
                          focusNode: _primaryActionFocusNode,
                          onPressed: editUnresolved
                              ? (!_isSubmitting &&
                                        _submissionOwnershipIsCurrent(
                                          widget.publishedEdit?.ownership,
                                        )
                                    ? _retryPublishedEdit
                                    : null)
                              : canSubmit
                              ? () => _submitProject(trimmedBody: trimmedBody)
                              : null,
                          label: editUnresolved
                              ? l10n.postEditRetrySave
                              : widget.publishedEdit != null
                              ? l10n.postEditSaveChanges
                              : _scheduleChoice == ScheduleChoice.later
                              ? l10n.scheduledPostAction
                              : l10n.postComposeSubmit,
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
          if (_isSubmitting)
            SubmissionBlockingOverlay(
              kind: widget.publishedEdit != null
                  ? SubmissionKind.edit
                  : SubmissionKind.post,
              scheduling: _scheduleChoice == ScheduleChoice.later,
              videoProgress: _videoProgress,
              onCancelVideo: _videoPublication?.cancel,
            ),
        ],
      ),
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

  Future<void> _selectVideo() async {
    final previous = _videoController.selection;
    try {
      final limits = await _videoController.selectExisting();
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
      final selected = _videoController.selection;
      if (selected != null && !identical(previous, selected)) {
        await showComposerVideoDetails(
          context,
          selection: selected,
          onAltTextChanged: (value) {
            if (mounted) setState(() => _videoController.setAltText(value));
          },
        );
      }
    } on Object catch (error) {
      if (mounted) {
        context.showError(
          videoSelectionFailureMessage(AppLocalizations.of(context), error),
        );
      }
    }
  }

  Future<bool> _confirmProjectClose(ComposerImagesState imagesState) async {
    if (widget.scheduledPost != null || widget.publishedEdit != null) {
      return await _confirmDiscard();
    }
    final choice = await showDraftCloseDialog(
      context,
      existingDraft: widget.draftSeed != null,
      canSave: imagesState.canSaveDraftMedia() && _hasUnsavedProject(),
    );
    switch (choice) {
      case DraftCloseChoice.save:
        await _saveProjectDraft(imagesState);
        return false;
      case DraftCloseChoice.discard:
        return true;
      case DraftCloseChoice.keepEditing:
        return false;
    }
  }

  bool _draftMediaChanged(ComposerImagesState imagesState) {
    if (widget.publishedEdit != null) {
      return !listEquals(
        imagesState.images.map((image) => (image.id, image.altText)).toList(),
        _publishedImages.map((image) => (image.id, image.altText)).toList(),
      );
    }
    final baseline = widget.draftSeed?.draft.media ?? const [];
    if (widget.draftSeed == null) return imagesState.images.isNotEmpty;
    if (imagesState.images.length != baseline.length) return true;
    for (var index = 0; index < imagesState.images.length; index++) {
      final image = imagesState.images[index];
      final stored = baseline[index];
      final digest = switch (image.phase) {
        ImageReady(:final sha256) => sha256,
        _ => 'unavailable',
      };
      if (image.id != stored.mediaId ||
          image.altText != stored.altText ||
          digest != stored.sha256) {
        return true;
      }
    }
    return false;
  }

  bool _draftVideoChanged() {
    final baseline = widget.draftSeed?.draft.video;
    final selection = _videoController.selection;
    if (baseline == null) return selection != null;
    if (selection == null || !_usesStoredDraftVideo) return true;
    return selection.altText != baseline.altText;
  }

  Future<void> _editImage(String imageId) async {
    final lease = ref.read(sessionRegistryProvider).value?.activeLease;
    final operation = ActiveAccountOperation.fromWidget(ref, lease);
    try {
      await ref
          .read(composerImagesProvider(_composerId).notifier)
          .editImage(
            imageId,
            initiatingLease: lease,
            editor: (request) =>
                showBasicImageEditor(context, request, ownership: lease),
          );
    } on Object {
      if (mounted && operation.isCurrent) {
        context.showError(AppLocalizations.of(context).imageEditFailed);
      }
    }
  }

  Future<void> _saveProjectDraft(ComposerImagesState imagesState) async {
    if (ref.read(composerImagesProvider(_composerId)).hasImageEditsInFlight) {
      return;
    }
    final active = ref.read(sessionRegistryProvider).value?.activeLease;
    if (active == null ||
        (widget.draftOwner != null && active != widget.draftOwner)) {
      return;
    }
    _formKey.currentState?.save();
    setState(() => _isSavingDraft = true);
    try {
      final request = _projectDraftWriteRequest(
        active.session.account,
        imagesState,
      );
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

  DraftWriteRequest _projectDraftWriteRequest(
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
    return const ProjectDraftSnapshotAdapter().toWriteRequest(
      id: existing?.id ?? _composerId,
      owner: owner,
      body: _bodyText,
      languages: _languages!.values,
      sponsored: _sponsored,
      schedule: schedule,
      formValues: _combinedFormValues(saved: true),
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

  void _ensureUnsavedWorkRegistration() {
    final owner = ref.read(sessionRegistryProvider).value?.activeLease?.session;
    if (owner == null || owner == _unsavedOwner) return;
    _unsavedOwner = owner;
    _unsavedRegistration = _unsavedGuard.replace(
      _unsavedRegistration,
      owner: owner,
      isDirty: _hasUnsavedProject,
      confirmAndClose: () async {
        if (!mounted) return true;
        final imagesState = ref.read(composerImagesProvider(_composerId));
        final shouldClose = await _confirmProjectClose(imagesState);
        if (!shouldClose || !mounted) return false;
        Navigator.of(context).pop();
        await Future<void>.delayed(Duration.zero);
        return true;
      },
    );
  }

  bool _hasUnsavedProject() {
    if (!mounted) return false;
    final imagesState = ref.read(composerImagesProvider(_composerId));
    final formValues = _combinedFormValues();
    return (_publishedEmbed?.changes.isNotEmpty ?? false) ||
        ProjectComposerDraftState.hasDraft(
          bodyText: _bodyText,
          initialBodyText: _initialBodyText,
          imageCount: _draftMediaChanged(imagesState) || _draftVideoChanged()
              ? 1
              : 0,
          formValues: formValues,
          initialFormValues: _initialFormValues,
        ) ||
        !listEquals(_languages?.values, _initialLanguages) ||
        _scheduleChoice != _initialScheduleChoice ||
        _scheduledAtLocal != _initialScheduledAtLocal ||
        _sponsored != _initialSponsored ||
        _draftVideoChanged();
  }

  void _consumeImageNotice({
    required AppLocalizations l10n,
    required ComposerImageNotice? notice,
    required void Function(int noticeId) clearNotice,
  }) {
    if (notice == null || _lastImageNoticeId == notice.id) return;
    _lastImageNoticeId = notice.id;
    switch (notice) {
      case ImageSelectionLimitNotice(:final maxImages):
        context.showError(l10n.postComposeImageLimitError(maxImages));
      case UnsupportedImagesNotice(:final count):
        context.showError(l10n.postComposeUnsupportedImagesError(count));
      case ImagePickerFailedNotice():
        context.showError(l10n.postComposeImagePickerError);
    }
    try {
      clearNotice(notice.id);
    } on Object {
      // Some focused widget tests override the image provider with a fixed
      // value, which has no notifier state to clear. The notice id guard still
      // keeps the one-shot behaviour observable in those tests.
    }
  }

  List<CraftskySelectOption<String>> _selectOptions(
    List<ProjectOption> options, {
    bool includeCraftIcons = false,
    bool alphabetize = true,
    String? retained,
  }) {
    final choices = [
      for (final option in [
        ...options,
        if (retained != null &&
            retained.trim().isNotEmpty &&
            !options.any((option) => option.value == retained))
          ProjectOption(value: retained, label: _retainedOptionLabel(retained)),
      ])
        CraftskySelectOption<String>(
          value: option.value,
          label: option.label,
          aliases: option.aliases,
          leadingBuilder: includeCraftIcons
              ? (_) => CraftIcon(craft: option.value)
              : null,
        ),
    ];
    return alphabetize ? alphabetizedSelectOptions(choices) : choices;
  }

  String _retainedOptionLabel(String value) {
    final knownLabel = ProjectOptionCatalogs.projectValueLabel(value);
    if (knownLabel != value) return knownLabel;
    final fragment = value.split('#').last;
    final label = fragment.isEmpty ? value : fragment;
    return label[0].toUpperCase() + label.substring(1);
  }

  List<String> _hashtagPrefix({
    bool pattern = false,
    bool designer = false,
    bool publisher = false,
  }) => [
    ...hashtagsInText(_bodyController.text),
    ...?(widget.publishedEdit == null
            ? _existingScheduledProject
            : ProjectMapper.fromMap(
                Map<String, dynamic>.from(
                  widget.publishedEdit!.seed.editable['project']! as Map,
                ),
              ))
        ?.common
        .tags,
    if (pattern || designer || publisher)
      ...hashtagsInText(_patternNameController.text),
    if (designer || publisher)
      ...hashtagsInText(_patternDesignerController.text),
    if (publisher) ...hashtagsInText(_patternPublisherController.text),
    if (publisher)
      for (final pattern in _patternEntries.skip(1))
        ..._patternHashtags(pattern),
  ];

  Map<String, dynamic> _combinedFormValues({bool saved = false}) => {
    ..._initialFormValues,
    ..._detailFormValues,
    ...?(saved
        ? _formKey.currentState?.value
        : _formKey.currentState?.instantValue),
  };

  Map<String, dynamic> _detailInitialValues(Iterable<String> names) => {
    for (final name in names) name: _detailFormValues[name],
  };

  void _syncDetailForm(
    GlobalKey<FormBuilderState> key,
    Iterable<String> names,
  ) {
    final values = key.currentState?.instantValue;
    if (values == null || !mounted) return;
    setState(() {
      for (final name in names) {
        if (values.containsKey(name)) _detailFormValues[name] = values[name];
      }
      _formValidationError = null;
    });
  }

  bool _hasAnyDetail(Iterable<String> names) => names.any(
    _hasDetail,
  );

  bool _hasDetail(String name) {
    final value = _detailFormValues[name];
    if (name == ProjectComposerFields.patternName && value == '#') return false;
    final craft = _activeCraftType;
    if (craft != null &&
        name == ProjectDetailFields.formName(craft) &&
        value is Map<Object?, Object?>) {
      return ProjectDetailFields.authoringForCraft(craft).any(
        (field) => _hasDetailValue(
          ProjectDetailFields.authoringValue(craft, field.key, value),
        ),
      );
    }
    return _hasDetailValue(value);
  }

  bool _hasDetailValue(Object? value) => switch (value) {
    final String text => text.trim().isNotEmpty,
    final Iterable<Object?> values => values.isNotEmpty,
    final num number => number > 0,
    true => true,
    _ => false,
  };

  String _detailSummary(
    Iterable<String> names, {
    required String empty,
    required AppLocalizations l10n,
  }) {
    final count = names.where(_hasDetail).length;
    return count == 0 ? empty : l10n.projectComposerDetailsAdded(count);
  }

  List<String> get _activeCraftDetailFieldNames {
    final craft = _activeCraftType;
    if (craft == null) return const [];
    return [
      if (ProjectOptionCatalogs.projectTypesForCraft(craft).isNotEmpty) ...[
        ProjectComposerFields.projectTypeFor(craft),
        ProjectComposerFields.projectSubtypeFor(craft),
      ],
      if (ProjectDetailFields.authoringForCraft(craft).isNotEmpty)
        ProjectDetailFields.formName(craft),
    ];
  }

  bool get _canEditSpecializedDetails =>
      widget.publishedEdit?.seed.editable['unsupportedProjectDetails'] != true;

  List<String> get _craftDetailsPageFieldNames => [
    if (_canEditSpecializedDetails) ..._activeCraftDetailFieldNames,
    ProjectComposerFields.additionalCrafts,
  ];

  String _craftDetailsTitle(AppLocalizations l10n) {
    final craft = ProjectOptionCatalogs.craftTypes
        .where((option) => option.value == _activeCraftType)
        .firstOrNull;
    return craft == null
        ? l10n.projectComposerGenericCraftDetailsTitle
        : l10n.projectComposerCraftDetailsTitle(craft.label);
  }

  String? get _craftItemSummary {
    final craft = _activeCraftType;
    if (craft == null ||
        !_canEditSpecializedDetails ||
        ProjectOptionCatalogs.projectTypesForCraft(craft).isEmpty) {
      return null;
    }
    final type = _detailFormValues[ProjectComposerFields.projectTypeFor(craft)];
    final subtype =
        _detailFormValues[ProjectComposerFields.projectSubtypeFor(craft)];
    final summary = [
      if (type case final String text when text.trim().isNotEmpty)
        ProjectOptionCatalogs.projectValueLabel(text.trim()),
      if (subtype case final String text when text.trim().isNotEmpty)
        if (type == ProjectOptionCatalogs.otherProjectTypeToken)
          text.trim()
        else
          ProjectOptionCatalogs.projectValueLabel(text.trim()),
    ].join(' · ');
    return summary.isEmpty ? null : summary;
  }

  String? get _primaryCraftId {
    final craft = _activeCraftType;
    return craft == null
        ? null
        : ProjectOptionCatalogs.profileIdForCraftToken(craft);
  }

  static const _craftsWithoutPatterns = {'spinning', 'dyeing'};

  bool get _showPatternDetails {
    if (!_craftsWithoutPatterns.contains(_primaryCraftId)) return true;
    final additionalCrafts =
        _detailFormValues[ProjectComposerFields.additionalCrafts];
    if (additionalCrafts is List<String> &&
        additionalCrafts
            .map(ProjectOptionCatalogs.profileIdForCraftToken)
            .any(_supportsPatterns)) {
      return true;
    }
    return projectComposerPatterns(
      _detailFormValues,
      savedFirstPattern: _savedPatternFor(_patternEntries.firstOrNull),
    ).any(
      (pattern) => pattern.toMap().entries.any(
        (field) =>
            field.key != 'existingIndex' &&
            switch (field.value) {
              final String text => text.trim().isNotEmpty,
              final List<Object?> values => values.isNotEmpty,
              final value => value != null && value != false,
            },
      ),
    );
  }

  static bool _supportsPatterns(String id) =>
      !_craftsWithoutPatterns.contains(id) &&
      ProjectOptionCatalogs.allCraftTypes.any(
        (craft) =>
            ProjectOptionCatalogs.profileIdForCraftToken(craft.value) == id,
      );

  ({String title, String description, String name}) _patternCopy(
    AppLocalizations l10n,
  ) => switch (_primaryCraftId) {
    'weaving' => (
      title: l10n.projectComposerPatternOrDraftTitle,
      description: l10n.projectComposerPatternOrDraftDescription,
      name: l10n.projectComposerPatternOrDraftName,
    ),
    'felting' => (
      title: l10n.projectComposerPatternOrTemplateTitle,
      description: l10n.projectComposerPatternOrTemplateDescription,
      name: l10n.projectComposerPatternOrTemplateName,
    ),
    _ => (
      title: l10n.projectComposerPatternDetailsTitle,
      description: l10n.projectComposerPatternDetailsDescription,
      name: l10n.projectComposerPatternNameLabel,
    ),
  };

  Future<void> _openPatternDetails() async {
    FocusManager.instance.primaryFocus?.unfocus();
    final l10n = AppLocalizations.of(context);
    await Navigator.of(context).push<void>(
      MaterialPageRoute<void>(
        builder: (_) => StatefulBuilder(
          builder: (context, setRouteState) => ProjectComposerDetailPage(
            title: _patternCopy(l10n).title,
            errorText: _formValidationError,
            child: FormBuilder(
              key: _patternDetailsFormKey,
              initialValue: _detailInitialValues(_patternDetailFieldNames),
              onChanged: () {
                _syncDetailForm(
                  _patternDetailsFormKey,
                  _patternDetailFieldNames,
                );
                setRouteState(() {});
              },
              child: ListenableBuilder(
                listenable: Listenable.merge([
                  _patternNameController,
                  _patternDesignerController,
                  _patternPublisherController,
                ]),
                builder: (context, child) =>
                    _patternDetailFields(l10n, setRouteState),
              ),
            ),
          ),
        ),
      ),
    );
    if (!mounted) return;
    (_showPatternDetails ? _patternDetailsFocusNode : _craftDetailsFocusNode)
        .requestFocus();
  }

  Future<void> _openCraftDetails() async {
    final craft = _activeCraftType;
    if (craft == null) return;
    FocusManager.instance.primaryFocus?.unfocus();
    final l10n = AppLocalizations.of(context);
    final fieldNames = _craftDetailsPageFieldNames;
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    await Navigator.of(context).push<void>(
      MaterialPageRoute<void>(
        builder: (_) => StatefulBuilder(
          builder: (context, setRouteState) => ProjectComposerDetailPage(
            title: _craftDetailsTitle(l10n),
            errorText: _formValidationError,
            child: FormBuilder(
              key: _craftDetailsFormKey,
              initialValue: _detailInitialValues(fieldNames),
              onChanged: () {
                _syncDetailForm(
                  _craftDetailsFormKey,
                  fieldNames,
                );
                setRouteState(() {});
              },
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                spacing: spacing.sp4,
                children: [
                  if (_canEditSpecializedDetails &&
                      _activeCraftDetailFieldNames.isNotEmpty)
                    ProjectCraftFields(craft: craft),
                  CraftskyFormBuilderMultiSelectField<String>(
                    name: ProjectComposerFields.additionalCrafts,
                    label: l10n.projectComposerAdditionalCraftsLabel,
                    options: _selectOptions(
                      ProjectOptionCatalogs.craftTypes
                          .where((option) => option.value != craft)
                          .toList(),
                      includeCraftIcons: true,
                    ),
                    maxSelected: 10,
                    searchHintText:
                        l10n.projectComposerAdditionalCraftsSearchHint,
                    disabledText: l10n.projectComposerFieldDisabledLabel,
                    maxSelectedErrorText: l10n
                        .projectComposerMultiSelectMaxSelectedError(10),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
    if (mounted) _craftDetailsFocusNode.requestFocus();
  }

  List<ProjectPattern> get _patternEntries =>
      _detailFormValues[ProjectComposerFields.patterns]
          as List<ProjectPattern>? ??
      const [];

  late final _savedPatternsByIndex = <int, ProjectPattern>{
    for (final pattern
        in _initialFormValues[ProjectComposerFields.patterns]
                as List<ProjectPattern>? ??
            const <ProjectPattern>[])
      if (pattern.extraFields['existingIndex'] case final int index)
        index: pattern,
  };

  ProjectPattern? _savedPatternFor(ProjectPattern? pattern) =>
      _savedPatternsByIndex[pattern?.extraFields['existingIndex']];

  List<String> _patternHashtags(ProjectPattern pattern) => [
    ...hashtagsInText(pattern.name ?? ''),
    ...hashtagsInText(pattern.designer ?? ''),
    ...hashtagsInText(pattern.publisher ?? ''),
  ];

  List<String> _extraPatternHashtagPrefix(int index, String field) => [
    ..._hashtagPrefix(designer: true),
    ...hashtagsInText(_patternPublisherController.text),
    for (final pattern in _patternEntries.skip(1).take(index - 1))
      ..._patternHashtags(pattern),
    if (field != 'name') ...hashtagsInText(_patternEntries[index].name ?? ''),
    if (field == 'publisher')
      ...hashtagsInText(_patternEntries[index].designer ?? ''),
  ];

  void _writePatternEntries(List<ProjectPattern> entries) {
    setState(() {
      _detailFormValues[ProjectComposerFields.patterns] = entries;
      _formValidationError = null;
    });
  }

  void _removeFirstPattern() {
    final rest = _patternEntries.skip(1).toList();
    final next = rest.firstOrNull;
    if (_extraPatternKeys.isNotEmpty) _extraPatternKeys.removeAt(0);
    _writePatternEntries(rest);
    _patternNameController.text = _patternDisplayText(next?.name);
    _patternDesignerController.text = next?.designer ?? '';
    _patternPublisherController.text = next?.publisher ?? '';
    final firstFields = <String, Object?>{
      ProjectComposerFields.patternName: next?.name,
      ProjectComposerFields.patternDesigner: next?.designer,
      ProjectComposerFields.patternPublisher: next?.publisher,
      ProjectComposerFields.patternUrl: next?.url,
      ProjectComposerFields.patternDifficulty: next?.difficulty,
      ProjectComposerFields.patternSelfDrafted: next?.selfDrafted,
    };
    _detailFormValues.addAll(firstFields);
    _patternDetailsFormKey.currentState?.patchValue(firstFields);
  }

  Widget _patternDetailFields(
    AppLocalizations l10n,
    StateSetter setRouteState,
  ) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final copy = _patternCopy(l10n);

    Widget patternHeader(int number, VoidCallback onRemove) => Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (number > 1) const Divider(),
        Wrap(
          alignment: WrapAlignment.spaceBetween,
          crossAxisAlignment: WrapCrossAlignment.center,
          children: [
            Semantics(
              headingLevel: 2,
              child: Text(
                l10n.projectComposerPatternNumber(number),
                style: theme.textTheme.titleLarge,
              ),
            ),
            TextButton.icon(
              onPressed: onRemove,
              style: TextButton.styleFrom(
                foregroundColor: theme.colorScheme.error,
                iconColor: theme.colorScheme.error,
              ),
              icon: const Icon(CraftskyIconsBold.delete),
              label: Text(l10n.projectComposerRemovePattern),
            ),
          ],
        ),
      ],
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (_patternEntries.length > 1)
          patternHeader(1, () {
            _removeFirstPattern();
            setRouteState(() {});
          }),
        ProjectFacetField(
          name: ProjectComposerFields.patternName,
          key: const Key('project-composer-pattern-name-field'),
          editorKey: const Key('project-composer-pattern-name-editor'),
          label: copy.name,
          hintText: l10n.projectComposerPatternNameHint,
          controller: _patternNameController,
          limit: RecordTextLimits.patternText,
          allowedTokenKinds: const {ActiveFacetTokenKind.hashtag},
          precedingHashtags: _hashtagPrefix,
          textCapitalization: TextCapitalization.none,
          focusNode: _patternNameFocusNode,
          normalizeValue: _patternFormValue,
        ),
        SizedBox(height: spacing.sp4),
        ProjectFacetField(
          name: ProjectComposerFields.patternDesigner,
          limit: RecordTextLimits.patternText,
          key: const Key('project-composer-pattern-designer-field'),
          editorKey: const Key('project-composer-pattern-designer-editor'),
          label: l10n.projectComposerPatternDesignerLabel,
          hintText: l10n.projectComposerPatternDesignerHint,
          controller: _patternDesignerController,
          precedingHashtags: () => _hashtagPrefix(pattern: true),
          focusNode: _patternDesignerFocusNode,
          allowedTokenKinds: const {ActiveFacetTokenKind.mention},
        ),
        SizedBox(height: spacing.sp4),
        ProjectFacetField(
          name: ProjectComposerFields.patternPublisher,
          limit: RecordTextLimits.patternText,
          key: const Key('project-composer-pattern-publisher-field'),
          editorKey: const Key('project-composer-pattern-publisher-editor'),
          label: l10n.projectComposerPatternPublisherLabel,
          hintText: l10n.projectComposerPatternPublisherHint,
          controller: _patternPublisherController,
          precedingHashtags: () => _hashtagPrefix(designer: true),
          focusNode: _patternPublisherFocusNode,
          allowedTokenKinds: const {ActiveFacetTokenKind.mention},
        ),
        SizedBox(height: spacing.sp4),
        CraftskyFormBuilderTextField(
          key: const ValueKey(ProjectComposerFields.patternUrl),
          name: ProjectComposerFields.patternUrl,
          label: l10n.projectComposerPatternUrlLabel,
          hintText: l10n.projectComposerPatternUrlHint,
          keyboardType: TextInputType.url,
          textFieldKey: const Key('pattern-url-input'),
        ),
        SizedBox(height: spacing.sp4),
        CraftskyFormBuilderDropdownField<String>(
          key: const ValueKey(ProjectComposerFields.patternDifficulty),
          name: ProjectComposerFields.patternDifficulty,
          label: l10n.projectComposerPatternDifficultyLabel,
          options: _selectOptions(ProjectOptionCatalogs.patternDifficulties),
        ),
        SizedBox(height: spacing.sp4),
        FormBuilderField<bool>(
          key: const ValueKey(ProjectComposerFields.patternSelfDrafted),
          name: ProjectComposerFields.patternSelfDrafted,
          builder: (field) => SwitchListTile(
            key: const Key('project-composer-self-drafted-switch'),
            contentPadding: EdgeInsets.zero,
            title: Text(
              l10n.projectComposerPatternSelfDraftedTitle,
              style: const TextStyle(fontWeight: FontWeight.bold),
            ),
            subtitle: Text(l10n.projectComposerPatternSelfDraftedDescription),
            value: field.value ?? false,
            onChanged: field.didChange,
            activeTrackColor: theme.colorScheme.primary,
            activeThumbColor: theme.colorScheme.onPrimary,
          ),
        ),
        for (var index = 1; index < _patternEntries.length; index++) ...[
          SizedBox(height: spacing.sp4),
          patternHeader(index + 1, () {
            _extraPatternKeys.removeAt(index - 1);
            _writePatternEntries([..._patternEntries]..removeAt(index));
            setRouteState(() {});
          }),
          ProjectPatternEditor(
            key: _extraPatternKeys[index - 1],
            pattern: _patternEntries[index],
            savedPattern: _savedPatternFor(_patternEntries[index]),
            nameLabel: copy.name,
            precedingHashtags: (field) =>
                _extraPatternHashtagPrefix(index, field),
            onChanged: (pattern) => _writePatternEntries(
              [..._patternEntries]..[index] = pattern,
            ),
          ),
        ],
        if (_patternEntries.length < 5)
          Align(
            alignment: AlignmentDirectional.centerStart,
            child: TextButton.icon(
              onPressed: () {
                _extraPatternKeys.add(UniqueKey());
                _writePatternEntries([
                  if (_patternEntries.isEmpty) const ProjectPattern(),
                  ..._patternEntries,
                  const ProjectPattern(),
                ]);
                setRouteState(() {});
              },
              icon: const Icon(CraftskyIconsBold.add),
              label: Text(l10n.projectComposerAddPattern),
            ),
          ),
      ],
    );
  }

  Widget _projectFields({
    required AppLocalizations l10n,
    required SpacingTheme spacing,
    required ComposerImagesState imagesState,
    required LocalVideoSelection? selectedVideo,

    required bool controlsEnabled,
    required String? photoErrorText,
    required Future<void> Function()? onAddImages,
    required Future<void> Function()? onTakePhoto,
    required Future<void> Function() onAddVideo,
    required void Function(String imageId, String value) onAltTextChanged,
    required ValueChanged<String> onRemoveImage,
    required Future<void> Function(String imageId, ImageSource source)
    onReplaceUnavailable,
    required void Function(int fromIndex, int toIndex) onReorderImages,
  }) {
    final patternCopy = _patternCopy(l10n);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
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
            enabled: controlsEnabled,
            onAltTextChanged: (value) =>
                setState(() => _publishedEmbed.videoAlt = value),
            onRemove: () => setState(() => _publishedEmbed.remove()),
          )
        else if (selectedVideo != null)
          ComposerVideoAttachmentCard(
            selection: selectedVideo,
            enabled: controlsEnabled,
            onAltTextChanged: (value) {
              setState(() => _videoController.setAltText(value));
            },
            onRemove: () {
              setState(() {
                _usesStoredDraftVideo = false;
                _videoController.remove();
              });
            },
          )
        else
          ComposerImageAttachmentSection(
            onEditImage: _editImage,
            useDetailsRoute: true,
            imagesProvider: composerImagesProvider(_composerId),
            imagesState: imagesState,
            enabled: controlsEnabled,
            validationErrorText: photoErrorText,
            required: true,
            requiredLabel: l10n.projectComposerRequiredLabel,
            onAddImages: onAddImages,
            onTakePhoto: onTakePhoto,
            onAltTextChanged: onAltTextChanged,
            onRemove: onRemoveImage,
            onReplaceUnavailable: onReplaceUnavailable,
            onReorder: onReorderImages,
            supportsVideo:
                widget.scheduledPost == null &&
                _scheduleChoice == ScheduleChoice.now,
            onAddVideo: onAddVideo,
          ),
        SizedBox(height: spacing.sp6),
        CraftskyFormBuilderDropdownField<String>(
          name: ProjectComposerFields.craftType,
          label: l10n.projectComposerCraftTypeLabel,
          required: true,
          requiredLabel: l10n.projectComposerRequiredLabel,
          options: _selectOptions(
            ref.watch(preferredProjectCraftsProvider),
            includeCraftIcons: true,
            alphabetize: false,
            retained: _activeCraftType,
          ),
          enabled: controlsEnabled,
          validator: (value) =>
              value == null ? l10n.projectComposerCraftRequiredError : null,
          onChanged: _onCraftTypeChanged,
        ),
        SizedBox(height: spacing.sp4),
        ProjectDetailActionTile(
          key: const Key('project-composer-craft-details-action'),
          focusNode: _craftDetailsFocusNode,
          icon: CraftskyIcons.adjustments,
          title: _craftDetailsTitle(l10n),
          subtitle:
              _craftItemSummary ??
              _detailSummary(
                _craftDetailsPageFieldNames,
                empty:
                    _canEditSpecializedDetails &&
                        _activeCraftDetailFieldNames.isNotEmpty
                    ? l10n.projectComposerCraftDetailsDescription
                    : l10n.projectComposerAdditionalCraftsDescription,
                l10n: l10n,
              ),
          errorText: _formValidationError,
          enabled: controlsEnabled && _activeCraftType != null,
          onTap: _openCraftDetails,
        ),
        if (_showPatternDetails)
          ProjectDetailActionTile(
            key: const Key('project-composer-pattern-details-action'),
            focusNode: _patternDetailsFocusNode,
            icon: CraftskyIcons.pattern,
            title: patternCopy.title,
            subtitle: _detailSummary(
              _patternDetailFieldNames,
              empty: patternCopy.description,
              l10n: l10n,
            ),
            enabled: controlsEnabled,
            onTap: _openPatternDetails,
          ),
        SizedBox(height: spacing.sp4),
        CraftskyFormBuilderDropdownField<String>(
          name: ProjectComposerFields.status,
          label: l10n.projectComposerStatusLabel,
          required: true,
          requiredLabel: l10n.projectComposerRequiredLabel,
          options: _selectOptions(
            ProjectOptionCatalogs.statuses,
            retained:
                _initialFormValues[ProjectComposerFields.status] as String?,
          ),
          enabled: controlsEnabled,
          validator: (value) => value == null || value.trim().isEmpty
              ? l10n.projectComposerStatusRequiredError
              : null,
        ),
        SizedBox(height: spacing.sp4),
        CraftskyFormBuilderTextField(
          name: ProjectComposerFields.title,
          maxLength: RecordTextLimits.projectTitle.graphemes,
          maxBytes: RecordTextLimits.projectTitle.bytes,
          label: l10n.projectComposerProjectTitleLabel,
          textCapitalization: TextCapitalization.words,
          hintText: l10n.projectComposerProjectTitleHint,
          textFieldKey: const Key('project-title-input'),
          enabled: controlsEnabled,
        ),
      ],
    );
  }

  Widget _descriptionField({
    required AppLocalizations l10n,
    required bool controlsEnabled,
    required String? bodyErrorText,
  }) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        FacetAutocompleteEditor(
          maxLength: RecordTextLimits.post.graphemes,
          maxBytes: RecordTextLimits.post.bytes,
          key: const Key('project-composer-body-editor'),
          label: l10n.projectComposerDescriptionLabel,
          required: true,
          requiredLabel: l10n.projectComposerRequiredLabel,
          hintText:
              '${l10n.projectComposerDescriptionHint}\n\n'
              '${l10n.composerBodyInputHelp}',
          controller: _bodyController,
          focusNode: _bodyFocusNode,
          minLines: 6,
          maxLines: 12,
          enabled: controlsEnabled,
          textInputAction: TextInputAction.newline,
          keyboardType: TextInputType.multiline,
          errorText: bodyErrorText,
          helperAlignment: AlignmentDirectional.centerEnd,
          onChanged: (value) => setState(() => _bodyText = value),
        ),
      ],
    );
  }

  void _onCraftTypeChanged(String? value) {
    _craftChosen = true;
    if (value == _activeCraftType) return;
    final selected = _detailFormValues[ProjectComposerFields.additionalCrafts];
    final previousFields = _activeCraftDetailFieldNames;
    final previousCraft = _activeCraftType;
    final clearedDetails = _hasAnyDetail(previousFields);
    setState(() {
      _activeCraftType = value;
      _formValidationError = null;
      if (selected is Iterable) {
        _detailFormValues[ProjectComposerFields.additionalCrafts] = selected
            .whereType<String>()
            .where((craft) => craft != value)
            .toList();
      }
      for (final name in previousFields) {
        final saved = _detailFormValues[name];
        if (previousCraft != null &&
            name == ProjectDetailFields.formName(previousCraft) &&
            saved is Map<String, Object?>) {
          // Keep prior details available if the user returns to this craft.
          final fields = ProjectDetailFields.authoringForCraft(previousCraft);
          _detailFormValues[name] = {...saved}
            ..removeWhere((key, _) => fields.any((field) => field.key == key));
        } else {
          _detailFormValues[name] = null;
        }
      }
    });
    if (clearedDetails) {
      context.showInfo(
        AppLocalizations.of(context).projectComposerCraftCleared,
      );
    }
  }

  static String _patternDisplayText(Object? value) {
    if (value is! String || value.trim().isEmpty) return '';
    return value;
  }

  String? _patternFormValue(String value) {
    final first = _patternEntries.firstOrNull;
    if (value == _savedPatternFor(first)?.name || value == first?.name) {
      return value;
    }
    final trimmed = value.trim();
    if (trimmed.isEmpty || trimmed == '#') return null;
    return trimmed;
  }

  Future<void> _savePublishedProject(
    PostEditOrigin edit,
    Project project,
    ComposerImagesState images,
    String bodyText,
  ) async {
    FocusScope.of(context).unfocus();
    await _runSubmission(edit.ownership, () async {
      final operation = ActiveAccountOperation.fromWidget(ref, edit.ownership)
        ..requireCurrent();
      final initialPayload = buildProjectComposerPayload(
        formValues: _initialFormValues,
        originalFormValues: _initialFormValues,
      ).project!;
      final initial = restorePublishedProjectEntries(
        seed: edit.seed,
        payload: initialPayload,
        formValues: _initialFormValues,
      );
      final current = restorePublishedProjectEntries(
        seed: edit.seed,
        payload: project,
        formValues: _combinedFormValues(saved: true),
      );
      final args = await buildProjectComposerSubmitArguments(
        text: bodyText,
        langs: _languages!.values,
        project: current,
        imagesState: images,
        generateFacets: ref.read(facetGeneratorProvider).generate,
        existingText: edit.seed.editable['text']! as String,
        existingFacets: (edit.seed.editable['facets'] as List?)
            ?.map((value) => Map<String, dynamic>.from(value as Map))
            .toList(),
        existingProject: ProjectMapper.fromMap(
          Map<String, dynamic>.from(edit.seed.editable['project']! as Map),
        ),
        materializeImagesFromState: false,
        preserveSavedFacets: true,
      );
      operation.requireCurrent();
      final changes = <String, Object?>{...?_publishedEmbed?.changes};
      final projectChanges = buildPublishedProjectPatch(
        initial: initial,
        current: args.project,
      );
      if (projectChanges.isNotEmpty) changes['project'] = projectChanges;
      if (_bodyText != _initialBodyText) {
        changes['text'] = args.text;
        changes['facets'] = args.facets;
      }
      if (_sponsored != _initialSponsored) changes['sponsored'] = _sponsored;
      if (!listEquals(_languages?.values, _initialLanguages)) {
        changes['langs'] = _languages!.values;
      }
      if (_draftMediaChanged(images)) {
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
      await submitPublishedPostEdit(
        ref: ref,
        edit: edit,
        changes: changes,
        selectedVideo: _videoController.selection,
        prepareVideo: widget.prepareVideoProof,
        onRetry: (retry) =>
            _retryPublishedEdit = () => _runSubmission(edit.ownership, retry),
        onAccepted: (result) {
          _submissionSucceeded = true;
          if (!mounted) return;
          _pendingPublishedDraftPost = result.post;
          context.showInfo(AppLocalizations.of(context).postEditSuccess);
        },
        onProgress: (progress) {
          if (mounted) setState(() => _videoProgress = progress);
        },
        onVideo: (coordinator, {required active}) {
          if (active) {
            _videoPublication = coordinator;
            _videoPublicationOwner = edit.ownership;
          } else if (identical(_videoPublication, coordinator)) {
            _videoPublication = null;
            _videoPublicationOwner = null;
            if (mounted) setState(() => _videoProgress = null);
          }
        },
      );
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

  Future<void> _submitProject({required String trimmedBody}) async {
    if (!ref.read(composerImagesProvider(_composerId)).canSubmitImages()) {
      return;
    }
    setState(() {
      _attemptedSubmit = true;
      _formValidationError = null;
    });
    final form = _formKey.currentState;
    if (form == null) return;
    final imagesState = ref.read(composerImagesProvider(_composerId));
    final hasRequiredBody = trimmedBody.isNotEmpty;
    final hasRequiredPhoto =
        imagesState.images.isNotEmpty ||
        _videoController.selection != null ||
        _publishedEmbed?.hasRetainedVideo == true;
    final isFormValid = form.saveAndValidate(
      autoScrollWhenFocusOnInvalid: hasRequiredPhoto,
    );
    final isBodyLengthValid = RecordTextLimits.post.accepts(_bodyText);
    if (!isFormValid ||
        !hasRequiredBody ||
        !hasRequiredPhoto ||
        !isBodyLengthValid) {
      if (!hasRequiredPhoto) {
        await _scrollController.animateTo(
          0,
          duration: const Duration(milliseconds: 250),
          curve: Curves.easeOut,
        );
      } else if (isFormValid) {
        _bodyFocusNode.requestFocus();
        if (_bodyFocusNode.context case final bodyContext?) {
          await Scrollable.ensureVisible(bodyContext);
        }
      }
      return;
    }

    final payload = buildProjectComposerPayload(
      formValues: _combinedFormValues(saved: true),
      savedFirstPattern: _savedPatternFor(_patternEntries.firstOrNull),
      originalFormValues: widget.publishedEdit == null
          ? null
          : _initialFormValues,
    );
    final project = payload.project;
    if (project == null) {
      setState(() {
        _formValidationError =
            payload.errors.any(
              (error) =>
                  error.code == ProjectComposerValidationCode.invalidGauge,
            )
            ? AppLocalizations.of(context).projectComposerGaugeInvalidError
            : payload.errors.any(
                (error) => error.code == ProjectComposerValidationCode.tooLong,
              )
            ? AppLocalizations.of(context).textInputTooLong
            : payload.errors.any(
                (error) =>
                    error.code == ProjectComposerValidationCode.invalidDetail,
              )
            ? AppLocalizations.of(context).projectComposerInvalidCraftDetails
            : payload.errors.any(
                (error) =>
                    error.code == ProjectComposerValidationCode.tooManyValues,
              )
            ? AppLocalizations.of(context)
                  .projectComposerMultiSelectMaxSelectedError(5)
            : null;
      });
      if (_formValidationError != null) {
        final field = payload.errors.first.fieldName;
        if (field == ProjectComposerFields.patterns ||
            _patternDetailFieldNames.contains(field)) {
          await _openPatternDetails();
        } else if (_activeCraftType != null &&
            _craftDetailsPageFieldNames.contains(field)) {
          await _openCraftDetails();
        } else {
          context.showError(_formValidationError!);
        }
      }
      return;
    }

    if (imagesState.hasImagesMissingAltText ||
        (_videoController.selection?.altText.trim().isEmpty ?? false)) {
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

    if (widget.publishedEdit case final edit?) {
      await _savePublishedProject(edit, project, imagesState, _bodyText);
      return;
    }

    final args = await buildProjectComposerSubmitArguments(
      text: _bodyText,
      langs: _languages!.values,
      project: project,
      imagesState: imagesState,
      generateFacets: ref.read(facetGeneratorProvider).generate,
      existingText: widget.scheduledPost?.payload['text'] as String?,
      existingFacets: _scheduledProjectFacets('facets'),
      existingProject: _existingScheduledProject,
      materializeImagesFromState: false,
      preserveSavedFacets: widget.scheduledPost != null,
    );
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
        await _publishExistingProjectNow(args);
        return;
      }

      if (_scheduleChoice == ScheduleChoice.later) {
        await _scheduleProject(args: args, imagesState: imagesState);
        return;
      }

      if (_videoController.selection case final selectedVideo?) {
        await _publishVideo(
          selectedVideo: selectedVideo,
          owner: submissionOwner,
          text: args.text,
          project: args.project,
          facets: args.facets,
        );
        return;
      }

      final images = await _mediaUploader.materializeImmediate(
        composerId: _composerId,
        images: imagesState.images,
        ownershipIsCurrent: () =>
            _submissionOwnershipIsCurrent(submissionOwner),
        upload: ({required bytes, required mimeType, required cancelToken}) =>
            uploadClient!.uploadImage(
              bytes: bytes,
              mimeType: mimeType,
              cancelToken: cancelToken,
            ),
      );
      final created = await ref
          .read(createPostProvider.notifier)
          .create(
            text: args.text,
            langs: args.langs,
            sponsored: _sponsored,
            reply: args.reply,
            project: args.project,
            images: images,
            facets: args.facets,
            ownership: submissionOwner,
          );
      _submissionSucceeded = created != null;
    });
  }

  Future<void> _publishVideo({
    required LocalVideoSelection selectedVideo,
    required ActiveAccountLease? owner,
    required String text,
    required Project project,
    required List<Map<String, dynamic>>? facets,
  }) async {
    if (owner == null) {
      throw DiagnosticStateError('Video publication requires an owner');
    }
    if (widget.prepareVideoProof case final prepare?) {
      final video = await prepare(selectedVideo);
      final created = await ref
          .read(createPostProvider.notifier)
          .create(
            text: text,
            langs: _languages!.values,
            sponsored: _sponsored,
            project: project,
            video: video,
            facets: facets,
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
              text: text,
              langs: _languages!.values,
              sponsored: _sponsored,
              project: project,
              video: video,
              facets: facets,
              ownership: owner,
              allowVideoBlobRecovery: allowBlobRecovery,
            );
        if (created == null) throw const ComposerSubmissionFailure();
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
      }
    }
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
      accountDid: submissionOwner?.session.account.did.value,
      kind: ComposerSubmissionKind.project,
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
      _formKey.currentState?.save();
      return saveComposerSnapshot(
        container: ProviderScope.containerOf(context, listen: false),
        owner: owner,
        request: _projectDraftWriteRequest(
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

  List<Map<String, dynamic>>? _scheduledProjectFacets(String key) {
    final value = widget.scheduledPost?.payload[key];
    if (value is! List) return null;
    return [
      for (final item in value)
        if (item is Map<String, dynamic>) item,
    ];
  }

  Project? _readScheduledProject() {
    final value = widget.scheduledPost?.payload['project'];
    if (value is! Map<String, dynamic>) return null;
    final project = ProjectMapper.fromMap(value);
    final common = project.common;
    if (common.patterns == null && common.pattern == null) return project;
    return project.copyWith.common(
      pattern: null,
      patterns: [
        for (final (index, pattern) in common.effectivePatterns.indexed)
          pattern.copyWith(
            extraFields: {...pattern.extraFields, 'existingIndex': index},
          ),
      ],
    );
  }

  String _projectLocalTimeLabel(BuildContext context, DateTime value) {
    final l10n = AppLocalizations.of(context);
    final localizations = MaterialLocalizations.of(context);
    return l10n.scheduledPostLocalTime(
      localizations.formatMediumDate(value),
      localizations.formatTimeOfDay(TimeOfDay.fromDateTime(value)),
      value.timeZoneName,
      _projectOffsetLabel(value.timeZoneOffset),
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

  Map<String, dynamic> _scheduledProjectPayload(
    ProjectComposerSubmitArguments args,
    List<Map<String, dynamic>> media,
  ) {
    final project = args.project.toCreateMap();
    final common = project['common'] as Map<String, dynamic>;
    if (common['patterns'] case final List<dynamic> patterns) {
      for (final pattern in patterns.cast<Map<String, dynamic>>()) {
        pattern.remove('existingIndex');
      }
    }
    return {
      'kind': 'project',
      'text': args.text,
      'langs': args.langs,
      'sponsored': _sponsored,
      'facets': ?args.facets,
      'project': project,
      'media': media,
    };
  }

  Future<void> _scheduleProject({
    required ProjectComposerSubmitArguments args,
    required ComposerImagesState imagesState,
  }) async {
    if (ref.read(composerImagesProvider(_composerId)).hasImageEditsInFlight) {
      return;
    }
    final owner = _captureScheduledOperationOwner();
    final scheduledAt = _scheduledAtLocal;
    if (owner == null || scheduledAt == null) return;
    final l10n = AppLocalizations.of(context);
    await _runScheduledWrite(
      owner,
      errorMessage: l10n.scheduledProjectSaveError,
      successMessage: l10n.scheduledProjectSaved,
      write: (submission) => submission.save(
        operationId: _composerId,
        scheduledAt: scheduledAt,
        existing: widget.scheduledPost,
        images: imagesState.images,
        onProgress: _scheduleProgress,
        payload: (_, media) async => _scheduledProjectPayload(args, media),
      ),
    );
  }

  Future<void> _publishExistingProjectNow(
    ProjectComposerSubmitArguments args,
  ) async {
    final existing = widget.scheduledPost;
    if (ref.read(composerImagesProvider(_composerId)).hasImageEditsInFlight) {
      return;
    }
    final owner = _captureScheduledOperationOwner();
    if (existing == null || owner == null) return;
    final imagesState = ref.read(composerImagesProvider(_composerId));
    await _runScheduledWrite(
      owner,
      errorMessage: AppLocalizations.of(context).scheduledProjectNowError,
      write: (submission) => submission.publishNow(
        existing: existing,
        images: imagesState.images,
        onProgress: _scheduleProgress,
        payload: (_, media) async => _scheduledProjectPayload(args, media),
      ),
    );
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
      if (owner == null) throw DiagnosticStateError('account unavailable');
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
    if (ref.read(composerImagesProvider(_composerId)).hasImageEditsInFlight) {
      return;
    }
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
