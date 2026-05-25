import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/media/blob_api_client.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'blob_api_client_provider.g.dart';

@riverpod
BlobApiClient blobApiClient(Ref ref) {
  return BlobApiClient(ref.watch(dioProvider));
}
