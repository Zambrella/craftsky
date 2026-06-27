part of 'search_page.dart';

class _BlankSearchView {
  const _BlankSearchView._();

  static List<Widget> slivers({
    required BuildContext context,
    required WidgetRef ref,
    required ValueChanged<String> onOpenQuery,
    required ValueChanged<String> onOpenHashtag,
    required ValueChanged<String> onOpenProfile,
  }) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final blankAsync = ref.watch(blankSearchProvider);
    return switch (blankAsync) {
      AsyncValue(:final value?) => [
        SliverPadding(
          padding: EdgeInsets.fromLTRB(
            spacing.sp4,
            spacing.sp2,
            spacing.sp4,
            spacing.sp5,
          ),
          sliver: SliverList.list(
            children: [
              _RecentSearchSection(
                data: value,
                onOpenQuery: onOpenQuery,
                onOpenHashtag: onOpenHashtag,
                onOpenProfile: onOpenProfile,
              ),
              SizedBox(height: spacing.sp5),
              _TopHashtagSection(data: value, onOpenHashtag: onOpenHashtag),
            ],
          ),
        ),
      ],
      _ when blankAsync.hasError => [
        _ErrorView(
          message: l10n.searchLoadError,
          onRetry: () => ref.invalidate(blankSearchProvider),
        ),
      ],
      _ => const [
        SliverFillRemaining(
          hasScrollBody: false,
          child: Center(child: StitchProgressIndicator()),
        ),
      ],
    };
  }
}

class _RecentSearchSection extends ConsumerWidget {
  const _RecentSearchSection({
    required this.data,
    required this.onOpenQuery,
    required this.onOpenHashtag,
    required this.onOpenProfile,
  });

  final BlankSearchData data;
  final ValueChanged<String> onOpenQuery;
  final ValueChanged<String> onOpenHashtag;
  final ValueChanged<String> onOpenProfile;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    if (data.recentSearches.items.isEmpty) return const SizedBox.shrink();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          l10n.searchRecentHeading,
          style: Theme.of(context).textTheme.titleMedium,
        ),
        for (final recent in data.recentSearches.items)
          ListTile(
            dense: true,
            contentPadding: EdgeInsets.zero,
            title: Text(recent.displayLabel),
            trailing: IconButton(
              tooltip: l10n.searchDeleteRecentAction,
              icon: const Icon(Icons.close),
              onPressed: () => ref
                  .read(deleteRecentSearchProvider.notifier)
                  .delete(recent.id),
            ),
            onTap: () => _openRecent(recent),
          ),
      ],
    );
  }

  void _openRecent(RecentSearchItem recent) {
    switch (recent.payload) {
      case QueryRecentSearchPayload(:final q):
        onOpenQuery(q);
      case HashtagRecentSearchPayload(:final tag):
        onOpenHashtag(tag);
      case ProfileRecentSearchPayload(:final handle):
        onOpenProfile(handle);
      case PostRecentSearchPayload(:final q):
        onOpenQuery(q);
      case ProjectRecentSearchPayload(:final q):
        if (q != null && q.isNotEmpty) onOpenQuery(q);
    }
  }
}

class _TopHashtagSection extends StatelessWidget {
  const _TopHashtagSection({required this.data, required this.onOpenHashtag});

  final BlankSearchData data;
  final ValueChanged<String> onOpenHashtag;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final groups = data.topHashtags.groups;
    if (groups.isEmpty) return const SizedBox.shrink();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          l10n.searchTrendingHashtagsHeading,
          style: Theme.of(context).textTheme.titleMedium,
        ),
        for (final group in groups) ...[
          SizedBox(height: spacing.sp3),
          Text(
            _optionLabel(ProjectOptionCatalogs.craftTypes, group.craftType),
            style: Theme.of(context).textTheme.titleSmall,
          ),
          if (group.items.isEmpty)
            Text(l10n.searchEmptyTags)
          else
            Wrap(
              spacing: spacing.sp2,
              runSpacing: spacing.sp2,
              children: [
                for (final item in group.items)
                  ActionChip(
                    label: Text('#${item.tag}'),
                    onPressed: () => onOpenHashtag(item.tag),
                  ),
              ],
            ),
        ],
      ],
    );
  }
}
