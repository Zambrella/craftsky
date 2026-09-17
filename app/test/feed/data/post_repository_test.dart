import 'package:craftsky_app/feed/data/api_post_repository.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('REG-003 repository rejects project-plus-quote', () async {
    final repository = ApiPostRepository(PostApiClient(Dio()));
    const project = Project(
      common: ProjectCommon(
        craftType: 'social.craftsky.feed.defs#embroidery',
      ),
    );
    final quote = PostRef(
      uri: 'at://did:plc:alice/social.craftsky.feed.post/target',
      cid: 'bafy_target',
    );

    await expectLater(
      () => repository.create(
        text: 'invalid',
        langs: const ['en'],
        sponsored: false,
        project: project,
        quote: quote,
      ),
      throwsA(isA<AssertionError>()),
    );
  });
}
