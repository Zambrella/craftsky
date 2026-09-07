import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:flutter/foundation.dart';

@immutable
final class PostInteractionAccountsState {
  PostInteractionAccountsState({
    required List<ProfileAccountSummary> items,
    required this.totalCount,
    this.cursor,
  }) : items = List.unmodifiable(items);

  final List<ProfileAccountSummary> items;
  final String? cursor;
  final int totalCount;

  bool get hasMore => cursor != null;
}

@immutable
final class PostQuotesState {
  PostQuotesState({required List<Post> items, this.cursor})
    : items = List.unmodifiable(items);

  final List<Post> items;
  final String? cursor;

  bool get hasMore => cursor != null;
}
