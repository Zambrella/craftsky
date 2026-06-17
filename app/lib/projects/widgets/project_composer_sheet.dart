import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/composer/project_composer_draft_state.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/composer/project_composer_payload.dart';
import 'package:craftsky_app/projects/composer/project_composer_submit_adapter.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/rich_text/facet_autocomplete_controller.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/shared/rich_text/widgets/facet_autocomplete_editor.dart';
import 'package:craftsky_app/shared/rich_text/widgets/faceted_text.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_text_field.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:uuid/uuid.dart';

Future<Post?> showProjectComposerSheet(BuildContext context) {
  return Navigator.of(context, rootNavigator: true).push<Post?>(
    MaterialPageRoute<Post?>(
      fullscreenDialog: true,
      builder: (_) => const ProjectComposerSheet(),
    ),
  );
}

class ProjectComposerSheet extends ConsumerStatefulWidget {
  const ProjectComposerSheet({super.key, this.composerId});

  static const maxCharacters = 2000;

  final String? composerId;

  @override
  ConsumerState<ProjectComposerSheet> createState() =>
      _ProjectComposerSheetState();
}

class _ProjectComposerSheetState extends ConsumerState<ProjectComposerSheet> {
  final _formKey = GlobalKey<FormBuilderState>();
  final _wizardPagesKey = GlobalKey<_MountedWizardPagesState>();
  final _scrollController = ScrollController();
  final _bodyController = FacetTextEditingController();
  final _bodyFocusNode = FocusNode(debugLabel: 'projectComposerBody');
  final _backActionFocusNode = FocusNode(
    debugLabel: 'projectComposerBackAction',
  );
  final _primaryActionFocusNode = FocusNode(
    debugLabel: 'projectComposerPrimaryAction',
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
  int _currentPage = 0;
  String _bodyText = '';
  String _patternNameText = '';
  bool _attemptedPageOneNext = false;
  String? _activeCraftType;
  String? _sewingProjectType;
  String? _knittingProjectType;
  String? _crochetProjectType;
  String? _quiltingProjectType;
  bool _attemptedSubmit = false;
  String? _formValidationError;
  int? _lastImageNoticeId;

  @override
  void initState() {
    super.initState();
    _composerId = widget.composerId ?? const Uuid().v4();
  }

  @override
  void dispose() {
    _scrollController.dispose();
    _bodyController.dispose();
    _bodyFocusNode.dispose();
    _backActionFocusNode.dispose();
    _primaryActionFocusNode.dispose();
    _patternNameController.dispose();
    _patternDesignerController.dispose();
    _patternPublisherController.dispose();
    _patternNameFocusNode.dispose();
    _patternDesignerFocusNode.dispose();
    _patternPublisherFocusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final createState = ref.watch(createPostProvider);
    final imagesProvider = composerImagesProvider(_composerId);
    final imagesState = ref.watch(imagesProvider);
    final controlsEnabled = !createState.isLoading;
    final trimmedBody = _bodyText.trim();
    final tooLong = _bodyText.length > ProjectComposerSheet.maxCharacters;
    final canSubmit = !createState.isLoading && imagesState.canSubmitImages();
    final bodyErrorText = switch ((_attemptedSubmit, trimmedBody.isEmpty)) {
      (true, true) => l10n.projectComposerBodyRequiredError,
      _ when tooLong => l10n.postComposeTooLong,
      _ => null,
    };
    final photoErrorText =
        (_attemptedSubmit || _attemptedPageOneNext) &&
            imagesState.images.isEmpty
        ? l10n.projectComposerPhotoRequiredError
        : null;
    final hasDraft = ProjectComposerDraftState.hasDraft(
      bodyText: _bodyText,
      initialBodyText: '',
      imageCount: imagesState.images.length,
      formValues: _formKey.currentState?.instantValue ?? const {},
    );
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
      canPop: _currentPage == 0 && (!hasDraft || createState.isLoading),
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        if (_currentPage > 0) {
          _setCurrentPage(_currentPage - 1);
          return;
        }
        final discard = await _confirmDiscard();
        if (!discard || !context.mounted) return;
        Navigator.of(context).pop();
      },
      child: Actions(
        actions: <Type, Action<Intent>>{
          NextFocusIntent: _ProjectComposerNextFocusAction(
            shouldEnterPage: () => _backActionFocusNode.hasPrimaryFocus,
            enterPage: _focusFirstPageField,
            exitPage: _focusPrimaryActionFromLastPageField,
          ),
        },
        child: Scaffold(
          backgroundColor: swatches.paper,
          appBar: AppBar(
            leading: _currentPage == 0
                ? null
                : IconButton(
                    key: const Key('project-composer-back-action'),
                    focusNode: _backActionFocusNode,
                    icon: const BackButtonIcon(),
                    tooltip: MaterialLocalizations.of(
                      context,
                    ).backButtonTooltip,
                    onPressed: createState.isLoading
                        ? null
                        : () => _setCurrentPage(_currentPage - 1),
                  ),
            title: Text(
              l10n.projectComposerTitle,
              style: theme.textTheme.titleLarge,
            ),
            actions: [
              Padding(
                padding: EdgeInsets.only(right: spacing.sp4),
                child: TextButton(
                  key: const Key('project-composer-primary-action'),
                  focusNode: _primaryActionFocusNode,
                  onPressed: _currentPage < 2
                      ? (controlsEnabled ? _goToNextPage : null)
                      : (canSubmit
                            ? () => _submitProject(trimmedBody: trimmedBody)
                            : null),
                  child: Text(
                    _currentPage < 2
                        ? l10n.projectComposerNextAction
                        : l10n.postComposeSubmit,
                  ),
                ),
              ),
            ],
          ),
          body: GestureDetector(
            behavior: HitTestBehavior.translucent,
            onTap: () => FocusManager.instance.primaryFocus?.unfocus(),
            child: SafeArea(
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
                    if (_formValidationError
                        case final formValidationError?) ...[
                      Text(
                        formValidationError,
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.error,
                        ),
                      ),
                      SizedBox(height: spacing.sp4),
                    ],
                    FormBuilder(
                      key: _formKey,
                      child: _MountedWizardPages(
                        key: _wizardPagesKey,
                        currentPage: _currentPage,
                        onExitBackward: _currentPage == 0
                            ? null
                            : () {
                                _backActionFocusNode.requestFocus();
                                return true;
                              },
                        onExitForward: () {
                          _primaryActionFocusNode.requestFocus();
                          return true;
                        },
                        children: [
                          _pageOne(
                            l10n: l10n,
                            theme: theme,
                            spacing: spacing,
                            imagesState: imagesState,
                            controlsEnabled: controlsEnabled,
                            photoErrorText: photoErrorText,
                            patternInfoTitle:
                                l10n.projectComposerPatternInfoSectionLabel,
                            onAddImages: () =>
                                ref.read(imagesProvider.notifier).addImages(),
                            onAltTextChanged: (imageId, value) => ref
                                .read(imagesProvider.notifier)
                                .setAltText(imageId, value),
                            onRemoveImage: (imageId) => ref
                                .read(imagesProvider.notifier)
                                .remove(imageId),
                            onReorderImages: (fromIndex, toIndex) => ref
                                .read(imagesProvider.notifier)
                                .reorder(
                                  fromIndex: fromIndex,
                                  toIndex: toIndex,
                                ),
                          ),
                          _pageTwo(
                            l10n: l10n,
                            theme: theme,
                            spacing: spacing,
                            controlsEnabled: controlsEnabled,
                          ),
                          _pageThree(
                            l10n: l10n,
                            spacing: spacing,
                            controlsEnabled: controlsEnabled,
                            bodyErrorText: bodyErrorText,
                          ),
                        ],
                      ),
                    ),
                    SizedBox(
                      key: const Key('project-composer-bottom-safe-space'),
                      height:
                          spacing.sp7 + MediaQuery.paddingOf(context).bottom,
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
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

  bool _focusFirstPageField() {
    return _wizardPagesKey.currentState?.focusFirstInCurrentPage() ?? false;
  }

  bool _focusPrimaryActionFromLastPageField() {
    final wizardPages = _wizardPagesKey.currentState;
    if (wizardPages == null || !wizardPages.primaryFocusAtEndOfCurrentPage()) {
      return false;
    }
    _primaryActionFocusNode.requestFocus();
    return true;
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
    List<ProjectOption> options,
  ) {
    return [
      for (final option in options)
        CraftskySelectOption<String>(
          value: option.value,
          label: option.label,
          description: option.description,
        ),
    ];
  }

  Widget _pageOne({
    required AppLocalizations l10n,
    required ThemeData theme,
    required SpacingTheme spacing,
    required ComposerImagesState imagesState,
    required bool controlsEnabled,
    required String? photoErrorText,
    required String patternInfoTitle,
    required Future<void> Function()? onAddImages,
    required void Function(String imageId, String value) onAltTextChanged,
    required ValueChanged<String> onRemoveImage,
    required void Function(int fromIndex, int toIndex) onReorderImages,
  }) {
    final showPatternDetails = _hasMeaningfulPatternName(_patternNameText);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ComposerImageAttachmentSection(
          imagesState: imagesState,
          enabled: controlsEnabled,
          validationErrorText: photoErrorText,
          required: true,
          requiredLabel: l10n.projectComposerRequiredLabel,
          onAddImages: onAddImages,
          onAltTextChanged: onAltTextChanged,
          onRemove: onRemoveImage,
          onReorder: onReorderImages,
        ),
        SizedBox(height: spacing.sp6),
        CraftskyFormBuilderDropdownField<String>(
          name: ProjectComposerFields.craftType,
          label: l10n.projectComposerCraftTypeLabel,
          required: true,
          requiredLabel: l10n.projectComposerRequiredLabel,
          options: _selectOptions(ProjectOptionCatalogs.craftTypes),
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
          label: l10n.projectComposerProjectTitleLabel,
          hintText: l10n.projectComposerProjectTitleHint,
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
        _FacetFormBuilderTextField(
          name: ProjectComposerFields.patternName,
          key: const Key('project-composer-pattern-name-field'),
          editorKey: const Key('project-composer-pattern-name-editor'),
          label: l10n.projectComposerPatternNameLabel,
          hintText: l10n.projectComposerPatternNameHint,
          controller: _patternNameController,
          focusNode: _patternNameFocusNode,
          enabled: controlsEnabled,
          initialDisplayText: '#',
          allowedTokenKinds: const {ActiveFacetTokenKind.hashtag},
          normalizeValue: _patternFormValue,
          onChanged: _onPatternNameChanged,
        ),
        if (showPatternDetails) ...[
          SizedBox(height: spacing.sp4),
          Text(
            patternInfoTitle,
            style: theme.textTheme.titleLarge?.copyWith(
              fontWeight: FontWeight.w900,
              color: theme.colorScheme.onSurface,
            ),
          ),
          SizedBox(height: spacing.sp3),
          _FacetFormBuilderTextField(
            name: ProjectComposerFields.patternDesigner,
            key: const Key('project-composer-pattern-designer-field'),
            editorKey: const Key('project-composer-pattern-designer-editor'),
            label: l10n.projectComposerPatternDesignerLabel,
            hintText: l10n.projectComposerPatternDesignerHint,
            controller: _patternDesignerController,
            focusNode: _patternDesignerFocusNode,
            enabled: controlsEnabled,
            allowedTokenKinds: const {ActiveFacetTokenKind.mention},
          ),
          SizedBox(height: spacing.sp4),
          _FacetFormBuilderTextField(
            name: ProjectComposerFields.patternPublisher,
            key: const Key('project-composer-pattern-publisher-field'),
            editorKey: const Key('project-composer-pattern-publisher-editor'),
            label: l10n.projectComposerPatternPublisherLabel,
            hintText: l10n.projectComposerPatternPublisherHint,
            controller: _patternPublisherController,
            focusNode: _patternPublisherFocusNode,
            enabled: controlsEnabled,
            allowedTokenKinds: const {ActiveFacetTokenKind.mention},
          ),
          SizedBox(height: spacing.sp4),
          CraftskyFormBuilderTextField(
            name: ProjectComposerFields.patternUrl,
            label: l10n.projectComposerPatternUrlLabel,
            hintText: l10n.projectComposerPatternUrlHint,
            keyboardType: TextInputType.url,
            textFieldKey: const Key('pattern-url-input'),
            enabled: controlsEnabled,
          ),
          SizedBox(height: spacing.sp4),
          CraftskyFormBuilderDropdownField<String>(
            name: ProjectComposerFields.patternDifficulty,
            label: l10n.projectComposerPatternDifficultyLabel,
            options: _selectOptions(ProjectOptionCatalogs.patternDifficulties),
            enabled: controlsEnabled,
          ),
        ],
      ],
    );
  }

  Widget _pageTwo({
    required AppLocalizations l10n,
    required ThemeData theme,
    required SpacingTheme spacing,
    required bool controlsEnabled,
  }) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          l10n.projectComposerOptionalDetailsPrompt,
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        SizedBox(height: spacing.sp4),
        _MaterialsFormBuilderField(
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
        SizedBox(height: spacing.sp4),
        ..._detailFields(l10n, spacing, controlsEnabled),
      ],
    );
  }

  Widget _pageThree({
    required AppLocalizations l10n,
    required SpacingTheme spacing,
    required bool controlsEnabled,
    required String? bodyErrorText,
  }) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        FacetAutocompleteEditor(
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
          helperText:
              '${_bodyText.length}/${ProjectComposerSheet.maxCharacters}',
          helperAlignment: AlignmentDirectional.centerEnd,
          onChanged: (value) => setState(() => _bodyText = value),
        ),
        SizedBox(height: spacing.sp4),
      ],
    );
  }

  void _onCraftTypeChanged(String? value) {
    setState(() {
      _activeCraftType = value;
      _formValidationError = null;
      _sewingProjectType = null;
      _knittingProjectType = null;
      _crochetProjectType = null;
      _quiltingProjectType = null;
      _formKey.currentState?.fields[ProjectComposerFields.sewingProjectSubtype]
          ?.didChange(null);
      _formKey
          .currentState
          ?.fields[ProjectComposerFields.knittingProjectSubtype]
          ?.didChange(null);
      _formKey.currentState?.fields[ProjectComposerFields.crochetProjectSubtype]
          ?.didChange(null);
      _formKey
          .currentState
          ?.fields[ProjectComposerFields.quiltingProjectSubtype]
          ?.didChange(null);
    });
  }

  void _goToNextPage() {
    if (_currentPage != 0) {
      _setCurrentPage(_currentPage + 1);
      return;
    }

    setState(() {
      _attemptedPageOneNext = true;
      _formValidationError = null;
    });
    final craftField =
        _formKey.currentState?.fields[ProjectComposerFields.craftType];
    final isCraftValid = craftField?.validate() ?? false;
    final hasRequiredPhoto = ref
        .read(composerImagesProvider(_composerId))
        .images
        .isNotEmpty;
    if (!isCraftValid || !hasRequiredPhoto) {
      return;
    }

    setState(() => _attemptedPageOneNext = false);
    _setCurrentPage(1);
  }

  void _setCurrentPage(int page) {
    if (_currentPage == page) {
      _scrollToTop();
      return;
    }
    FocusManager.instance.primaryFocus?.unfocus();
    setState(() => _currentPage = page);
    _scrollToTop();
  }

  void _scrollToTop() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_scrollController.hasClients) return;
      _scrollController.jumpTo(0);
    });
  }

  void _onPatternNameChanged(String value) {
    final nextValue = _patternFormValue(value);
    final hadDetails = _hasMeaningfulPatternName(_patternNameText);
    final hasDetails = _hasMeaningfulPatternName(nextValue ?? '');
    setState(() => _patternNameText = nextValue ?? '');
    if (hadDetails && !hasDetails) {
      _patternDesignerController.clear();
      _patternPublisherController.clear();
      _formKey.currentState?.fields[ProjectComposerFields.patternDesigner]
          ?.didChange(null);
      _formKey.currentState?.fields[ProjectComposerFields.patternPublisher]
          ?.didChange(null);
      _formKey.currentState?.fields[ProjectComposerFields.patternUrl]
          ?.didChange(null);
      _formKey.currentState?.fields[ProjectComposerFields.patternDifficulty]
          ?.didChange(null);
    }
  }

  static bool _hasMeaningfulPatternName(String value) {
    final trimmed = value.trim();
    return trimmed.isNotEmpty && trimmed != '#';
  }

  static String? _patternFormValue(String value) {
    final trimmed = value.trim();
    if (trimmed.isEmpty || trimmed == '#') return null;
    return trimmed;
  }

  List<Widget> _detailFields(
    AppLocalizations l10n,
    SpacingTheme spacing,
    bool controlsEnabled,
  ) {
    return switch (_activeCraftType) {
      ProjectOptionCatalogs.sewingCraftToken => _sewingDetailFields(
        l10n,
        spacing,
        controlsEnabled,
      ),
      ProjectOptionCatalogs.knittingCraftToken => _knittingDetailFields(
        l10n,
        spacing,
        controlsEnabled,
      ),
      ProjectOptionCatalogs.crochetCraftToken => _crochetDetailFields(
        l10n,
        spacing,
        controlsEnabled,
      ),
      ProjectOptionCatalogs.quiltingCraftToken => _quiltingDetailFields(
        l10n,
        spacing,
        controlsEnabled,
      ),
      _ => [
        Align(
          alignment: AlignmentDirectional.centerStart,
          child: Text(
            l10n.projectComposerSelectCraftTypeEmptyState,
            style: Theme.of(context).textTheme.bodyLarge?.copyWith(
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            ),
          ),
        ),
      ],
    };
  }

  List<Widget> _sewingDetailFields(
    AppLocalizations l10n,
    SpacingTheme spacing,
    bool controlsEnabled,
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
            _formKey
                .currentState
                ?.fields[ProjectComposerFields.sewingProjectSubtype]
                ?.didChange(null);
          });
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
        label: l10n.projectComposerSizeMadeLabel,
        hintText: l10n.projectComposerSizeMadeHint,
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderMultilineTextField(
        name: ProjectComposerFields.sewingFitNotes,
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
            _formKey
                .currentState
                ?.fields[ProjectComposerFields.knittingProjectSubtype]
                ?.didChange(null);
          });
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
        options: _selectOptions(ProjectOptionCatalogs.yarnWeights),
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.knittingNeedleSize,
        label: l10n.projectComposerNeedleSizeLabel,
        options: _selectOptions(ProjectOptionCatalogs.needleSizes),
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
        options: _selectOptions(ProjectOptionCatalogs.gaugeUnits),
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderTextField(
        name: ProjectComposerFields.knittingFinishedSize,
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
            _formKey
                .currentState
                ?.fields[ProjectComposerFields.crochetProjectSubtype]
                ?.didChange(null);
          });
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
        options: _selectOptions(ProjectOptionCatalogs.yarnWeights),
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderDropdownField<String>(
        name: ProjectComposerFields.crochetHookSize,
        label: l10n.projectComposerHookSizeLabel,
        options: _selectOptions(ProjectOptionCatalogs.hookSizes),
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
        options: _selectOptions(ProjectOptionCatalogs.gaugeUnits),
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderTextField(
        name: ProjectComposerFields.crochetFinishedSize,
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
            _formKey
                .currentState
                ?.fields[ProjectComposerFields.quiltingProjectSubtype]
                ?.didChange(null);
          });
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
    final hasRequiredPhoto = imagesState.images.isNotEmpty;
    final isBodyLengthValid =
        _bodyText.length <= ProjectComposerSheet.maxCharacters;
    if (!isFormValid ||
        !hasRequiredBody ||
        !hasRequiredPhoto ||
        !isBodyLengthValid) {
      _showFirstInvalidPage(
        isFormValid: isFormValid,
        hasRequiredBody: hasRequiredBody,
        hasRequiredPhoto: hasRequiredPhoto,
        isBodyLengthValid: isBodyLengthValid,
      );
      return;
    }

    final payload = buildProjectComposerPayload(formValues: form.value);
    final project = payload.project;
    if (project == null) {
      setState(() {
        _formValidationError =
            payload.errors.any(
              (error) =>
                  error.code == ProjectComposerValidationCode.invalidGauge,
            )
            ? AppLocalizations.of(context).projectComposerGaugeInvalidError
            : null;
      });
      _setCurrentPage(1);
      return;
    }

    if (imagesState.hasImagesMissingAltText) {
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
      project: project,
      imagesState: imagesState,
      generateFacets: ref.read(facetGeneratorProvider).generate,
    );

    await ref
        .read(createPostProvider.notifier)
        .create(
          text: args.text,
          reply: args.reply,
          project: args.project,
          images: args.images,
          facets: args.facets,
        );
  }

  void _showFirstInvalidPage({
    required bool isFormValid,
    required bool hasRequiredBody,
    required bool hasRequiredPhoto,
    required bool isBodyLengthValid,
  }) {
    final craftField =
        _formKey.currentState?.fields[ProjectComposerFields.craftType];
    final page = switch ((
      !hasRequiredPhoto || craftField?.hasError == true,
      !isFormValid,
      !hasRequiredBody || !isBodyLengthValid,
    )) {
      (true, _, _) => 0,
      (_, true, _) => 1,
      (_, _, true) => 2,
      _ => _currentPage,
    };
    _setCurrentPage(page);
  }
}

class _ProjectComposerNextFocusAction extends NextFocusAction {
  _ProjectComposerNextFocusAction({
    required this.shouldEnterPage,
    required this.enterPage,
    required this.exitPage,
  });

  final bool Function() shouldEnterPage;
  final bool Function() enterPage;
  final bool Function() exitPage;

  @override
  bool invoke(NextFocusIntent intent) {
    if (shouldEnterPage() && enterPage()) return true;
    if (exitPage()) return true;
    return super.invoke(intent);
  }
}

class _MountedWizardPages extends StatefulWidget {
  const _MountedWizardPages({
    required this.currentPage,
    required this.children,
    super.key,
    this.onExitBackward,
    this.onExitForward,
  });

  final int currentPage;
  final List<Widget> children;
  final bool Function()? onExitBackward;
  final bool Function()? onExitForward;

  @override
  State<_MountedWizardPages> createState() => _MountedWizardPagesState();
}

class _MountedWizardPagesState extends State<_MountedWizardPages> {
  late List<FocusNode> _pageFocusNodes;

  @override
  void initState() {
    super.initState();
    _pageFocusNodes = _createPageFocusNodes(widget.children.length);
  }

  @override
  void didUpdateWidget(covariant _MountedWizardPages oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.children.length != widget.children.length) {
      for (final node in _pageFocusNodes) {
        node.dispose();
      }
      _pageFocusNodes = _createPageFocusNodes(widget.children.length);
    }
  }

  @override
  void dispose() {
    for (final node in _pageFocusNodes) {
      node.dispose();
    }
    super.dispose();
  }

  static List<FocusNode> _createPageFocusNodes(int count) {
    return [
      for (var index = 0; index < count; index++)
        FocusNode(debugLabel: 'projectComposerPage$index'),
    ];
  }

  bool focusFirstInCurrentPage() {
    if (widget.currentPage < 0 ||
        widget.currentPage >= _pageFocusNodes.length) {
      return false;
    }
    final pageNode = _pageFocusNodes[widget.currentPage];
    final orderedGroups = _orderedFocusGroups(pageNode, pageNode);
    if (orderedGroups.isEmpty) return false;
    _preferredFocusTarget(orderedGroups.first).requestFocus();
    return true;
  }

  bool primaryFocusAtEndOfCurrentPage() {
    if (widget.currentPage < 0 ||
        widget.currentPage >= _pageFocusNodes.length) {
      return false;
    }
    final focusedNode = FocusManager.instance.primaryFocus;
    if (focusedNode == null) return false;
    final pageNode = _pageFocusNodes[widget.currentPage];
    if (focusedNode != pageNode && !focusedNode.ancestors.contains(pageNode)) {
      return false;
    }
    final orderedGroups = _orderedFocusGroups(pageNode, focusedNode);
    if (orderedGroups.isEmpty) return false;
    final currentGroup = _nearestOrderedNode(focusedNode, orderedGroups);
    return currentGroup == orderedGroups.last;
  }

  KeyEventResult _handlePageKey(FocusNode pageNode, KeyEvent event) {
    if (event is! KeyDownEvent || event.logicalKey != LogicalKeyboardKey.tab) {
      return KeyEventResult.ignored;
    }
    final focusedNode = FocusManager.instance.primaryFocus;
    if (focusedNode == null || !focusedNode.ancestors.contains(pageNode)) {
      return KeyEventResult.ignored;
    }

    final orderedGroups = _orderedFocusGroups(pageNode, focusedNode);
    if (orderedGroups.length < 2) return KeyEventResult.ignored;

    final currentGroup = _nearestOrderedNode(focusedNode, orderedGroups);
    if (currentGroup == null) return KeyEventResult.ignored;
    final currentIndex = orderedGroups.indexOf(currentGroup);
    if (currentIndex == -1) return KeyEventResult.ignored;
    final direction = HardwareKeyboard.instance.isShiftPressed ? -1 : 1;
    final nextIndex = currentIndex + direction;
    if (nextIndex < 0 || nextIndex >= orderedGroups.length) {
      final handled = direction < 0
          ? widget.onExitBackward?.call()
          : widget.onExitForward?.call();
      if (handled == true) return KeyEventResult.handled;
      return KeyEventResult.ignored;
    }
    _preferredFocusTarget(orderedGroups[nextIndex]).requestFocus();
    return KeyEventResult.handled;
  }

  static List<FocusNode> _orderedFocusGroups(
    FocusNode pageNode,
    FocusNode anchor,
  ) {
    final focusGroups = pageNode.descendants.where(
      (node) =>
          node.canRequestFocus &&
          node.context != null &&
          !_hasFocusableAncestorWithin(node, pageNode),
    );
    return WidgetOrderTraversalPolicy()
        .sortDescendants(focusGroups, anchor)
        .where((node) => node.context != null)
        .toList(growable: false);
  }

  static bool _hasFocusableAncestorWithin(FocusNode node, FocusNode root) {
    for (final ancestor in node.ancestors) {
      if (ancestor == root) return false;
      if (ancestor.canRequestFocus && ancestor.context != null) return true;
    }
    return false;
  }

  static FocusNode? _nearestOrderedNode(
    FocusNode node,
    List<FocusNode> orderedNodes,
  ) {
    if (orderedNodes.contains(node)) return node;
    for (final ancestor in node.ancestors) {
      if (orderedNodes.contains(ancestor)) return ancestor;
    }
    return null;
  }

  static FocusNode _preferredFocusTarget(FocusNode group) {
    final leafNodes = group.descendants.where(
      (node) => node.canRequestFocus && node.context != null,
    );
    final orderedLeafNodes = WidgetOrderTraversalPolicy()
        .sortDescendants(leafNodes, group)
        .toList(growable: false);
    if (orderedLeafNodes.isEmpty) return group;
    return orderedLeafNodes.last;
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (final (index, child) in widget.children.indexed)
          Offstage(
            offstage: index != widget.currentPage,
            child: TickerMode(
              enabled: index == widget.currentPage,
              child: FocusTraversalGroup(
                policy: WidgetOrderTraversalPolicy(),
                child: Focus(
                  focusNode: _pageFocusNodes[index],
                  canRequestFocus: index == widget.currentPage,
                  skipTraversal: true,
                  descendantsAreFocusable: index == widget.currentPage,
                  descendantsAreTraversable: index == widget.currentPage,
                  onKeyEvent: _handlePageKey,
                  child: child,
                ),
              ),
            ),
          ),
      ],
    );
  }
}

class _MaterialsFormBuilderField extends StatelessWidget {
  const _MaterialsFormBuilderField({
    required this.name,
    required this.label,
    required this.inputHintText,
    required this.addButtonLabel,
    required this.disabledText,
    required this.maxSelectedErrorText,
    required this.maxLengthErrorText,
    required this.enabled,
  });

  final String name;
  final String label;
  final String inputHintText;
  final String addButtonLabel;
  final String disabledText;
  final String maxSelectedErrorText;
  final String maxLengthErrorText;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<List<ProjectMaterial>>(
      name: name,
      initialValue: const [],
      enabled: enabled,
      builder: (field) {
        return _MaterialsInput(
          label: label,
          values: List<ProjectMaterial>.from(field.value ?? const []),
          inputHintText: inputHintText,
          addButtonLabel: addButtonLabel,
          disabledText: disabledText,
          maxSelectedErrorText: maxSelectedErrorText,
          maxLengthErrorText: maxLengthErrorText,
          enabled: field.widget.enabled,
          onChanged: field.didChange,
        );
      },
    );
  }
}

class _MaterialsInput extends ConsumerStatefulWidget {
  const _MaterialsInput({
    required this.label,
    required this.values,
    required this.inputHintText,
    required this.addButtonLabel,
    required this.disabledText,
    required this.maxSelectedErrorText,
    required this.maxLengthErrorText,
    required this.enabled,
    required this.onChanged,
  });

  static const maxSelected = 10;
  static const maxGraphemes = 100;

  final String label;
  final List<ProjectMaterial> values;
  final String inputHintText;
  final String addButtonLabel;
  final String disabledText;
  final String maxSelectedErrorText;
  final String maxLengthErrorText;
  final bool enabled;
  final ValueChanged<List<ProjectMaterial>> onChanged;

  @override
  ConsumerState<_MaterialsInput> createState() => _MaterialsInputState();
}

class _MaterialsInputState extends ConsumerState<_MaterialsInput> {
  final _controller = FacetTextEditingController();
  final _focusNode = FocusNode(debugLabel: 'projectMaterials');
  String? _errorText;

  bool get _canAdd => widget.enabled && _controller.text.trim().isNotEmpty;

  @override
  void initState() {
    super.initState();
    _controller.addListener(_handleTextChanged);
  }

  @override
  void dispose() {
    _controller
      ..removeListener(_handleTextChanged)
      ..dispose();
    _focusNode.dispose();
    super.dispose();
  }

  @override
  void didUpdateWidget(covariant _MaterialsInput oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.values.length < _MaterialsInput.maxSelected &&
        oldWidget.values.length != widget.values.length) {
      _errorText = null;
    }
  }

  void _handleTextChanged() {
    if (mounted) setState(() {});
  }

  Future<void> _addCurrent() async {
    if (!widget.enabled) return;
    final text = _controller.text.trim();
    if (text.isEmpty) return;
    if (text.characters.length > _MaterialsInput.maxGraphemes) {
      setState(() => _errorText = widget.maxLengthErrorText);
      _focusNode.requestFocus();
      return;
    }
    if (widget.values.length >= _MaterialsInput.maxSelected) {
      setState(() => _errorText = widget.maxSelectedErrorText);
      _focusNode.requestFocus();
      return;
    }

    final facets = await ref
        .read(facetGeneratorProvider)
        .generate(
          text,
          includeLinks: false,
        );
    if (!mounted) return;

    widget.onChanged([
      ...widget.values,
      ProjectMaterial(text: text, facets: facets.isEmpty ? null : facets),
    ]);
    _controller.clear();
    setState(() => _errorText = null);
    _focusNode.requestFocus();
  }

  void _remove(ProjectMaterial material) {
    if (!widget.enabled) return;
    widget.onChanged(
      List<ProjectMaterial>.from(widget.values)..remove(material),
    );
    setState(() => _errorText = null);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        FacetAutocompleteEditor(
          key: const Key('${ProjectComposerFields.materials}-custom-input'),
          label: widget.label,
          hintText: widget.inputHintText,
          controller: _controller,
          focusNode: _focusNode,
          enabled: widget.enabled,
          errorText: _errorText,
          betweenLabelAndField: widget.values.isEmpty
              ? null
              : _MaterialEntryList(
                  values: widget.values,
                  enabled: widget.enabled,
                  onRemove: _remove,
                ),
          suffixIcon: Padding(
            padding: const EdgeInsetsDirectional.only(end: 8),
            child: TextButton(
              key: const Key('${ProjectComposerFields.materials}-add-custom'),
              style: TextButton.styleFrom(
                minimumSize: Size.zero,
                padding: const EdgeInsets.symmetric(horizontal: 8),
                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              ),
              onPressed: _canAdd ? _addCurrent : null,
              child: Text(widget.addButtonLabel),
            ),
          ),
          allowedTokenKinds: const {
            ActiveFacetTokenKind.mention,
            ActiveFacetTokenKind.hashtag,
          },
          textInputAction: TextInputAction.done,
          onSubmitted: (_) => unawaited(_addCurrent()),
        ),
        if (!widget.enabled)
          Padding(
            padding: EdgeInsets.only(top: spacing.sp2),
            child: Text(
              widget.disabledText,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
            ),
          ),
      ],
    );
  }
}

class _MaterialEntryList extends StatelessWidget {
  const _MaterialEntryList({
    required this.values,
    required this.enabled,
    required this.onRemove,
  });

  final List<ProjectMaterial> values;
  final bool enabled;
  final ValueChanged<ProjectMaterial> onRemove;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final colors = theme.colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (final (index, material) in values.indexed) ...[
          DecoratedBox(
            decoration: BoxDecoration(
              color: colors.surface,
              border: Border.all(color: colors.outline),
              borderRadius: BorderRadius.circular(radii.r3),
            ),
            child: Padding(
              padding: EdgeInsetsDirectional.only(
                start: spacing.sp4,
                top: spacing.sp3,
                bottom: spacing.sp3,
                end: spacing.sp2,
              ),
              child: Row(
                children: [
                  Expanded(
                    child: FacetedText(
                      text: material.text,
                      facets: material.facets,
                      style: theme.textTheme.bodyLarge?.copyWith(
                        fontWeight: FontWeight.w700,
                        color: colors.onSurface,
                      ),
                    ),
                  ),
                  IconButton(
                    key: Key(
                      '${ProjectComposerFields.materials}-remove-'
                      '${material.text}',
                    ),
                    icon: const Icon(Icons.close),
                    tooltip: 'Remove material',
                    visualDensity: VisualDensity.compact,
                    padding: EdgeInsets.zero,
                    constraints: const BoxConstraints.tightFor(
                      width: 32,
                      height: 32,
                    ),
                    onPressed: enabled ? () => onRemove(material) : null,
                  ),
                ],
              ),
            ),
          ),
          if (index != values.length - 1) SizedBox(height: spacing.sp2),
        ],
      ],
    );
  }
}

class _FacetFormBuilderTextField extends StatefulWidget {
  const _FacetFormBuilderTextField({
    required this.name,
    required this.label,
    required this.controller,
    super.key,
    this.editorKey,
    this.focusNode,
    this.hintText,
    this.enabled = true,
    this.initialDisplayText,
    this.allowedTokenKinds,
    this.normalizeValue,
    this.onChanged,
  });

  final String name;
  final String label;
  final FacetTextEditingController controller;
  final Key? editorKey;
  final FocusNode? focusNode;
  final String? hintText;
  final bool enabled;
  final String? initialDisplayText;
  final Set<ActiveFacetTokenKind>? allowedTokenKinds;
  final String? Function(String value)? normalizeValue;
  final ValueChanged<String>? onChanged;

  @override
  State<_FacetFormBuilderTextField> createState() =>
      _FacetFormBuilderTextFieldState();
}

class _FacetFormBuilderTextFieldState
    extends State<_FacetFormBuilderTextField> {
  @override
  void initState() {
    super.initState();
    if (widget.initialDisplayText case final initialDisplayText?
        when widget.controller.text.isEmpty) {
      widget.controller.text = initialDisplayText;
      widget.controller.selection = TextSelection.collapsed(
        offset: initialDisplayText.length,
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<String>(
      name: widget.name,
      initialValue: '',
      enabled: widget.enabled,
      builder: (field) {
        return FacetAutocompleteEditor(
          key: widget.editorKey,
          label: widget.label,
          hintText: widget.hintText,
          controller: widget.controller,
          focusNode: widget.focusNode,
          enabled: field.widget.enabled,
          textInputAction: TextInputAction.next,
          allowedTokenKinds: widget.allowedTokenKinds,
          onChanged: (value) {
            field.didChange(widget.normalizeValue?.call(value) ?? value);
            widget.onChanged?.call(value);
          },
        );
      },
    );
  }
}
