import 'package:flutter_riverpod/flutter_riverpod.dart';

bool isPublicReadingLocation(String path) =>
    path == '/safety' ||
    path == '/feed' ||
    path == '/projects' ||
    path == '/search' ||
    RegExp(r'^/search/tags/[^/]+$').hasMatch(path) ||
    RegExp(r'^/profiles/(?!me$|recent$)[^/]+$').hasMatch(path) ||
    RegExp(r'^/posts/[^/]+/[^/]+(?:/(?:likes|reposts|quotes))?$')
        .hasMatch(path) ||
    RegExp(r'^/events/[^/]+/[^/]+$').hasMatch(path);

/// Process lifetime, independent of account-bound caches. No credentials/state
/// from a member are stored here and no attempted mutation is ever replayed.
final class GuestSession {
  bool explored = false;
  String returnTo = '/feed';
  bool shouldPrompt(String path) =>
      !explored && (path == '/feed' || path == '/projects');
  void explore() => explored = true;
  String consumeReturnTo() {
    final destination = returnTo;
    returnTo = '/feed';
    return destination;
  }

  void prepareSignIn(String location) {
    final uri = Uri.tryParse(location);
    returnTo =
        uri != null &&
            !uri.hasScheme &&
            !uri.hasAuthority &&
            !uri.hasFragment &&
            isPublicReadingLocation(uri.path) &&
            uri.queryParameters.keys.every(
              const {
                'q',
                'tab',
                'craft',
                'showComments',
                'initialImageCid',
              }.contains,
            )
        ? uri.toString()
        : '/feed';
  }
}

final guestSessionProvider = Provider<GuestSession>((ref) => GuestSession());

class GuestPromptOpen extends Notifier<bool> {
  @override
  bool build() => false;
  void open() => state = true;
  void close() {
    if (ref.mounted) state = false;
  }
}

final guestPromptOpenProvider = NotifierProvider<GuestPromptOpen, bool>(
  GuestPromptOpen.new,
);
