import 'package:craftsky_app/feed/models/create_post_image.dart';

class CreatePostExternal {
  const CreatePostExternal({
    required this.uri,
    required this.title,
    required this.description,
    this.thumb,
  });

  final String uri;
  final String title;
  final String description;
  final CreatePostBlob? thumb;

  Map<String, dynamic> toMap() => {
    'uri': uri,
    'title': title,
    'description': description,
    if (thumb != null) 'thumb': thumb!.toMap(),
  };
}
