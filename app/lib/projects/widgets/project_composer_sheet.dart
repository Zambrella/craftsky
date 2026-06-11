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
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/shared/rich_text/widgets/facet_autocomplete_editor.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_text_field.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
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
  final _bodyController = FacetTextEditingController();
  final _bodyFocusNode = FocusNode(debugLabel: 'projectComposerBody');
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

  @override
  void initState() {
    super.initState();
    _composerId = widget.composerId ?? const Uuid().v4();
  }

  @override
  void dispose() {
    _bodyController.dispose();
    _bodyFocusNode.dispose();
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
    final photoErrorText = _attemptedSubmit && imagesState.images.isEmpty
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
      canPop: !hasDraft || createState.isLoading,
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        final discard = await _confirmDiscard();
        if (!discard || !context.mounted) return;
        Navigator.of(context).pop();
      },
      child: Scaffold(
        backgroundColor: swatches.paper,
        appBar: AppBar(
          title: Text(
            l10n.projectComposerTitle,
            style: theme.textTheme.titleLarge,
          ),
          actions: [
            Padding(
              padding: EdgeInsets.only(right: spacing.sp4),
              child: TextButton(
                onPressed: canSubmit
                    ? () => _submitProject(trimmedBody: trimmedBody)
                    : null,
                child: Text(l10n.postComposeSubmit),
              ),
            ),
          ],
        ),
        body: SafeArea(
          top: false,
          bottom: false,
          child: SingleChildScrollView(
            padding: EdgeInsets.fromLTRB(
              spacing.sp4,
              spacing.sp5,
              spacing.sp4,
              0,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                ComposerImageAttachmentSection(
                  imagesState: imagesState,
                  enabled: controlsEnabled,
                  validationErrorText: photoErrorText,
                  onAddImages: () =>
                      ref.read(imagesProvider.notifier).addImages(),
                  onAltTextChanged: (imageId, value) => ref
                      .read(imagesProvider.notifier)
                      .setAltText(imageId, value),
                  onRemove: (imageId) =>
                      ref.read(imagesProvider.notifier).remove(imageId),
                  onReorder: (fromIndex, toIndex) => ref
                      .read(imagesProvider.notifier)
                      .reorder(fromIndex: fromIndex, toIndex: toIndex),
                ),
                SizedBox(height: spacing.sp6),
                if (_formValidationError case final formValidationError?) ...[
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
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      CraftskyFormBuilderTextField(
                        name: ProjectComposerFields.title,
                        label: l10n.projectComposerProjectTitleLabel,
                        hintText: l10n.projectComposerProjectTitleHint,
                        enabled: controlsEnabled,
                      ),
                      SizedBox(height: spacing.sp4),
                      CraftskyFormBuilderDropdownField<String>(
                        name: ProjectComposerFields.craftType,
                        label: l10n.projectComposerCraftTypeLabel,
                        options: _selectOptions(
                          ProjectOptionCatalogs.craftTypes,
                        ),
                        enabled: controlsEnabled,
                        validator: (value) => value == null
                            ? l10n.projectComposerCraftRequiredError
                            : null,
                        onChanged: (value) {
                          setState(() {
                            _activeCraftType = value;
                            _formValidationError = null;
                            _sewingProjectType = null;
                            _knittingProjectType = null;
                            _crochetProjectType = null;
                            _quiltingProjectType = null;
                            _formKey
                                .currentState
                                ?.fields[ProjectComposerFields
                                    .sewingProjectSubtype]
                                ?.didChange(null);
                            _formKey
                                .currentState
                                ?.fields[ProjectComposerFields
                                    .knittingProjectSubtype]
                                ?.didChange(null);
                            _formKey
                                .currentState
                                ?.fields[ProjectComposerFields
                                    .crochetProjectSubtype]
                                ?.didChange(null);
                            _formKey
                                .currentState
                                ?.fields[ProjectComposerFields
                                    .quiltingProjectSubtype]
                                ?.didChange(null);
                          });
                        },
                      ),
                      SizedBox(height: spacing.sp4),
                      FacetAutocompleteEditor(
                        key: const Key('project-composer-body-editor'),
                        label: l10n.postComposeHint,
                        hintText: l10n.postComposeBodyHint,
                        controller: _bodyController,
                        focusNode: _bodyFocusNode,
                        minLines: 3,
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
                      CraftskyFormBuilderDropdownField<String>(
                        name: ProjectComposerFields.status,
                        label: l10n.projectComposerStatusLabel,
                        initialValue: ProjectOptionCatalogs.finishedStatusToken,
                        options: _selectOptions(ProjectOptionCatalogs.statuses),
                        enabled: controlsEnabled,
                      ),
                      SizedBox(height: spacing.sp4),
                      CraftskyFormBuilderMultiSelectField<String>(
                        name: ProjectComposerFields.materials,
                        label: l10n.projectComposerMaterialsLabel,
                        allowCustomValues: true,
                        maxSelected: 20,
                        customValueHintText:
                            l10n.projectComposerMaterialsAddHint,
                        addCustomValueLabel:
                            l10n.projectComposerMaterialsAddAction,
                        disabledText: l10n.projectComposerFieldDisabledLabel,
                        maxSelectedErrorText: l10n
                            .projectComposerMultiSelectMaxSelectedError(20),
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
                        maxSelectedErrorText: l10n
                            .projectComposerMultiSelectMaxSelectedError(10),
                        enabled: controlsEnabled,
                      ),
                      SizedBox(height: spacing.sp4),
                      CraftskyFormBuilderMultiSelectField<String>(
                        name: ProjectComposerFields.designTags,
                        label: l10n.projectComposerDesignTagsLabel,
                        options: _selectOptions(
                          ProjectOptionCatalogs.designTags,
                        ),
                        maxSelected: 10,
                        searchHintText:
                            l10n.projectComposerDesignTagsSearchHint,
                        disabledText: l10n.projectComposerFieldDisabledLabel,
                        maxSelectedErrorText: l10n
                            .projectComposerMultiSelectMaxSelectedError(10),
                        enabled: controlsEnabled,
                      ),
                      SizedBox(height: spacing.sp4),
                      ExpansionTile(
                        tilePadding: EdgeInsets.zero,
                        maintainState: true,
                        childrenPadding: EdgeInsets.only(bottom: spacing.sp3),
                        title: Text(l10n.projectComposerPatternSectionLabel),
                        children: [
                          CraftskyFormBuilderTextField(
                            name: ProjectComposerFields.patternName,
                            label: l10n.projectComposerPatternNameLabel,
                            hintText: l10n.projectComposerPatternNameHint,
                            textFieldKey: const Key('pattern-name-input'),
                            enabled: controlsEnabled,
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
                            options: _selectOptions(
                              ProjectOptionCatalogs.patternDifficulties,
                            ),
                            enabled: controlsEnabled,
                          ),
                          SizedBox(height: spacing.sp4),
                          CraftskyFormBuilderTextField(
                            name: ProjectComposerFields.patternDesigner,
                            label: l10n.projectComposerPatternDesignerLabel,
                            hintText: l10n.projectComposerPatternDesignerHint,
                            enabled: controlsEnabled,
                          ),
                          SizedBox(height: spacing.sp4),
                          CraftskyFormBuilderTextField(
                            name: ProjectComposerFields.patternPublisher,
                            label: l10n.projectComposerPatternPublisherLabel,
                            hintText: l10n.projectComposerPatternPublisherHint,
                            enabled: controlsEnabled,
                          ),
                        ],
                      ),
                      SizedBox(height: spacing.sp2),
                      ExpansionTile(
                        tilePadding: EdgeInsets.zero,
                        childrenPadding: EdgeInsets.only(bottom: spacing.sp3),
                        title: Text(l10n.projectComposerMoreDetailsLabel),
                        children: _detailFields(l10n, spacing, controlsEnabled),
                      ),
                    ],
                  ),
                ),
                SizedBox(
                  key: const Key('project-composer-bottom-safe-space'),
                  height: spacing.sp7 + MediaQuery.paddingOf(context).bottom,
                ),
              ],
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
      _ => const [SizedBox.shrink()],
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
      CraftskyFormBuilderTextField(
        name: ProjectComposerFields.knittingGaugeStitches,
        label: l10n.projectComposerGaugeStitchesLabel,
        hintText: l10n.projectComposerGaugeStitchesHint,
        keyboardType: TextInputType.number,
        textFieldKey: const Key('knitting-gauge-stitches-input'),
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderTextField(
        name: ProjectComposerFields.knittingGaugeRows,
        label: l10n.projectComposerGaugeRowsLabel,
        hintText: l10n.projectComposerGaugeRowsHint,
        keyboardType: TextInputType.number,
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderTextField(
        name: ProjectComposerFields.knittingGaugeMeasurement,
        label: l10n.projectComposerGaugeMeasurementLabel,
        hintText: l10n.projectComposerGaugeMeasurementHint,
        keyboardType: TextInputType.number,
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
      CraftskyFormBuilderTextField(
        name: ProjectComposerFields.crochetGaugeStitches,
        label: l10n.projectComposerGaugeStitchesLabel,
        hintText: l10n.projectComposerGaugeStitchesHint,
        keyboardType: TextInputType.number,
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderTextField(
        name: ProjectComposerFields.crochetGaugeRows,
        label: l10n.projectComposerGaugeRowsLabel,
        hintText: l10n.projectComposerGaugeRowsHint,
        keyboardType: TextInputType.number,
        enabled: controlsEnabled,
      ),
      SizedBox(height: spacing.sp4),
      CraftskyFormBuilderTextField(
        name: ProjectComposerFields.crochetGaugeMeasurement,
        label: l10n.projectComposerGaugeMeasurementLabel,
        hintText: l10n.projectComposerGaugeMeasurementHint,
        keyboardType: TextInputType.number,
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
}
