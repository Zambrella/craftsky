import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

const autoLoadMoreExtent = 240.0;

class AutoPaginatedListView extends StatelessWidget {
  const AutoPaginatedListView({
    required this.itemCount,
    required this.emptyText,
    required this.isLoadingMore,
    required this.hasLoadMoreError,
    required this.onNearEnd,
    required this.itemBuilder,
    super.key,
  });

  final int itemCount;
  final String emptyText;
  final bool isLoadingMore;
  final bool hasLoadMoreError;
  final VoidCallback onNearEnd;
  final IndexedWidgetBuilder itemBuilder;

  @override
  Widget build(BuildContext context) {
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    if (itemCount == 0) return Center(child: Text(emptyText));
    return NotificationListener<ScrollNotification>(
      onNotification: (notification) {
        if (notification.metrics.extentAfter < autoLoadMoreExtent &&
            !isLoadingMore &&
            !hasLoadMoreError) {
          onNearEnd();
        }
        return false;
      },
      child: ListView.builder(
        padding: EdgeInsets.only(bottom: spacing.sp5),
        itemCount: itemCount + (isLoadingMore || hasLoadMoreError ? 1 : 0),
        itemBuilder: (context, index) {
          if (index < itemCount) return itemBuilder(context, index);
          return Padding(
            padding: EdgeInsets.all(spacing.sp4),
            child: Center(
              child: isLoadingMore
                  ? const StitchProgressIndicator()
                  : TextButton.icon(
                      onPressed: onNearEnd,
                      icon: const Icon(Icons.refresh),
                      label: Text(AppLocalizations.of(context).retryButton),
                    ),
            ),
          );
        },
      ),
    );
  }
}
