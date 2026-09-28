import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

Iterable<Did> authorPostCacheIds(Post post) => [post.author.did];
