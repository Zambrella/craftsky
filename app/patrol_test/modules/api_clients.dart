import 'dart:async';

import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/profile/data/profile_api_client.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final class ApiClients {
  new(this.container);

  final ProviderContainer container;

  Dio get _dio => container.read(dioProvider);

  Future<void> verifyBio(String bio) async {
    await _until(() async {
      final profile = await ProfileApiClient(_dio).getMyProfile();
      return profile.description == bio;
    }, 'The onboarding bio was not persisted.');
  }

  Future<Post> findPublishedPost(String text) async {
    Post? result;
    await _until(() async {
      final profile = await ProfileApiClient(_dio).getMyProfile();
      final page = await PostApiClient(_dio)
          .listPostsByAuthor(profile.did.value);
      result = page.items.where((post) => post.text == text).firstOrNull;
      return result != null;
    }, 'The published post did not reach AppView.');
    final post = result!;
    if (post.project != null || post.images?.length != 2) {
      throw StateError(
        'Expected a regular post with exactly two persisted images.',
      );
    }
    // A separate client deliberately excludes the AppView bearer header.
    final media = Dio(
      BaseOptions(
        connectTimeout: const Duration(seconds: 10),
        receiveTimeout: const Duration(seconds: 15),
      ),
    );
    try {
      for (final image in post.images!) {
        final url = image.fullsize;
        if (url == null) throw StateError('Published image URL is missing.');
        final response = await media.get<List<int>>(
          url,
          options: Options(responseType: ResponseType.bytes),
        );
        if (response.statusCode != 200 || response.data?.isNotEmpty != true) {
          throw StateError('Published image is unavailable.');
        }
      }
    } finally {
      media.close(force: true);
    }
    return post;
  }

  Future<void> verifyLiked(Post post) => _until(() async {
    final current = await PostApiClient(_dio)
        .getPost(post.author.did, post.rkey);
    return current.viewerHasLiked && current.likeCount >= 1;
  }, 'The like was not persisted.');

  Future<void> verifyEdited(Post original, String text) => _until(
    () async {
      final current = await PostApiClient(_dio)
          .getPost(original.author.did, original.rkey);
      return current.uri == original.uri &&
          current.cid != original.cid &&
          current.editedAt != null &&
          current.text == text &&
          current.images?.length == 1 &&
          current.images!.single.cid == original.images!.first.cid &&
          current.viewerHasLiked &&
          current.likeCount >= 1;
    },
    'The edited text and retained first image were not persisted, '
    'or the existing like was lost.',
  );

  Future<void> verifyDeleted(Post post) => _until(() async {
    final response = await _dio.get<Object?>(
      '/v1/posts/${post.author.did}/${post.rkey}',
      options: Options(
        validateStatus: (status) => status == 200 || status == 404,
      ),
    );
    return response.statusCode == 404;
  }, 'The deleted post remains available in AppView.');

  Future<void> _until(Future<bool> Function() ready, String failure) async {
    final deadline = DateTime.now().add(const Duration(seconds: 60));
    do {
      if (await ready()) return;
      await Future<void>.delayed(const Duration(milliseconds: 500));
    } while (DateTime.now().isBefore(deadline));
    throw StateError(failure);
  }
}
