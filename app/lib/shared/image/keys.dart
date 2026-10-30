import 'package:flutter/widgets.dart';

class ImageEditorKeys {
  final cancel = const ValueKey<String>('photo-editor-cancel');
  final done = const ValueKey<String>('photo-editor-done');
  // Assigned by pro_image_editor's crop toolbar, rather than CraftSky.
  final rotate = const ValueKey<String>('crop-rotate-editor-rotate-btn');
}
