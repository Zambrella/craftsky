import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/media/blob_api_client.dart';
import 'package:dio/dio.dart';
import 'package:flutter/widgets.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'blob_api_client_provider.g.dart';

@riverpod
BlobApiClient blobApiClient(Ref ref) {
  return managedBlobApiClient(ref, ref.watch(dioProvider));
}

/// Cancel capabilities on account-provider disposal and app interruption.
BlobApiClient managedBlobApiClient(Ref ref, Dio dio) {
  final client = BlobApiClient(dio);
  final initialState = WidgetsBinding.instance.lifecycleState;
  client.setLifecycleActive(
    active: initialState == null || initialState == AppLifecycleState.resumed,
  );
  final lifecycle = AppLifecycleListener(
    onStateChange: (state) {
      client.setLifecycleActive(active: state == AppLifecycleState.resumed);
    },
  );
  ref.onDispose(() {
    lifecycle.dispose();
    client.dispose();
  });
  return client;
}
