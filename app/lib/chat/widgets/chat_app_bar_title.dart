import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:material_ui/material_ui.dart';

/// Shares background synchronization state without adding layout above a page.
class ChatBackgroundLoading extends InheritedWidget {
  const new({required this.loading, required super.child, super.key});
  final bool loading;

  static bool of(BuildContext context) =>
      context
          .dependOnInheritedWidgetOfExactType<ChatBackgroundLoading>()
          ?.loading ??
      false;

  @override
  bool updateShouldNotify(ChatBackgroundLoading oldWidget) =>
      loading != oldWidget.loading;
}

/// Refresh feedback stays beside the title, leaving private content in place.
class ChatAppBarTitle extends StatelessWidget {
  const new({
    required this.child,
    required this.loading,
    this.showBackgroundLoading = true,
    super.key,
  });
  final Widget child;
  final bool loading;
  final bool showBackgroundLoading;

  @override
  Widget build(BuildContext context) => Row(
    mainAxisSize: MainAxisSize.min,
    children: [
      Flexible(child: child),
      if (loading ||
          (showBackgroundLoading && ChatBackgroundLoading.of(context)))
        const Padding(
          padding: EdgeInsetsDirectional.only(start: 8),
          child: StitchProgressIndicator(size: 16),
        ),
    ],
  );
}

class ChatLoadingPage extends StatelessWidget {
  const new({required this.title, this.onClose, super.key});
  final String title;
  final VoidCallback? onClose;

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      leading: onClose == null ? null : BackButton(onPressed: onClose),
      title: Text(title, maxLines: 1, overflow: TextOverflow.ellipsis),
    ),
    body: const Center(child: StitchProgressIndicator()),
  );
}
