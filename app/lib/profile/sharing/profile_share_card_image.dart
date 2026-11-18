import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:craftsky_app/profile/sharing/craft_foil_surface.dart';
import 'package:craftsky_app/profile/sharing/profile_share_card.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/widgets.dart';

/// Prepares the fixed 1200x1500 PNG for the native share sheet.
/// Use the card's inner [ProfileShareCard.artworkKey], and pause motion before
/// capture. Screen scaling, perspective, shadow and controls are excluded.
Future<Uint8List> captureProfileShareCard(GlobalKey artworkKey) async {
  await CraftFoilSurface.prepareAssets();
  await WidgetsBinding.instance.endOfFrame;
  final boundary = artworkKey.currentContext?.findRenderObject();
  if (boundary is! RenderRepaintBoundary ||
      !boundary.hasSize ||
      boundary.size != shareCardCanvasSize) {
    throw DiagnosticStateError('Share card artwork is unavailable');
  }
  final image = await boundary.toImage(
    pixelRatio: shareCardImageSize.width / shareCardCanvasSize.width,
  );
  try {
    final data = await image.toByteData(format: ui.ImageByteFormat.png);
    if (data == null) {
      throw DiagnosticStateError('Share card PNG encoding failed');
    }
    return data.buffer.asUint8List(data.offsetInBytes, data.lengthInBytes);
  } finally {
    image.dispose();
  }
}
