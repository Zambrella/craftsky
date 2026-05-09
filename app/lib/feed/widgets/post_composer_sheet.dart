import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

Future<void> showPostComposerSheet(BuildContext context) {
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    builder: (_) => const PostComposerSheet(),
  );
}

class PostComposerSheet extends ConsumerStatefulWidget {
  const PostComposerSheet({super.key});

  static const maxCharacters = 2000;

  @override
  ConsumerState<PostComposerSheet> createState() => _PostComposerSheetState();
}

class _PostComposerSheetState extends ConsumerState<PostComposerSheet> {
  final _controller = TextEditingController();
  var _text = '';

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final createState = ref.watch(createPostProvider);
    final trimmed = _text.trim();
    final tooLong = _text.length > PostComposerSheet.maxCharacters;
    final canSubmit = trimmed.isNotEmpty && !tooLong && !createState.isLoading;

    ref.listen(createPostProvider, (previous, next) {
      switch ((previous, next)) {
        case (AsyncLoading(), AsyncData(value: != null)):
          Navigator.of(context).pop();
          context.showInfo(l10n.postCreateSuccess);
          ref.read(createPostProvider.notifier).reset();
        case (AsyncLoading(), AsyncError()):
          context.showError(l10n.postCreateError);
          ref.read(createPostProvider.notifier).reset();
        case _:
          break;
      }
    });

    return Padding(
      padding: EdgeInsets.only(
        left: spacing.sp4,
        right: spacing.sp4,
        top: spacing.sp4,
        bottom: MediaQuery.viewInsetsOf(context).bottom + spacing.sp4,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(l10n.postComposeTitle, style: theme.textTheme.headlineSmall),
          SizedBox(height: spacing.sp4),
          BrandTextField(
            label: l10n.postComposeHint,
            controller: _controller,
            minLines: 5,
            maxLines: 10,
            textInputAction: TextInputAction.newline,
            keyboardType: TextInputType.multiline,
            enabled: !createState.isLoading,
            errorText: tooLong ? l10n.postComposeTooLong : null,
            helperText: '${_text.length}/${PostComposerSheet.maxCharacters}',
            onChanged: (value) => setState(() => _text = value),
          ),
          SizedBox(height: spacing.sp5),
          ChunkyButton(
            onPressed: canSubmit
                ? () => ref
                      .read(createPostProvider.notifier)
                      .create(text: trimmed)
                : null,
            child: createState.isLoading
                ? const SizedBox(
                    width: 18,
                    height: 18,
                    child: StitchProgressIndicator(strokeWidth: 2),
                  )
                : Text(l10n.postComposeSubmit),
          ),
        ],
      ),
    );
  }
}
