part of 'search_page.dart';

class _BlankSearchView {
  const new _();

  static List<Widget> slivers({
    required BuildContext context,
    required WidgetRef ref,
    required ValueChanged<String> onOpenQuery,
    required ValueChanged<String> onOpenHashtag,
    required ValueChanged<Did> onOpenProfile,
    required Future<void> Function(ProfileAccountSummary) beforeOpenMaker,
  }) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final blankAsync = ref.watch(blankSearchProvider);
    final audience = ref.watch(browsingContextProvider);
    final makers = audience is MemberBrowsingContext
        ? ref.watch(recentMembersProvider(audience.lease)).value
        : null;
    final discoverySlivers = switch (blankAsync) {
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
      _ => [
        CraftskySkeletonSliverList(
          itemBuilder: (context, index) => const AccountRowSkeleton(),
        ),
      ],
    };
    return [
      ...discoverySlivers,
      if (makers != null && makers.isNotEmpty)
        SliverPadding(
          padding: EdgeInsets.fromLTRB(
            spacing.sp4,
            0,
            spacing.sp4,
            spacing.sp5,
          ),
          sliver: SliverList.list(
            children: [
              Semantics(
                header: true,
                child: Text(
                  l10n.searchNewMakersHeading,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
              ),
              for (final maker in makers)
                ProfileAccountListTile(
                  account: maker,
                  contentPadding: EdgeInsets.zero,
                  beforeOpen: () => beforeOpenMaker(maker),
                ),
            ],
          ),
        ),
    ];
  }
}

class _RecentSearchSection extends ConsumerWidget {
  const new({
    required this.data,
    required this.onOpenQuery,
    required this.onOpenHashtag,
    required this.onOpenProfile,
  });

  final BlankSearchData data;
  final ValueChanged<String> onOpenQuery;
  final ValueChanged<String> onOpenHashtag;
  final ValueChanged<Did> onOpenProfile;

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
        for (final recent in data.recentSearches.items.take(10))
          if (recent.payload case ProfileRecentSearchPayload(
            :final did,
            :final handle,
            :final displayName,
            :final avatar,
            :final crafts,
            :final accountType,
            :final customisation,
          ))
            ProfileAccountListTile(
              contentPadding: EdgeInsets.zero,
              account: ProfileAccountSummary(
                did: did.toString(),
                handle: handle,
                isCraftskyProfile: false,
                displayName: displayName,
                avatar: avatar,
                crafts: crafts,
                accountType: accountType,
                customisation: customisation,
              ),
              trailing: IconButton(
                tooltip: l10n.searchDeleteRecentAction,
                icon: const Icon(CraftskyIconsBold.close),
                onPressed: () => ref
                    .read(deleteRecentSearchProvider.notifier)
                    .delete(recent.id),
              ),
            )
          else if (recent.payload case HashtagRecentSearchPayload(:final tag))
            _HashtagResultTile.recent(
              tag: tag,
              onTap: () => _openRecent(recent),
              trailing: IconButton(
                tooltip: l10n.searchDeleteRecentAction,
                icon: const Icon(CraftskyIconsBold.close),
                onPressed: () => ref
                    .read(deleteRecentSearchProvider.notifier)
                    .delete(recent.id),
              ),
            )
          else
            _SearchResultTile(
              leading: switch (recent.payload) {
                QueryRecentSearchPayload(:final q)
                    when q.trim().startsWith('#') =>
                  const Text('#'),
                _ => const Icon(CraftskyIcons.search),
              },
              title: recent.displayLabel,
              trailing: IconButton(
                tooltip: l10n.searchDeleteRecentAction,
                icon: const Icon(CraftskyIconsBold.close),
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
      case ProfileRecentSearchPayload(:final did):
        onOpenProfile(did);
      case PostRecentSearchPayload(:final q):
        onOpenQuery(q);
      case ProjectRecentSearchPayload(:final q):
        if (q != null && q.isNotEmpty) onOpenQuery(q);
    }
  }
}

class _TopHashtagSection extends ConsumerWidget {
  const new({required this.data, required this.onOpenHashtag});

  final BlankSearchData data;
  final ValueChanged<String> onOpenHashtag;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (data.topHashtags.groups.isEmpty) return const SizedBox.shrink();
    final audience = ref.watch(browsingContextProvider);
    if (audience is! MemberBrowsingContext) return const SizedBox.shrink();
    final crafts = ref.watch(preferredCraftLinksProvider(audience.lease)).value;
    final groupsByCraft = {
      for (final group in data.topHashtags.groups) group.craftType: group,
    };
    final groups = [
      for (final craft in crafts ?? const <ProjectOption>[])
        ?groupsByCraft[craft.value],
    ];
    if (groups.isEmpty) return const SizedBox.shrink();

    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          l10n.searchTrendingHashtagsHeading,
          style: Theme.of(context).textTheme.titleMedium,
        ),
        for (final group in groups) ...[
          SizedBox(height: spacing.sp3),
          CraftIconLabel(
            craft: group.craftType,
            label: _optionLabel(
              ProjectOptionCatalogs.craftTypes,
              group.craftType,
            ),
            style: Theme.of(context).textTheme.titleSmall,
          ),
          if (group.items.isEmpty)
            Text(l10n.searchEmptyTags)
          else
            Wrap(
              spacing: spacing.sp2,
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
