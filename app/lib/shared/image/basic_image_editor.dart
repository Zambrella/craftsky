import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/shared/image/basic_image_editor_page.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

enum BasicImageEditMode { composer, avatar }

/// Encoded, already normalized local pixels. Never serialized or logged.
class BasicImageEditRequest {
  const new({
    required this.bytes,
    required this.mimeType,
    required this.mode,
  });
  final Uint8List bytes;
  final String mimeType;
  final BasicImageEditMode mode;
  @override
  String toString() => 'BasicImageEditRequest(<redacted>)';
}

class BasicImageEditOutput {
  const new({required this.bytes, required this.mimeType});
  final Uint8List bytes;
  final String mimeType;
  @override
  String toString() => 'BasicImageEditOutput(<redacted>)';
}

typedef BasicImageEditor = Future<BasicImageEditOutput?> Function(
  BasicImageEditRequest request,
);

Future<BasicImageEditOutput?> showBasicImageEditor(
  BuildContext context,
  BasicImageEditRequest request, {
  ActiveAccountLease? ownership,
}) async {
  if (!context.mounted) return null;
  final navigator = Navigator.of(context);
  Object? failure;
  StackTrace? failureStack;
  late final MaterialPageRoute<BasicImageEditOutput> route;
  route = MaterialPageRoute<BasicImageEditOutput>(
    settings: const RouteSettings(name: 'photo-editor'),
    builder: (_) => BasicImageEditorPage(
      request: request,
      onExportError: (error, stack) {
        failure = error;
        failureStack = stack;
        if (route.isActive) navigator.removeRoute(route);
      },
    ),
  );
  final container = ownership == null
      ? null
      : ProviderScope.containerOf(context, listen: false);
  if (ownership != null &&
      container?.read(sessionRegistryProvider).value?.isCurrent(ownership) !=
          true) {
    return null;
  }
  final subscription = container?.listen(sessionRegistryProvider, (_, next) {
    if (ownership != null &&
        next.value?.isCurrent(ownership) != true &&
        route.isActive) {
      navigator.removeRoute(route);
    }
  });
  try {
    final result = await navigator.push(route);
    if (failure != null) Error.throwWithStackTrace(failure!, failureStack!);
    return result;
  } finally {
    subscription?.close();
  }
}
