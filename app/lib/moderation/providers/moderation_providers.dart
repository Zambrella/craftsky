import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/moderation/data/api_moderation_repository.dart';
import 'package:craftsky_app/moderation/data/moderation_repository.dart';
import 'package:craftsky_app/moderation/models/account_moderation.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'moderation_providers.g.dart';

const moderationAppealAddress = 'moderation@craftsky.social';
const moderationHistoryPageLimit = 20;

typedef TextCopier = Future<void> Function(String text);

final moderationMailLauncherProvider = Provider<ExternalLinkLauncher>(
  (_) => launchExternalLink,
);

final moderationTextCopierProvider = Provider<TextCopier>(
  (_) =>
      (text) => Clipboard.setData(ClipboardData(text: text)),
);

Uri moderationAppealUri(ModerationCaseReference reference) => Uri(
  scheme: 'mailto',
  path: moderationAppealAddress,
  queryParameters: {
    'subject': 'CraftSky moderation appeal ${reference.value}',
    'body': 'Please explain why you are appealing ${reference.value}.',
  },
);

@riverpod
Future<ModerationRepository> accountModerationRepository(
  Ref ref,
  AccountKey account,
) async => ApiModerationRepository(
  await ref.watch(accountDioProvider(account).future),
);

@riverpod
Future<AccountStanding> accountStanding(Ref ref, AccountKey account) async {
  final repository = await ref.watch(
    accountModerationRepositoryProvider(account).future,
  );
  return repository.getStanding();
}

final class ModerationHistoryState {
  const ModerationHistoryState({
    required this.items,
    this.cursor,
    this.loadingMore = false,
    this.loadMoreFailed = false,
  });

  final List<ModerationHistoryEntry> items;
  final String? cursor;
  final bool loadingMore;
  final bool loadMoreFailed;

  bool get hasMore => cursor != null && cursor!.isNotEmpty;

  ModerationHistoryState copyWith({
    List<ModerationHistoryEntry>? items,
    String? cursor,
    bool clearCursor = false,
    bool? loadingMore,
    bool? loadMoreFailed,
  }) => ModerationHistoryState(
    items: items ?? this.items,
    cursor: clearCursor ? null : cursor ?? this.cursor,
    loadingMore: loadingMore ?? this.loadingMore,
    loadMoreFailed: loadMoreFailed ?? this.loadMoreFailed,
  );
}

@riverpod
class AccountModerationHistory extends _$AccountModerationHistory {
  @override
  Future<ModerationHistoryState> build(AccountKey account) async {
    final repository = await ref.watch(
      accountModerationRepositoryProvider(account).future,
    );
    final page = await repository.getHistory(limit: moderationHistoryPageLimit);
    return ModerationHistoryState(items: page.items, cursor: page.cursor);
  }

  Future<void> refresh() async => ref.invalidateSelf();

  Future<void> loadMore() async {
    final current = state.value;
    if (current == null || current.loadingMore || !current.hasMore) return;
    state = AsyncData(
      current.copyWith(loadingMore: true, loadMoreFailed: false),
    );
    try {
      final repository = await ref.read(
        accountModerationRepositoryProvider(account).future,
      );
      final page = await repository.getHistory(
        cursor: current.cursor,
        limit: moderationHistoryPageLimit,
      );
      if (!ref.mounted) return;
      state = AsyncData(
        ModerationHistoryState(
          items: [...current.items, ...page.items],
          cursor: page.cursor,
        ),
      );
    } on Object {
      if (!ref.mounted) return;
      state = AsyncData(
        current.copyWith(loadingMore: false, loadMoreFailed: true),
      );
    }
  }
}

@riverpod
Future<ModerationHistoryPage> accountModerationHistoryEntry(
  Ref ref,
  AccountKey account,
  ModerationCaseReference reference,
) async {
  final repository = await ref.watch(
    accountModerationRepositoryProvider(account).future,
  );
  return repository.getHistoryEntry(reference);
}
