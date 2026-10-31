import 'dart:async';

import 'package:characters/characters.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/chat/models/chat_link_target.dart';
import 'package:craftsky_app/chat/models/chat_preview.dart';
import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';

final class ChatPreviewResolver {
  new({
    required this.account,
    required this._operation,
    required this._posts,
    required this._profiles,
    required this._events,
    ErrorReporter reporter = const NoopErrorReporter(),
  }) : _reporter = GuardedErrorReporter(reporter);
  final AccountKey account;
  final ActiveAccountOperation _operation;
  final PostRepository _posts;
  final ProfileRepository _profiles;
  final BusinessRepository _events;
  final ErrorReporter _reporter;
  String _excerpt(String value) => value.characters.take(160).toString();
  Uri? _thumbnail(String? raw) {
    final uri = raw == null ? null : Uri.tryParse(raw);
    return uri?.scheme == 'https' &&
            uri!.host.isNotEmpty &&
            uri.userInfo.isEmpty
        ? uri
        : null;
  }

  Future<ChatPreview?> resolveText(
    String text, {
    required bool accepted,
  }) async {
    if (!accepted || !_operation.isCurrent) return null;
    final candidate = ChatLinkTarget.firstIn(text);
    if (candidate == null) return null;
    try {
      switch (candidate.kind) {
        case ChatLinkKind.post:
          final focused = candidate.focus == null
              ? null
              : parseCraftskyPostUri(candidate.focus!);
          final post = await _operation.step(
            () => _posts.fetch(
              focused?.did ?? candidate.did,
              focused?.rkey ?? candidate.rkey!,
            ),
          );
          if (post.isProtected || post.moderation != null) return null;
          if (post.author.did != (focused?.did ?? candidate.did) ||
              post.rkey != (focused?.rkey ?? candidate.rkey)) {
            throw const FormatException('Unexpected preview content');
          }
          final root = post.reply == null
              ? null
              : parseCraftskyPostUri(post.reply!.root.uri);
          final target = ChatLinkTarget.post(
            did: root?.did ?? post.author.did,
            rkey: root?.rkey ?? post.rkey,
            focus: root == null ? null : post.uri,
          );
          return ChatPreview(
            target: target,
            title: post.author.displayName ?? post.author.handle.value,
            excerpt: _excerpt(post.text),
            postSummary: PostSummaryData.fromPost(post),
            thumbnail: _thumbnail(
              post.images?.firstOrNull?.thumb ?? post.video?.thumbnail,
            ),
          );
        case ChatLinkKind.profile:
          final profile = await _operation.step(
            () => _profiles.fetch(candidate.did.value),
          );
          if (profile.did != candidate.did) {
            throw const FormatException('Unexpected preview content');
          }
          if (profile.muted ||
              profile.blocking ||
              profile.blockedBy ||
              profile.moderation != null) {
            return null;
          }
          return ChatPreview(
            target: candidate,
            title: profile.displayName ?? profile.handle.value,
            excerpt: _excerpt(profile.description ?? ''),
            thumbnail: _thumbnail(profile.avatar),
          );
        case ChatLinkKind.event:
          final event = await _operation.step(
            () => _events.getEvent(candidate.did, candidate.rkey!),
          );
          if (event.did != candidate.did || event.rkey != candidate.rkey) {
            throw const FormatException('Unexpected preview content');
          }
          if (event.publicSuppressionReasons.isNotEmpty) return null;
          return ChatPreview(
            target: candidate,
            title: event.name,
            excerpt: _excerpt(event.summary ?? event.venueName ?? ''),
            thumbnail: _thumbnail(event.image?.thumb),
            startsAt: event.startsAt,
          );
      }
    } on AccountOperationDiscarded {
      return null;
    } on Object catch (error, stack) {
      if (error is ApiBadRequest &&
          const [403, 404, 410].contains(error.details.statusCode)) {
        return null;
      }
      if (_operation.isCurrent) {
        unawaited(
          _reporter.captureException(
            error,
            stackTrace: stack,
            context: ReportContext(
              feature: 'Chat',
              operation: 'preview',
              classification: 'chat.failure',
              workflow: PrivateOperationalFailureContext(
                accountDid: account.did.value,
              ),
            ),
          ),
        );
      }
      return null;
    }
  }
}
