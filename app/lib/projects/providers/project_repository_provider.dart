import 'package:craftsky_app/projects/data/api_project_repository.dart';
import 'package:craftsky_app/projects/data/project_repository.dart';
import 'package:craftsky_app/projects/providers/project_api_client_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'project_repository_provider.g.dart';

@Riverpod(keepAlive: true)
ProjectRepository projectRepository(Ref ref) =>
    ApiProjectRepository(ref.watch(projectApiClientProvider));
