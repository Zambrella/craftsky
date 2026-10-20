import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/widgets.dart';

class FeedKeys {
  ValueKey<String> action(String tooltip) => ValueKey('feed-action-$tooltip');
  final compose = const ValueKey<String>('feed-compose');
  final composerBody = const ValueKey<String>('post-composer-body');
  final delete = const ValueKey<String>('feed-delete');
  final menu = const ValueKey<String>('feed-menu');
  ValueKey<String> post(AtUri uri) => ValueKey('feed-post-$uri');
  final regularPost = const ValueKey<String>('feed-regular-post');
  final submit = const ValueKey<String>('post-composer-primary-action');
}
