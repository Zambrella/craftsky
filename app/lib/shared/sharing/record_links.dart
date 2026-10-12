import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

/// Shareable links use stable identifiers and the canonical web app origin.
abstract final class RecordLinks {
  static final Uri _origin = Uri.parse('https://app.craftsky.social');

  static String post(Post post) {
    final root = post.reply?.root.uri;
    final target = root == null ? null : parseCraftskyPostUri(root);
    return _origin
        .resolve(
          PostThreadRoute(
            did: target?.did.value ?? post.author.did.value,
            rkey: target?.rkey.value ?? post.rkey.value,
            focus: root == null ? null : post.uri.value,
          ).location,
        )
        .toString();
  }

  static String profile(Did did) =>
      _origin.resolve(UserProfileRoute(did: did).location).toString();

  static String event(Did did, RecordKey rkey) => _origin
      .resolve(BusinessEventRoute(did: did.value, rkey: rkey.value).location)
      .toString();
}
