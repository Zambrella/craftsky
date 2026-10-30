import 'package:flutter/widgets.dart';

class ComposerImageKeys {
  ValueKey<String> add(String prefix) => ValueKey('$prefix-add-image');
  ValueKey<String> details(String prefix) => ValueKey('$prefix-edit-images');
  ValueKey<String> edit(String prefix, String imageId) =>
      ValueKey('$prefix-edit-image-$imageId');
  ValueKey<String> page(String prefix) => ValueKey('$prefix-photo-details');
  ValueKey<String> remove(String prefix, String imageId) =>
      ValueKey('$prefix-remove-$imageId');
  ValueKey<String> section(String prefix) => ValueKey('$prefix-image-section');
}
