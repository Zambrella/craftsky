import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/shared/rich_text/widgets/facet_autocomplete_editor.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:uuid/uuid.dart';

Future<Post?> showPostComposerSheet(
  BuildContext context, {
  Post? replyTarget,
  Post? quoteTarget,
}) {
  return Navigator.of(context, rootNavigator: true).push<Post?>(
    MaterialPageRoute<Post?>(
      fullscreenDialog: true,
      builder: (_) => PostComposerSheet(
        replyTarget: replyTarget,
        quoteTarget: quoteTarget,
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
  }) : assert(
         replyTarget == null || quoteTarget == null,
         'replyTarget and quoteTarget are mutually exclusive',
       );

  static const maxCharacters = 2000;

  final Post? replyTarget;
  final Post? quoteTarget;
  final String? composerId;

  @override
  ConsumerState<PostComposerSheet> createState() => _PostComposerSheetState();
}

class _PostComposerSheetState extends ConsumerState<PostComposerSheet> {
  final _controller = FacetTextEditingController();
  final _focusNode = FocusNode(debugLabel: 'postComposerText');
  late final String _composerId;
  String _initialText = '';
  String _text = '';

  @override
  void initState() {
    super.initState();
    _composerId = widget.composerId ?? const Uuid().v4();
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
    _controller.dispose();
    _focusNode.dispose();
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
    final isReply = widget.replyTarget != null;
    final isQuote = widget.quoteTarget != null;
    final trimmedText = _text.trim();
    final tooLong = _text.length > PostComposerSheet.maxCharacters;
    final canSubmit =
        !createState.isLoading &&
        trimmedText.isNotEmpty &&
        !tooLong &&
        imagesState.canSubmitImages();
    final submitLabel = isReply
        ? l10n.postComposeReplySubmit
        : l10n.postComposeSubmit;
    final hasDraft = _hasDraft(imagesState);

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
      canPop: !hasDraft || createState.isLoading,
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        final discard = await _confirmDiscard();
        if (!discard) return;
        if (!context.mounted) return;
        Navigator.of(context).pop();
      },
      child: Scaffold(
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
            Padding(
              padding: EdgeInsets.only(right: spacing.sp4),
              child: _PostAction(
                isSaving: createState.isLoading,
                label: submitLabel,
                onPressed: canSubmit
                    ? () => _submitPost(trimmedText: trimmedText)
                    : null,
              ),
            ),
          ],
        ),
        body: SafeArea(
          top: false,
          bottom: false,
          child: SingleChildScrollView(
            clipBehavior: Clip.none,
            padding: EdgeInsets.fromLTRB(
              spacing.sp4,
              spacing.sp5,
              spacing.sp4,
              spacing.sp7,
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
                if (!isReply) ...[
                  SizedBox(height: spacing.sp6),
                  ComposerImageAttachmentSection(
                    imagesState: imagesState,
                    enabled: !createState.isLoading,
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
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  bool _hasDraft(ComposerImagesState imagesState) {
    return _text != _initialText || imagesState.images.isNotEmpty;
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

    final facets = await ref.read(facetGeneratorProvider).generate(trimmedText);

    await ref
        .read(createPostProvider.notifier)
        .create(
          text: trimmedText,
          reply: _replyFor(widget.replyTarget),
          quote: _quoteFor(widget.quoteTarget),
          images: widget.replyTarget == null
              ? imagesState.toCreatePostImages()
              : null,
          facets: facets.isEmpty ? null : facets,
        );
  }
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
    required this.isSaving,
    required this.label,
    required this.onPressed,
  });

  final bool isSaving;
  final String label;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: onPressed,
      style: TextButton.styleFrom(
        textStyle: const TextStyle(fontWeight: FontWeight.w800),
      ),
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
