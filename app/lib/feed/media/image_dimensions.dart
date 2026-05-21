import 'package:craftsky_app/feed/models/create_post_image.dart';

CreatePostImageAspectRatio? toOptionalAspectRatio({
  required int? width,
  required int? height,
}) {
  if (width == null || height == null) return null;
  if (width <= 0 || height <= 0) return null;

  return CreatePostImageAspectRatio(width: width, height: height);
}
