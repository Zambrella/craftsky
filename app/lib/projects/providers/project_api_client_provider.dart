import 'package:craftsky_app/projects/data/project_api_client.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'project_api_client_provider.g.dart';

@Riverpod(keepAlive: true)
ProjectApiClient projectApiClient(Ref ref) =>
    ProjectApiClient(ref.watch(dioProvider));
