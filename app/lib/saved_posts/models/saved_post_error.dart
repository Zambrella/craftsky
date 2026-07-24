import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';

part 'saved_post_error.mapper.dart';

enum SavedPostOperation {
  loadPosts,
  loadFolders,
  createFolder,
  saveOrMove,
  unsave,
  renameFolder,
  deleteFolder,
}

enum SavedPostFailureKind {
  network,
  unauthorized,
  validation,
  server,
  canceled,
  unknown,
}

@immutable
@MappableClass(generateMethods: GenerateMethods.copy | GenerateMethods.equals)
final class SavedPostFailure with SavedPostFailureMappable {
  const SavedPostFailure({required this.kind, required this.operation});

  factory SavedPostFailure.from(
    Object error, {
    required SavedPostOperation operation,
  }) => SavedPostFailure(
    kind: switch (error) {
      ApiNetworkError() => SavedPostFailureKind.network,
      ApiUnauthorized() => SavedPostFailureKind.unauthorized,
      ApiBadRequest() => SavedPostFailureKind.validation,
      ApiServerError() => SavedPostFailureKind.server,
      ApiCanceled() => SavedPostFailureKind.canceled,
      _ => SavedPostFailureKind.unknown,
    },
    operation: operation,
  );

  final SavedPostFailureKind kind;
  final SavedPostOperation operation;

  bool get shouldPresent => kind != SavedPostFailureKind.canceled;

  bool get canRetry => switch (kind) {
    SavedPostFailureKind.network ||
    SavedPostFailureKind.server ||
    SavedPostFailureKind.unknown => true,
    SavedPostFailureKind.unauthorized ||
    SavedPostFailureKind.validation ||
    SavedPostFailureKind.canceled => false,
  };

  String localizedMessage(AppLocalizations l10n) => switch (operation) {
    SavedPostOperation.loadPosts => l10n.savedPostsLoadError,
    SavedPostOperation.loadFolders => l10n.savedPostFoldersLoadError,
    SavedPostOperation.createFolder => l10n.savedPostCreateFolderError,
    SavedPostOperation.unsave => l10n.savedPostUnsaveError,
    SavedPostOperation.saveOrMove ||
    SavedPostOperation.renameFolder ||
    SavedPostOperation.deleteFolder => l10n.savedPostConfirmError,
  };

  @override
  String toString() => 'SavedPostFailure(${kind.name}, ${operation.name})';
}
