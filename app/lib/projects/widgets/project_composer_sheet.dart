import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/drafts/composer/composer_draft_session.dart';
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/composer/draft_schedule_restoration.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/draft_save_controller.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/drafts/providers/local_post_drafts_provider.dart';
import 'package:craftsky_app/drafts/widgets/draft_close_dialog.dart';
import 'package:craftsky_app/feed/composer/composer_media_uploader.dart';
import 'package:craftsky_app/feed/composer/composer_submission_coordinator.dart';
import 'package:craftsky_app/feed/composer/composer_video_publication.dart';
import 'package:craftsky_app/feed/composer/submission_screen_awake.dart';
import 'package:craftsky_app/feed/composer/video_failure_message.dart';
import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/feed/providers/image_picker_existing_video.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/feed/providers/video_service_client_provider.dart';
import 'package:craftsky_app/feed/providers/video_upload_feature_provider.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/feed/widgets/composer_metadata_controls.dart';
import 'package:craftsky_app/feed/widgets/composer_video_attachment_card.dart';
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
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/providers/preferred_project_crafts_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_detail_page.dart';
import 'package:craftsky_app/projects/widgets/project_facet_field.dart';
import 'package:craftsky_app/projects/widgets/project_materials_field.dart';
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
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/rich_text/facet_autocomplete_controller.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/shared/rich_text/widgets/facet_autocomplete_editor.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_text_field.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
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
  LocalPostDraftSeed? draftSeed,
  ActiveAccountLease? draftOwner,
}) {
  return responsiveModalNavigator(context).push<Post?>(
    FullscreenModalRoute<Post?>(
      builder: (_) => ProjectComposerSheet(
        scheduledPost: scheduledPost,
        scheduledOwner: scheduledOwner,
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
    this.draftSeed,
    this.draftOwner,
    this.videoController,
    this.prepareVideoProof,
  }) : assert(
         scheduledPost == null || draftSeed == null,
         'Scheduled and local draft edits are mutually exclusive',
       );

  static const maxCharacters = 2000;

  final String? composerId;
  final ScheduledPostDetail? scheduledPost;
  final ActiveAccountLease? scheduledOwner;
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
  late final Map<String, dynamic> _detailFormValues;
  static const List<String> _patternDetailFieldNames = [
    ProjectComposerFields.patternDesigner,
    ProjectComposerFields.patternPublisher,
    ProjectComposerFields.patternUrl,
    ProjectComposerFields.patternDifficulty,
    ProjectComposerFields.patternSelfDrafted,
  ];
  static const List<String> _commonDetailFieldNames = [
    ProjectComposerFields.materials,
    ProjectComposerFields.colours,
    ProjectComposerFields.designTags,
  ];
  static const List<String> _craftDetailFieldNames = [
    ProjectComposerFields.sewingProjectType,
    ProjectComposerFields.sewingProjectSubtype,
    ProjectComposerFields.sewingSizeMade,
    ProjectComposerFields.sewingFitNotes,
    ProjectComposerFields.knittingProjectType,
    ProjectComposerFields.knittingProjectSubtype,
    ProjectComposerFields.knittingYarnWeight,
    ProjectComposerFields.knittingNeedleSize,
    ProjectComposerFields.knittingGaugeStitches,
    ProjectComposerFields.knittingGaugeRows,
    ProjectComposerFields.knittingGaugeMeasurement,
    ProjectComposerFields.knittingGaugeUnit,
    ProjectComposerFields.knittingFinishedSize,
    ProjectComposerFields.crochetProjectType,
    ProjectComposerFields.crochetProjectSubtype,
    ProjectComposerFields.crochetYarnWeight,
    ProjectComposerFields.crochetHookSize,
    ProjectComposerFields.crochetGaugeStitches,
    ProjectComposerFields.crochetGaugeRows,
    ProjectComposerFields.crochetGaugeMeasurement,
    ProjectComposerFields.crochetGaugeUnit,
    ProjectComposerFields.crochetFinishedSize,
    ProjectComposerFields.quiltingProjectType,
    ProjectComposerFields.quiltingProjectSubtype,
    ProjectComposerFields.quiltingSize,
    ProjectComposerFields.quiltingPiecingTechnique,
    ProjectComposerFields.quiltingMethod,
  ];

  final _formKey = GlobalKey<FormBuilderState>();
  final _patternDetailsFormKey = GlobalKey<FormBuilderState>();
  final _commonDetailsFormKey = GlobalKey<FormBuilderState>();
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
  final _commonDetailsFocusNode = FocusNode(
    debugLabel: 'projectCommonDetailsAction',
  );
  final _craftDetailsFocusNode = FocusNode(
    debugLabel: 'projectCraftDetailsAction',
  );
  final _patternNameController = FacetTextEditingController(text: '#');
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
  String? _sewingProjectType;
  String? _knittingProjectType;
  String? _crochetProjectType;
  String? _quiltingProjectType;
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
  String _initialBodyText = '';
  List<String>? _initialLanguages;
  ScheduleChoice _initialScheduleChoice = ScheduleChoice.now;
  DateTime? _initialScheduledAtLocal;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _origin = ComposerDraftSession(widget.draftSeed?.draft);
    _scheduledOwner = widget.scheduledOwner;
    final draftContent = widget.draftSeed?.draft.content;
    final project = widget.scheduledPost?.payload['project'];
    _initialFormValues = draftContent is ProjectDraftContent
        ? const ProjectDraftSnapshotAdapter().decodeKnownFields(
            Map<String, dynamic>.from(draftContent.knownProjectFieldValues),
          )
        : hydrateScheduledProjectComposer(
            project is Map<String, dynamic> ? project : null,
          );
    _detailFormValues = {
      for (final name in [
        ..._patternDetailFieldNames,
        ..._commonDetailFieldNames,
        ..._craftDetailFieldNames,
      ])
        if (_initialFormValues.containsKey(name))
          name: _initialFormValues[name],
    };
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
    _sewingProjectType =
        _initialFormValues[ProjectComposerFields.sewingProjectType] as String?;
    _knittingProjectType =
        _initialFormValues[ProjectComposerFields.knittingProjectType]
            as String?;
    _crochetProjectType =
        _initialFormValues[ProjectComposerFields.crochetProjectType] as String?;
    _quiltingProjectType =
        _initialFormValues[ProjectComposerFields.quiltingProjectType]
            as String?;
    _unsavedGuard = ref.read(unsavedWorkGuardProvider);
    _composerId = widget.composerId ?? const Uuid().v4();
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
    _commonDetailsFocusNode.dispose();
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
    final createState = ref.watch(createPostProvider);
    final videoUploadsEnabled = ref.watch(videoUploadsEnabledProvider);
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
        !_isSavingDraft &&
        !createState.isLoading &&
        !_isSubmitting &&
        !_isScheduling &&
        !_isLoadingScheduledMedia &&
        !_scheduledMediaLoadFailed;
    final trimmedBody = _bodyText.trim();
    final tooLong = !RecordTextLimits.post.accepts(_bodyText);
    final canSubmit =
        !_isSavingDraft &&
        !_isScheduling &&
        !createState.isLoading &&
        !_isSubmitting &&
        !_isLoadingScheduledMedia &&
        !_scheduledMediaLoadFailed &&
        _languages != null &&
        (selectedVideo != null || imagesState.canSubmitImages()) &&
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
        _attemptedSubmit && imagesState.images.isEmpty && selectedVideo == null
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
                if (widget.scheduledPost == null)
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
                          FormBuilder(
                            key: _formKey,
                            initialValue: _initialFormValues,
                            onChanged: () {
                              if (mounted) setState(() {});
                            },
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.stretch,
                              children: [
                                _pageOne(
                                  l10n: l10n,
                                  theme: theme,
                                  spacing: spacing,
                                  imagesState: imagesState,
                                  selectedVideo: selectedVideo,
                                  videoUploadsEnabled: videoUploadsEnabled,
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
                                _pageThree(
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
                          Text(
                            l10n.projectComposerMoreDetailsHeading,
                            style: theme.textTheme.titleLarge?.copyWith(
                              fontWeight: FontWeight.w900,
                            ),
                          ),
                          SizedBox(height: spacing.sp2),
                          Text(
                            l10n.projectComposerMoreDetailsPrompt,
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                          SizedBox(height: spacing.sp2),
                          ProjectDetailActionTile(
                            key: const Key(
                              'project-composer-pattern-details-action',
                            ),
                            focusNode: _patternDetailsFocusNode,
                            icon: CraftskyIcons.pattern,
                            title: l10n.projectComposerPatternDetailsTitle,
                            subtitle: _detailSummary(
                              _patternDetailFieldNames,
                              empty:
                                  l10n.projectComposerPatternDetailsDescription,
                              l10n: l10n,
                            ),
                            enabled: controlsEnabled,
                            onTap: _openPatternDetails,
                          ),
                          ProjectDetailActionTile(
                            key: const Key(
                              'project-composer-common-details-action',
                            ),
                            focusNode: _commonDetailsFocusNode,
                            icon: CraftskyIcons.palette,
                            title: l10n.projectComposerCommonDetailsTitle,
                            subtitle: _detailSummary(
                              _commonDetailFieldNames,
                              empty:
                                  l10n.projectComposerCommonDetailsDescription,
                              l10n: l10n,
                            ),
                            enabled: controlsEnabled,
                            onTap: _openCommonDetails,
                          ),
                          if (_craftDetailsTitle(l10n)
                              case final craftDetailsTitle?)
                            ProjectDetailActionTile(
                              key: const Key(
                                'project-composer-craft-details-action',
                              ),
                              focusNode: _craftDetailsFocusNode,
                              icon: CraftskyIcons.adjustments,
                              title: craftDetailsTitle,
                              subtitle: _detailSummary(
                                _activeCraftDetailFieldNames,
                                empty:
                                    l10n.projectComposerCraftDetailsDescription,
                                l10n: l10n,
                              ),
                              errorText: _formValidationError,
                              enabled: controlsEnabled,
                              onTap: _openCraftDetails,
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
                        child: ChunkyButton(
                          key: const Key('project-composer-primary-action'),
                          focusNode: _primaryActionFocusNode,
                          onPressed: canSubmit
                              ? () => _submitProject(trimmedBody: trimmedBody)
                              : null,
                          child: Text(
                            _scheduleChoice == ScheduleChoice.later
                                ? l10n.scheduledPostAction
                                : l10n.postComposeSubmit,
                          ),
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

  Future<bool> _confirmProjectClose(ComposerImagesState imagesState) async {
    if (widget.scheduledPost != null) return await _confirmDiscard();
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

  Future<void> _saveProjectDraft(ComposerImagesState imagesState) async {
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
    return ProjectComposerDraftState.hasDraft(
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
  }) {
    final selectOptions = [
      for (final option in options)
        CraftskySelectOption<String>(
          value: option.value,
          label: option.label,
          description: option.description,
          leadingBuilder: includeCraftIcons
              ? (_) => CraftIcon(craft: option.value)
              : null,
        ),
    ];
    return alphabetize
        ? alphabetizedSelectOptions(selectOptions)
        : selectOptions;
  }

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
        _detailFormValues[name] = values[name];
      }
      _formValidationError = null;
    });
  }

  bool _hasAnyDetail(Iterable<String> names) => names.any(
    (name) => switch (_detailFormValues[name]) {
      final String value => value.trim().isNotEmpty,
      final Iterable<Object?> values => values.isNotEmpty,
      final num value => value > 0,
      true => true,
      _ => false,
    },
  );

  String _detailSummary(
    Iterable<String> names, {
    required String empty,
    required AppLocalizations l10n,
  }) {
    final count = names.where((name) {
      final value = _detailFormValues[name];
      return switch (value) {
        final String text => text.trim().isNotEmpty,
        final Iterable<Object?> values => values.isNotEmpty,
        final num number => number > 0,
        true => true,
        _ => false,
      };
    }).length;
    return count == 0 ? empty : l10n.projectComposerDetailsAdded(count);
  }

  List<String> get _activeCraftDetailFieldNames => switch (_activeCraftType) {
    ProjectOptionCatalogs.sewingCraftToken =>
      _craftDetailFieldNames
          .where((name) => name.startsWith('sewing'))
          .toList(growable: false),
    ProjectOptionCatalogs.knittingCraftToken =>
      _craftDetailFieldNames
          .where((name) => name.startsWith('knitting'))
          .toList(growable: false),
    ProjectOptionCatalogs.crochetCraftToken =>
      _craftDetailFieldNames
          .where((name) => name.startsWith('crochet'))
          .toList(growable: false),
    ProjectOptionCatalogs.quiltingCraftToken =>
      _craftDetailFieldNames
          .where((name) => name.startsWith('quilting'))
          .toList(growable: false),
    _ => const [],
  };

  String? _craftDetailsTitle(AppLocalizations l10n) {
    if (_activeCraftDetailFieldNames.isEmpty) return null;
    final craft = ProjectOptionCatalogs.craftTypes
        .where((option) => option.value == _activeCraftType)
        .firstOrNull;
    return craft == null
        ? null
        : l10n.projectComposerCraftDetailsTitle(craft.label);
  }

  Future<void> _openPatternDetails() async {
    final l10n = AppLocalizations.of(context);
    await Navigator.of(context).push<void>(
      MaterialPageRoute<void>(
        builder: (_) => ProjectComposerDetailPage(
          title: l10n.projectComposerPatternDetailsTitle,
          child: FormBuilder(
            key: _patternDetailsFormKey,
            initialValue: _detailInitialValues(_patternDetailFieldNames),
            onChanged: () => _syncDetailForm(
              _patternDetailsFormKey,
              _patternDetailFieldNames,
            ),
            child: _patternDetailFields(l10n),
          ),
        ),
      ),
    );
    if (mounted) _patternDetailsFocusNode.requestFocus();
  }

  Future<void> _openCommonDetails() async {
    final l10n = AppLocalizations.of(context);
    await Navigator.of(context).push<void>(
      MaterialPageRoute<void>(
        builder: (_) => ProjectComposerDetailPage(
          title: l10n.projectComposerCommonDetailsTitle,
          child: FormBuilder(
            key: _commonDetailsFormKey,
            initialValue: _detailInitialValues(_commonDetailFieldNames),
            onChanged: () => _syncDetailForm(
              _commonDetailsFormKey,
              _commonDetailFieldNames,
            ),
            child: _pageTwo(
              l10n: l10n,
              spacing: Theme.of(context).extension<SpacingTheme>()!,
              controlsEnabled: true,
            ),
          ),
        ),
      ),
    );
    if (mounted) _commonDetailsFocusNode.requestFocus();
  }

  Future<void> _openCraftDetails() async {
    final l10n = AppLocalizations.of(context);
    final title = _craftDetailsTitle(l10n);
    if (title == null) return;
    await Navigator.of(context).push<void>(
      MaterialPageRoute<void>(
        builder: (_) => StatefulBuilder(
          builder: (context, setRouteState) => ProjectComposerDetailPage(
            title: title,
            errorText: _formValidationError,
            child: FormBuilder(
              key: _craftDetailsFormKey,
              initialValue: _detailInitialValues(
                _activeCraftDetailFieldNames,
              ),
              onChanged: () {
                _syncDetailForm(
                  _craftDetailsFormKey,
                  _activeCraftDetailFieldNames,
                );
                setRouteState(() {});
              },
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: _detailFields(
                  l10n,
                  Theme.of(context).extension<SpacingTheme>()!,
                  true,
                  routeSetState: setRouteState,
                ),
              ),
            ),
          ),
        ),
      ),
    );
    if (mounted) _craftDetailsFocusNode.requestFocus();
  }

  Widget _patternDetailFields(AppLocalizations l10n) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ProjectFacetField(
          name: ProjectComposerFields.patternDesigner,
          limit: RecordTextLimits.patternText,
          key: const Key('project-composer-pattern-designer-field'),
          editorKey: const Key('project-composer-pattern-designer-editor'),
          label: l10n.projectComposerPatternDesignerLabel,
          hintText: l10n.projectComposerPatternDesignerHint,
          controller: _patternDesignerController,
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
          focusNode: _patternPublisherFocusNode,
          allowedTokenKinds: const {ActiveFacetTokenKind.mention},
        ),
        SizedBox(height: spacing.sp4),
        CraftskyFormBuilderTextField(
          name: ProjectComposerFields.patternUrl,
          label: l10n.projectComposerPatternUrlLabel,
          hintText: l10n.projectComposerPatternUrlHint,
          keyboardType: TextInputType.url,
          textFieldKey: const Key('pattern-url-input'),
        ),
        SizedBox(height: spacing.sp4),
        CraftskyFormBuilderDropdownField<String>(
          name: ProjectComposerFields.patternDifficulty,
          label: l10n.projectComposerPatternDifficultyLabel,
          options: _selectOptions(
            ProjectOptionCatalogs.patternDifficulties,
            alphabetize: false,
          ),
        ),
        SizedBox(height: spacing.sp4),
        FormBuilderField<bool>(
          name: ProjectComposerFields.patternSelfDrafted,
          builder: (field) => SwitchListTile(
            key: const Key('project-composer-self-drafted-switch'),
            contentPadding: EdgeInsets.zero,
            title: Text(l10n.projectComposerPatternSelfDraftedTitle),
            subtitle: Text(l10n.projectComposerPatternSelfDraftedDescription),
            value: field.value ?? false,
            onChanged: field.didChange,
            activeTrackColor: theme.colorScheme.primary,
            activeThumbColor: theme.colorScheme.onPrimary,
          ),
        ),
      ],
    );
  }

  Widget _pageOne({
    required AppLocalizations l10n,
    required ThemeData theme,
    required SpacingTheme spacing,
    required ComposerImagesState imagesState,
    required LocalVideoSelection? selectedVideo,
    required bool videoUploadsEnabled,
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
        else if (selectedVideo != null)
          ComposerVideoAttachmentCard(
            selection: selectedVideo,
            enabled: controlsEnabled,
            onAltTextChanged: (value) {
              setState(() => _videoController.setAltText(value));
            },
            onReplace: () => unawaited(_selectVideo(replace: true)),
            onRemove: () {
              setState(() {
                _usesStoredDraftVideo = false;
                _videoController.remove();
              });
            },
          )
        else
          ComposerImageAttachmentSection(
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
                videoUploadsEnabled &&
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
          ),
          enabled: controlsEnabled,
          validator: (value) =>
              value == null ? l10n.projectComposerCraftRequiredError : null,
          onChanged: _onCraftTypeChanged,
        ),
        SizedBox(height: spacing.sp4),
        Text(
          l10n.projectComposerDetailsPrompt,
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        SizedBox(height: spacing.sp3),
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
        SizedBox(height: spacing.sp4),
        CraftskyFormBuilderDropdownField<String>(
          name: ProjectComposerFields.status,
          label: l10n.projectComposerStatusLabel,
          initialValue: ProjectOptionCatalogs.finishedStatusToken,
          options: _selectOptions(ProjectOptionCatalogs.statuses),
          enabled: controlsEnabled,
        ),
        SizedBox(height: spacing.sp4),
        ProjectFacetField(
          name: ProjectComposerFields.patternName,
          limit: RecordTextLimits.patternText,
          key: const Key('project-composer-pattern-name-field'),
          editorKey: const Key('project-composer-pattern-name-editor'),
          label: l10n.projectComposerPatternNameLabel,
          hintText: l10n.projectComposerPatternNameHint,
          controller: _patternNameController,
          textCapitalization: TextCapitalization.none,
          focusNode: _patternNameFocusNode,
          enabled: controlsEnabled,
          initialDisplayText: '#',
          allowedTokenKinds: const {ActiveFacetTokenKind.hashtag},
          normalizeValue: _patternFormValue,
          onChanged: _onPatternNameChanged,
        ),
      ],
    );
  }

  Widget _pageTwo({
    required AppLocalizations l10n,
    required SpacingTheme spacing,
    required bool controlsEnabled,
  }) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ProjectMaterialsField(
          name: ProjectComposerFields.materials,
          label: l10n.projectComposerMaterialsLabel,
          inputHintText: l10n.projectComposerMaterialsAddHint,
          addButtonLabel: l10n.projectComposerMaterialsAddAction,
          disabledText: l10n.projectComposerFieldDisabledLabel,
          maxSelectedErrorText: l10n.projectComposerMultiSelectMaxSelectedError(
            10,
          ),
          maxLengthErrorText: l10n.projectComposerMaterialsMaxLengthError(100),
          enabled: controlsEnabled,
        ),
        SizedBox(height: spacing.sp4),
        CraftskyFormBuilderMultiSelectField<String>(
          name: ProjectComposerFields.colours,
          label: l10n.projectComposerColoursLabel,
          options: _selectOptions(ProjectOptionCatalogs.colours),
          maxSelected: 10,
          searchHintText: l10n.projectComposerColoursSearchHint,
          disabledText: l10n.projectComposerFieldDisabledLabel,
          maxSelectedErrorText: l10n.projectComposerMultiSelectMaxSelectedError(
            10,
          ),
          enabled: controlsEnabled,
        ),
        SizedBox(height: spacing.sp4),
        CraftskyFormBuilderMultiSelectField<String>(
          name: ProjectComposerFields.designTags,
          label: l10n.projectComposerDesignTagsLabel,
          options: _selectOptions(ProjectOptionCatalogs.designTags),
          maxSelected: 10,
          searchHintText: l10n.projectComposerDesignTagsSearchHint,
          disabledText: l10n.projectComposerFieldDisabledLabel,
          maxSelectedErrorText: l10n.projectComposerMultiSelectMaxSelectedError(
            10,
          ),
          enabled: controlsEnabled,
        ),
      ],
    );
  }

  Widget _pageThree({
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
          hintText: l10n.projectComposerDescriptionHint,
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
    if (value == _activeCraftType) return;
    final clearedDetails = _hasAnyDetail(_craftDetailFieldNames);
    setState(() {
      _activeCraftType = value;
      _formValidationError = null;
      _sewingProjectType = null;
      _knittingProjectType = null;
      _crochetProjectType = null;
      _quiltingProjectType = null;
      for (final name in _craftDetailFieldNames) {
        _detailFormValues[name] = null;
      }
    });
    if (clearedDetails) {
      context.showInfo(
        AppLocalizations.of(context).projectComposerCraftCleared,
      );
    }
  }

  void _onPatternNameChanged(String value) {
    setState(() {});
  }

  static String _patternDisplayText(Object? value) {
    if (value is! String) return '#';
    final trimmed = value.trim();
    if (trimmed.isEmpty || trimmed == '#') return '#';
    return trimmed.startsWith('#') ? trimmed : '#$trimmed';
  }

  static String? _patternFormValue(String value) {
    final trimmed = value.trim();
    if (trimmed.isEmpty || trimmed == '#') return null;
    return trimmed;
  }

  List<Widget> _detailFields(
    AppLocalizations l10n,
    SpacingTheme spacing,
    bool controlsEnabled, {
    StateSetter? routeSetState,
  }) {
    return switch (_activeCraftType) {
      ProjectOptionCatalogs.sewingCraftToken => _sewingDetailFields(
        l10n,
        spacing,
        controlsEnabled,
        routeSetState,
      ),
      ProjectOptionCatalogs.knittingCraftToken => _knittingDetailFields(
        l10n,
        spacing,
        controlsEnabled,
        routeSetState,
      ),
      ProjectOptionCatalogs.crochetCraftToken => _crochetDetailFields(
        l10n,
        spacing,
        controlsEnabled,
        routeSetState,
      ),
      ProjectOptionCatalogs.quiltingCraftToken => _quiltingDetailFields(
        l10n,
        spacing,
        controlsEnabled,
        routeSetState,
      ),
      ProjectOptionCatalogs.embroideryCraftToken => const <Widget>[],
      null => [
        _SelectCraftTypeEmptyState(
          text: l10n.projectComposerSelectCraftTypeEmptyState,
        ),
      ],
      _ => [
        _SelectCraftTypeEmptyState(
          text: l10n.projectComposerSelectCraftTypeEmptyState,
        ),
      ],
    };
  }

  List<Widget> _sewingDetailFields(
    AppLocalizations l10n,
    SpacingTheme spacing,
    bool controlsEnabled,
    StateSetter? routeSetState,
  ) {
    final subtypeOptions = ProjectOptionCatalogs.projectSubtypesFor(
      craftToken: ProjectOptionCatalogs.sewingCraftToken,
      projectTypeToken: _sewingProjectType,
    );
    return [
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.sewingProjectType,
        label: l10n.projectComposerSewingProjectTypeLabel,
        options: _selectOptions(
          ProjectOptionCatalogs.projectTypesForCraft(
            ProjectOptionCatalogs.sewingCraftToken,
          ),
        ),
        enabled: controlsEnabled,
        onChanged: (value) {
          setState(() {
            _sewingProjectType = value;
            _craftDetailsFormKey
                .currentState
                ?.fields[ProjectComposerFields.sewingProjectSubtype]
                ?.didChange(null);
          });
          routeSetState?.call(() {});
        },
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.sewingProjectSubtype,
        label: l10n.projectComposerProjectSubtypeLabel,
        options: _selectOptions(subtypeOptions),
        enabled: controlsEnabled && subtypeOptions.isNotEmpty,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderTextField(
        name: ProjectComposerFields.sewingSizeMade,
        maxLength: RecordTextLimits.projectSize.graphemes,
        maxBytes: RecordTextLimits.projectSize.bytes,
        label: l10n.projectComposerSizeMadeLabel,
        hintText: l10n.projectComposerSizeMadeHint,
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderMultilineTextField(
        name: ProjectComposerFields.sewingFitNotes,
        maxLength: RecordTextLimits.fitNotes.graphemes,
        maxBytes: RecordTextLimits.fitNotes.bytes,
        label: l10n.projectComposerFitNotesLabel,
        hintText: l10n.projectComposerFitNotesHint,
        enabled: controlsEnabled,
      ),
    ];
  }

  List<Widget> _knittingDetailFields(
    AppLocalizations l10n,
    SpacingTheme spacing,
    bool controlsEnabled,
    StateSetter? routeSetState,
  ) {
    final subtypeOptions = ProjectOptionCatalogs.projectSubtypesFor(
      craftToken: ProjectOptionCatalogs.knittingCraftToken,
      projectTypeToken: _knittingProjectType,
    );
    return [
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.knittingProjectType,
        label: l10n.projectComposerKnittingProjectTypeLabel,
        options: _selectOptions(
          ProjectOptionCatalogs.projectTypesForCraft(
            ProjectOptionCatalogs.knittingCraftToken,
          ),
        ),
        enabled: controlsEnabled,
        onChanged: (value) {
          setState(() {
            _knittingProjectType = value;
            _craftDetailsFormKey
                .currentState
                ?.fields[ProjectComposerFields.knittingProjectSubtype]
                ?.didChange(null);
          });
          routeSetState?.call(() {});
        },
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.knittingProjectSubtype,
        label: l10n.projectComposerProjectSubtypeLabel,
        options: _selectOptions(subtypeOptions),
        enabled: controlsEnabled && subtypeOptions.isNotEmpty,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.knittingYarnWeight,
        label: l10n.projectComposerYarnWeightLabel,
        options: _selectOptions(
          ProjectOptionCatalogs.yarnWeights,
          alphabetize: false,
        ),
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.knittingNeedleSize,
        label: l10n.projectComposerNeedleSizeLabel,
        options: _selectOptions(
          ProjectOptionCatalogs.needleSizes,
          alphabetize: false,
        ),
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormNumberField(
        name: ProjectComposerFields.knittingGaugeStitches,
        label: l10n.projectComposerGaugeStitchesLabel,
        hintText: l10n.projectComposerGaugeStitchesHint,
        mode: CraftskyNumberInputMode.integer,
        textFieldKey: const Key('knitting-gauge-stitches-input'),
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormNumberField(
        name: ProjectComposerFields.knittingGaugeRows,
        label: l10n.projectComposerGaugeRowsLabel,
        hintText: l10n.projectComposerGaugeRowsHint,
        mode: CraftskyNumberInputMode.integer,
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormNumberField(
        name: ProjectComposerFields.knittingGaugeMeasurement,
        label: l10n.projectComposerGaugeMeasurementLabel,
        hintText: l10n.projectComposerGaugeMeasurementHint,
        mode: CraftskyNumberInputMode.integer,
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.knittingGaugeUnit,
        label: l10n.projectComposerGaugeUnitLabel,
        options: _selectOptions(
          ProjectOptionCatalogs.gaugeUnits,
          alphabetize: false,
        ),
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderTextField(
        name: ProjectComposerFields.knittingFinishedSize,
        maxLength: RecordTextLimits.projectSize.graphemes,
        maxBytes: RecordTextLimits.projectSize.bytes,
        label: l10n.projectComposerFinishedSizeLabel,
        hintText: l10n.projectComposerFinishedSizeHint,
        enabled: controlsEnabled,
      ),
    ];
  }

  List<Widget> _crochetDetailFields(
    AppLocalizations l10n,
    SpacingTheme spacing,
    bool controlsEnabled,
    StateSetter? routeSetState,
  ) {
    final subtypeOptions = ProjectOptionCatalogs.projectSubtypesFor(
      craftToken: ProjectOptionCatalogs.crochetCraftToken,
      projectTypeToken: _crochetProjectType,
    );
    return [
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.crochetProjectType,
        label: l10n.projectComposerCrochetProjectTypeLabel,
        options: _selectOptions(
          ProjectOptionCatalogs.projectTypesForCraft(
            ProjectOptionCatalogs.crochetCraftToken,
          ),
        ),
        enabled: controlsEnabled,
        onChanged: (value) {
          setState(() {
            _crochetProjectType = value;
            _craftDetailsFormKey
                .currentState
                ?.fields[ProjectComposerFields.crochetProjectSubtype]
                ?.didChange(null);
          });
          routeSetState?.call(() {});
        },
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.crochetProjectSubtype,
        label: l10n.projectComposerProjectSubtypeLabel,
        options: _selectOptions(subtypeOptions),
        enabled: controlsEnabled && subtypeOptions.isNotEmpty,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.crochetYarnWeight,
        label: l10n.projectComposerYarnWeightLabel,
        options: _selectOptions(
          ProjectOptionCatalogs.yarnWeights,
          alphabetize: false,
        ),
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.crochetHookSize,
        label: l10n.projectComposerHookSizeLabel,
        options: _selectOptions(
          ProjectOptionCatalogs.hookSizes,
          alphabetize: false,
        ),
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormNumberField(
        name: ProjectComposerFields.crochetGaugeStitches,
        label: l10n.projectComposerGaugeStitchesLabel,
        hintText: l10n.projectComposerGaugeStitchesHint,
        mode: CraftskyNumberInputMode.integer,
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormNumberField(
        name: ProjectComposerFields.crochetGaugeRows,
        label: l10n.projectComposerGaugeRowsLabel,
        hintText: l10n.projectComposerGaugeRowsHint,
        mode: CraftskyNumberInputMode.integer,
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormNumberField(
        name: ProjectComposerFields.crochetGaugeMeasurement,
        label: l10n.projectComposerGaugeMeasurementLabel,
        hintText: l10n.projectComposerGaugeMeasurementHint,
        mode: CraftskyNumberInputMode.integer,
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.crochetGaugeUnit,
        label: l10n.projectComposerGaugeUnitLabel,
        options: _selectOptions(
          ProjectOptionCatalogs.gaugeUnits,
          alphabetize: false,
        ),
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderTextField(
        name: ProjectComposerFields.crochetFinishedSize,
        maxLength: RecordTextLimits.projectSize.graphemes,
        maxBytes: RecordTextLimits.projectSize.bytes,
        label: l10n.projectComposerFinishedSizeLabel,
        hintText: l10n.projectComposerFinishedSizeHint,
        enabled: controlsEnabled,
      ),
    ];
  }

  List<Widget> _quiltingDetailFields(
    AppLocalizations l10n,
    SpacingTheme spacing,
    bool controlsEnabled,
    StateSetter? routeSetState,
  ) {
    final subtypeOptions = ProjectOptionCatalogs.projectSubtypesFor(
      craftToken: ProjectOptionCatalogs.quiltingCraftToken,
      projectTypeToken: _quiltingProjectType,
    );
    return [
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.quiltingProjectType,
        label: l10n.projectComposerQuiltingProjectTypeLabel,
        options: _selectOptions(
          ProjectOptionCatalogs.projectTypesForCraft(
            ProjectOptionCatalogs.quiltingCraftToken,
          ),
        ),
        enabled: controlsEnabled,
        onChanged: (value) {
          setState(() {
            _quiltingProjectType = value;
            _craftDetailsFormKey
                .currentState
                ?.fields[ProjectComposerFields.quiltingProjectSubtype]
                ?.didChange(null);
          });
          routeSetState?.call(() {});
        },
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.quiltingProjectSubtype,
        label: l10n.projectComposerProjectSubtypeLabel,
        options: _selectOptions(subtypeOptions),
        enabled: controlsEnabled && subtypeOptions.isNotEmpty,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderTextField(
        name: ProjectComposerFields.quiltingSize,
        maxLength: RecordTextLimits.projectSize.graphemes,
        maxBytes: RecordTextLimits.projectSize.bytes,
        label: l10n.projectComposerSizeLabel,
        hintText: l10n.projectComposerFinishedSizeHint,
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.quiltingPiecingTechnique,
        label: l10n.projectComposerPiecingTechniqueLabel,
        options: _selectOptions(
          ProjectOptionCatalogs.quiltingPiecingTechniques,
        ),
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.quiltingMethod,
        label: l10n.projectComposerQuiltingMethodLabel,
        options: _selectOptions(ProjectOptionCatalogs.quiltingMethods),
        enabled: controlsEnabled,
      ),
    ];
  }

  Future<void> _submitProject({required String trimmedBody}) async {
    setState(() {
      _attemptedSubmit = true;
      _formValidationError = null;
    });
    final form = _formKey.currentState;
    if (form == null) return;
    final isFormValid = form.saveAndValidate();
    final imagesState = ref.read(composerImagesProvider(_composerId));
    final hasRequiredBody = trimmedBody.isNotEmpty;
    final hasRequiredPhoto =
        imagesState.images.isNotEmpty || _videoController.selection != null;
    final isBodyLengthValid = RecordTextLimits.post.accepts(_bodyText);
    if (!isFormValid ||
        !hasRequiredBody ||
        !hasRequiredPhoto ||
        !isBodyLengthValid) {
      if (!hasRequiredPhoto || !isFormValid) {
        await _scrollController.animateTo(
          0,
          duration: const Duration(milliseconds: 250),
          curve: Curves.easeOut,
        );
      } else {
        _bodyFocusNode.requestFocus();
        if (_bodyFocusNode.context case final bodyContext?) {
          await Scrollable.ensureVisible(bodyContext);
        }
      }
      return;
    }

    final payload = buildProjectComposerPayload(
      formValues: _combinedFormValues(saved: true),
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
            : null;
      });
      if (_formValidationError != null) {
        final field = payload.errors.first.fieldName;
        if (field == ProjectComposerFields.patternName ||
            field == ProjectComposerFields.patternDesigner ||
            field == ProjectComposerFields.patternPublisher) {
          await _openPatternDetails();
        } else if (field == ProjectComposerFields.materials) {
          await _openCommonDetails();
        } else {
          await _openCraftDetails();
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

    final args = await buildProjectComposerSubmitArguments(
      text: trimmedBody,
      langs: _languages!.values,
      project: project,
      imagesState: imagesState,
      generateFacets: ref.read(facetGeneratorProvider).generate,
      existingText: widget.scheduledPost?.payload['text'] as String?,
      existingFacets: _scheduledProjectFacets('facets'),
      existingProject: _existingScheduledProject(),
      materializeImagesFromState: false,
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
        if (mounted) {
          final l10n = AppLocalizations.of(context);
          final retryOperation = _videoController.selection == null
              ? null
              : MessageAction(
                  label: l10n.retryButton,
                  onPressed: () => unawaited(
                    _runSubmission(submissionOwner, operation),
                  ),
                );
          context.showError(
            retryOperation == null
                ? l10n.postCreateError
                : videoPublicationFailureMessage(l10n, _videoFailure),
            action: retryOperation,
          );
        }
      },
    );
    final published = _pendingPublishedDraftPost;
    if (published != null) await WidgetsBinding.instance.endOfFrame;
    if (mounted && published != null && Navigator.of(context).canPop()) {
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
      return ref
          .read(draftSaveControllerProvider(owner).notifier)
          .save(
            _projectDraftWriteRequest(
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

  Project? _existingScheduledProject() {
    final value = widget.scheduledPost?.payload['project'];
    if (value is! Map<String, dynamic>) return null;
    return ProjectMapper.fromMap(value);
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
  ) => {
    'kind': 'project',
    'text': args.text,
    'langs': args.langs,
    'sponsored': _sponsored,
    'facets': ?args.facets,
    'project': args.project.toCreateMap(),
    'media': media,
  };

  Future<void> _scheduleProject({
    required ProjectComposerSubmitArguments args,
    required ComposerImagesState imagesState,
  }) async {
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

class _SelectCraftTypeEmptyState extends StatelessWidget {
  const new({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: AlignmentDirectional.centerStart,
      child: Text(
        text,
        style: Theme.of(context).textTheme.bodyLarge?.copyWith(
          color: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      ),
    );
  }
}
