import 'package:craftsky_app/shared/image/image_cache_managers.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'image_cache_providers.g.dart';

@Riverpod(keepAlive: true)
BaseCacheManager profileImageCacheManager(Ref ref) =>
    ProfileImageCacheManager();

@Riverpod(keepAlive: true)
BaseCacheManager feedImageCacheManager(Ref ref) => FeedImageCacheManager();
