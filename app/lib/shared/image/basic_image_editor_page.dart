import 'dart:async';

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/image/basic_image_editor.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:material_ui/material_ui.dart';
import 'package:pro_image_editor/pro_image_editor.dart';

/// The package owns crop geometry; CraftSky owns tools and the route result.
class BasicImageEditorPage extends StatefulWidget {
  const new({
    required this.request,
    this.onExportError,
    super.key,
  });
  final BasicImageEditRequest request;
  final void Function(Object, StackTrace)? onExportError;

  @override
  State<BasicImageEditorPage> createState() => _BasicImageEditorPageState();
}

class _BasicImageEditorPageState extends State<BasicImageEditorPage> {
  BasicImageEditOutput? _output;
  bool _completed = false;
  bool _exporting = false;

  Future<void> _export(CropRotateEditorState editor) async {
    if (_exporting || _completed) return;
    setState(() => _exporting = true);
    try {
      await editor.done();
    } on Object catch (error, stack) {
      LoadingDialog.instance.hide();
      if (mounted) widget.onExportError?.call(error, stack);
    } finally {
      if (mounted) setState(() => _exporting = false);
    }
  }

  @override
  void dispose() {
    // Package export overlays live above the route. Release them on any exit,
    // including account invalidation while generation is still in flight.
    if (_exporting) LoadingDialog.instance.hide();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    // Photo editing always uses the dark palette, including custom controls
    // and package overlays, even when the surrounding app is in light mode.
    final theme = AppTheme.darkThemeData;
    final colors = theme.colorScheme;
    final avatar = widget.request.mode == BasicImageEditMode.avatar;
    final duration =
        MediaQuery.disableAnimationsOf(context) ||
            MediaQuery.accessibleNavigationOf(context)
        ? Duration.zero
        : const Duration(milliseconds: 200);
    return CropRotateEditor.memory(
      widget.request.bytes,
      initConfigs: CropRotateEditorInitConfigs(
        theme: theme,
        convertToUint8List: true,
        enablePopWhenDone: false,
        callbacks: ProImageEditorCallbacks(
          onImageEditingComplete: (bytes) async {
            if (_completed || _output != null || !mounted) return;
            _output = BasicImageEditOutput(bytes: bytes, mimeType: 'image/png');
          },
          onCloseEditor: (_) {
            if (_completed || !mounted) return;
            _completed = true;
            Navigator.of(context).pop(_output);
          },
        ),
        configs: ProImageEditorConfigs(
          theme: theme,
          i18n: I18n(
            done: l10n.postComposeDone,
            cancel: l10n.actionCancel,
            doneLoadingMsg: l10n.postComposePreparingImage,
            cropRotateEditor: I18nCropRotateEditor(
              bottomNavigationBarText: l10n.imageEditCrop,
              rotate: l10n.imageEditRotate,
              done: l10n.postComposeDone,
              cancel: l10n.actionCancel,
              back: l10n.actionCancel,
            ),
          ),
          imageGeneration: const ImageGenerationConfigs(
            outputFormat: OutputFormat.png,
            maxOutputSize: Size(4000, 4000),
            enableUseOriginalBytes: false,
            enableBackgroundGeneration: false,
          ),
          cropRotateEditor: CropRotateEditorConfigs(
            style: CropRotateEditorStyle(
              background: theme.scaffoldBackgroundColor,
              appBarBackground: theme.scaffoldBackgroundColor,
              appBarColor: colors.onSurface,
              bottomBarBackground: theme.scaffoldBackgroundColor,
              bottomBarColor: colors.onSurface,
              cropCornerColor: colors.primary,
            ),
            widgets: CropRotateEditorWidgets(
              appBar: (editor, stream) => ReactiveAppbar(
                stream: stream,
                builder: (_) => AppBar(
                  leading: IconButton(
                    key: const Key('photo-editor-cancel'),
                    tooltip: l10n.actionCancel,
                    onPressed: _exporting ? null : editor.close,
                    icon: const Icon(Icons.close),
                  ),
                  title: Text(l10n.imageEditTitle),
                  actions: [
                    IconButton(
                      key: const Key('photo-editor-done'),
                      tooltip: l10n.postComposeDone,
                      onPressed: _exporting
                          ? null
                          : () => unawaited(_export(editor)),
                      icon: const Icon(Icons.check),
                    ),
                  ],
                ),
              ),
            ),
            tools: const [CropRotateTool.rotate],
            initAspectRatio: avatar ? 1 : -1,
            initialCropMode: avatar ? CropMode.oval : CropMode.rectangular,
            exportOvalMask: false,
            enableKeyboardShortcuts: false,
            tiltConfigs: const TiltConfigs(
              showTiltButton: false,
              showTiltRotate: false,
              showTiltVertical: false,
              showTiltHorizontal: false,
            ),
            animationDuration: duration,
            cropDragAnimationDuration: duration,
            fadeInOutsideCropAreaAnimationDuration: duration,
            opacityOutsideCropAreaDuration: duration,
          ),
        ),
      ),
    );
  }
}
