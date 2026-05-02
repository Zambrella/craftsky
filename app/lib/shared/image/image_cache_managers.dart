import 'package:flutter_cache_manager/flutter_cache_manager.dart';

/// Long-lived disk cache for avatar and banner images. URLs from the
/// Bluesky CDN are content-addressed (CID-keyed), so cached entries can
/// never go stale-but-wrong; a long stale period and large object cap
/// are safe and avoid re-downloading reusable identity images under
/// LRU pressure from larger feed media.
class ProfileImageCacheManager extends CacheManager {
  factory ProfileImageCacheManager() => _instance;

  ProfileImageCacheManager._()
    : super(
        Config(
          _key,
          stalePeriod: const Duration(days: 90),
          maxNrOfCacheObjects: 500,
        ),
      );

  static const _key = 'craftskyProfileImages';
  static final ProfileImageCacheManager _instance =
      ProfileImageCacheManager._();
}

/// Shorter-lived disk cache reserved for feed post images. No call sites
/// in v1 — wired up so adding feed-image call sites later is a one-line
/// change.
class FeedImageCacheManager extends CacheManager {
  factory FeedImageCacheManager() => _instance;

  FeedImageCacheManager._()
    : super(
        Config(
          _key,
          stalePeriod: const Duration(days: 7),
          maxNrOfCacheObjects: 300,
        ),
      );

  static const _key = 'craftskyFeedImages';
  static final FeedImageCacheManager _instance = FeedImageCacheManager._();
}
