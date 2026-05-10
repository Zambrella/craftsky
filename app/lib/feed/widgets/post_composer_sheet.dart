import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

Future<void> showPostComposerSheet(
  BuildContext context, {
  Post? replyTarget,
}) {
  return Navigator.of(context, rootNavigator: true).push<void>(
    MaterialPageRoute<void>(
      fullscreenDialog: true,
      builder: (_) => PostComposerSheet(replyTarget: replyTarget),
    ),
  );
}

class PostComposerSheet extends ConsumerStatefulWidget {
  const PostComposerSheet({super.key, this.replyTarget});

  static const maxCharacters = 2000;

  final Post? replyTarget;

  @override
  ConsumerState<PostComposerSheet> createState() => _PostComposerSheetState();
}

class _PostComposerSheetState extends ConsumerState<PostComposerSheet> {
  final _controller = TextEditingController();
  final _focusNode = FocusNode(debugLabel: 'postComposerText');
  var _text = '';

  @override
  void initState() {
    super.initState();
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
    final isReply = widget.replyTarget != null;
    final trimmed = _text.trim();
    final tooLong = _text.length > PostComposerSheet.maxCharacters;
    final canSubmit = trimmed.isNotEmpty && !tooLong && !createState.isLoading;
    final submitLabel = isReply
        ? l10n.postComposeReplySubmit
        : l10n.postComposeSubmit;

    ref.listen(createPostProvider, (previous, next) {
      switch ((previous, next)) {
        case (AsyncLoading(), AsyncData(value: != null)):
          if (Navigator.of(context).canPop()) {
            Navigator.of(context).pop();
          }
          context.showInfo(l10n.postCreateSuccess);
          ref.read(createPostProvider.notifier).reset();
        case (AsyncLoading(), AsyncError()):
          context.showError(l10n.postCreateError);
          ref.read(createPostProvider.notifier).reset();
        case _:
          break;
      }
    });

    return Scaffold(
      backgroundColor: swatches.paper,
      appBar: AppBar(
        title: Text(
          isReply ? l10n.postComposeReplyTitle : l10n.postComposeTitle,
        ),
        actions: [
          Padding(
            padding: EdgeInsets.only(right: spacing.sp3),
            child: _PostAction(
              isSaving: createState.isLoading,
              label: submitLabel,
              onPressed: canSubmit
                  ? () => ref
                        .read(createPostProvider.notifier)
                        .create(
                          text: trimmed,
                          reply: _replyFor(widget.replyTarget),
                        )
                  : null,
            ),
          ),
        ],
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: EdgeInsets.fromLTRB(
            spacing.sp4,
            spacing.sp4,
            spacing.sp4,
            spacing.sp6,
          ),
          child: BrandTextField(
            label: isReply ? l10n.postComposeReplyHint : l10n.postComposeHint,
            controller: _controller,
            focusNode: _focusNode,
            minLines: 8,
            maxLines: 16,
            textInputAction: TextInputAction.newline,
            keyboardType: TextInputType.multiline,
            enabled: !createState.isLoading,
            errorText: tooLong ? l10n.postComposeTooLong : null,
            helperText: '${_text.length}/${PostComposerSheet.maxCharacters}',
            onChanged: (value) => setState(() => _text = value),
          ),
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
    if (isSaving) {
      return const SizedBox(
        width: 48,
        child: Center(
          child: SizedBox(
            width: 18,
            height: 18,
            child: StitchProgressIndicator(strokeWidth: 2),
          ),
        ),
      );
    }
    return TextButton(
      onPressed: onPressed,
      child: Text(label),
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
