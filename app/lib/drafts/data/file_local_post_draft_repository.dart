import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/draft_file_store.dart';
import 'package:craftsky_app/drafts/data/draft_storage_paths.dart';
import 'package:craftsky_app/drafts/data/draft_update_plan.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/draft_manifest_codec.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:crypto/crypto.dart';
import 'package:path/path.dart' as p;
import 'package:uuid/uuid.dart';

final class FileLocalPostDraftRepository implements LocalPostDraftRepository {
  FileLocalPostDraftRepository({
    required String documentsRoot,
    required AccountKey owner,
    DraftFileStore? fileStore,
    DateTime Function()? clock,
    String Function()? nextId,
  }) : _owner = owner,
       _paths = DraftStoragePaths(documentsRoot: documentsRoot, owner: owner),
       _files = fileStore ?? IoDraftFileStore(),
       _clock = clock ?? DateTime.now,
       _nextId = nextId ?? const Uuid().v4;

  final AccountKey _owner;
  final DraftStoragePaths _paths;
  final DraftFileStore _files;
  final DateTime Function() _clock;
  final String Function() _nextId;
  final Map<String, Future<void>> _draftTails = {};

  @override
  Future<LocalPostDraft> save(DraftWriteRequest request) =>
      _synchronized(request.id, () => _saveLocked(request));

  Future<LocalPostDraft> _saveLocked(DraftWriteRequest request) async {
    _validateRequest(request);
    final manifestPath = _paths.manifestPath(request.id);
    final current = await _readExisting(manifestPath);
    if (current == null && request.expectedRevision != null ||
        current != null && request.expectedRevision != current.revision) {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.conflict,
      );
    }

    final operationId = _nextId();
    final plan = DraftUpdatePlan.build(
      currentMedia: current?.media ?? const [],
      orderedMedia: request.orderedMedia,
      nextStorageRevision: _nextId,
    );
    final draftDirectory = _paths.draftDirectory(request.id);
    final mediaDirectory = _paths.mediaDirectory(request.id);
    final pendingPath = _paths.pendingManifestPath(request.id, operationId);
    final newlyWritten = <String>[];
    var manifestSwitched = false;

    try {
      await _files.ensureDirectory(draftDirectory);
      await _files.ensureDirectory(mediaDirectory);
      for (final write in plan.mediaWrites) {
        final target = _paths.mediaFilePath(
          request.id,
          write.descriptor.storageFileName,
        );
        await _files.writeBytesFlushed(target, write.bytes);
        newlyWritten.add(target);
        final verified = await _files.readBytes(target);
        if (verified.length != write.descriptor.byteLength ||
            sha256.convert(verified).toString() != write.descriptor.sha256) {
          throw const DraftRepositoryException(
            DraftRepositoryFailureReason.storageUnavailable,
          );
        }
      }

      final timestamp = _clock().toUtc();
      final next = LocalPostDraft(
        id: request.id,
        owner: request.owner,
        kind: request.kind,
        createdAt:
            current?.createdAt ?? request.createdAt?.toUtc() ?? timestamp,
        updatedAt: timestamp,
        content: request.content,
        schedule: request.schedule,
        media: plan.nextMedia,
        revision: (current?.revision ?? 0) + 1,
      );
      await _files.writeBytesFlushed(
        pendingPath,
        Uint8List.fromList(utf8.encode(DraftManifestCodec.encode(next))),
      );
      await _files.atomicReplace(
        sourcePath: pendingPath,
        targetPath: manifestPath,
      );
      manifestSwitched = true;

      for (final fileName in plan.cleanupFileNames) {
        try {
          await _files.deleteFile(
            _paths.mediaFilePath(request.id, fileName),
          );
        } on Object {
          // The committed manifest is authoritative; reconciliation retries.
        }
      }
      return next;
    } on DraftRepositoryException {
      rethrow;
    } on Object {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.storageUnavailable,
      );
    } finally {
      if (!manifestSwitched) {
        for (final path in newlyWritten) {
          try {
            await _files.deleteFile(path);
          } on Object {
            // Startup reconciliation removes unreferenced immutable files.
          }
        }
        try {
          await _files.deleteFile(pendingPath);
        } on Object {
          // Startup reconciliation removes incomplete pending manifests.
        }
      }
    }
  }

  @override
  Future<LocalPostDraft> get(String draftId) async {
    try {
      final draft = await _readManifest(_paths.manifestPath(draftId));
      if (draft.owner != _owner) {
        throw const DraftRepositoryException(
          DraftRepositoryFailureReason.unavailable,
        );
      }
      return _withVerifiedMedia(draft);
    } on DraftRepositoryException {
      rethrow;
    } on Object {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.unavailable,
      );
    }
  }

  @override
  Future<List<LocalPostDraft>> list() async {
    try {
      if (!await _files.directoryExists(_paths.accountRoot)) return const [];
      final directories = await _files.listChildDirectories(_paths.accountRoot);
      final drafts = <LocalPostDraft>[];
      for (final directory in directories) {
        final draftId = p.basename(directory);
        try {
          final draft = await get(draftId);
          drafts.add(draft);
          await _reconcileDraftArtifacts(draft);
        } on Object {
          try {
            _paths.draftDirectory(draftId);
            drafts.add(LocalPostDraft.unavailable(id: draftId, owner: _owner));
            await _deletePendingManifests(draftId);
          } on Object {
            // Ignore non-draft directories without exposing their names.
          }
        }
      }
      drafts.sort((left, right) {
        final updated = right.updatedAt.compareTo(left.updatedAt);
        return updated != 0 ? updated : left.id.compareTo(right.id);
      });
      return drafts;
    } on Object {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.storageUnavailable,
      );
    }
  }

  @override
  Future<Uint8List> readMedia(String draftId, String mediaId) async {
    try {
      final draft = await get(draftId);
      final descriptor = draft.media.singleWhere(
        (candidate) => candidate.mediaId == mediaId,
      );
      final bytes = await _files.readBytes(
        _paths.mediaFilePath(draftId, descriptor.storageFileName),
      );
      if (bytes.length != descriptor.byteLength ||
          sha256.convert(bytes).toString() != descriptor.sha256) {
        throw const DraftRepositoryException(
          DraftRepositoryFailureReason.unavailable,
        );
      }
      return bytes;
    } on DraftRepositoryException {
      rethrow;
    } on Object {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.unavailable,
      );
    }
  }

  @override
  Future<void> delete(String draftId) => _synchronized(draftId, () async {
    try {
      final directory = _paths.draftDirectory(draftId);
      if (!await _files.directoryExists(directory)) return;
      await _files.deleteDirectory(directory);
    } on Object {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.storageUnavailable,
      );
    }
  });

  Future<LocalPostDraft?> _readExisting(String manifestPath) async {
    if (!await _files.fileExists(manifestPath)) return null;
    return _readManifest(manifestPath);
  }

  Future<LocalPostDraft> _readManifest(String path) async {
    final bytes = await _files.readBytes(path);
    return DraftManifestCodec.decode(utf8.decode(bytes));
  }

  Future<LocalPostDraft> _withVerifiedMedia(LocalPostDraft draft) async {
    var draftAvailable = true;
    final media = <DraftMediaDescriptor>[];
    for (final descriptor in draft.media) {
      var available = false;
      try {
        final bytes = await _files.readBytes(
          _paths.mediaFilePath(draft.id, descriptor.storageFileName),
        );
        available =
            bytes.length == descriptor.byteLength &&
            sha256.convert(bytes).toString() == descriptor.sha256;
      } on Object {
        available = false;
      }
      if (!available) draftAvailable = false;
      media.add(
        descriptor.withAvailability(
          available
              ? DraftMediaAvailability.available
              : DraftMediaAvailability.unavailable,
        ),
      );
    }
    return draft.withStorageState(
      availability: draftAvailable
          ? LocalPostDraftAvailability.available
          : LocalPostDraftAvailability.unavailable,
      media: media,
    );
  }

  Future<void> _reconcileDraftArtifacts(LocalPostDraft draft) async {
    await _deletePendingManifests(draft.id);
    final mediaDirectory = _paths.mediaDirectory(draft.id);
    if (!await _files.directoryExists(mediaDirectory)) return;
    final referenced = {
      for (final descriptor in draft.media) descriptor.storageFileName,
    };
    for (final filePath in await _files.listChildFiles(mediaDirectory)) {
      final fileName = p.basename(filePath);
      if (!referenced.contains(fileName)) {
        try {
          await _files.deleteFile(filePath);
        } on Object {
          // A later reconciliation pass retries coarse cleanup failures.
        }
      }
    }
  }

  Future<void> _deletePendingManifests(String draftId) async {
    for (final filePath in await _files.listChildFiles(
      _paths.draftDirectory(draftId),
    )) {
      final name = p.basename(filePath);
      if (name.startsWith('.pending-') && name.endsWith('-manifest.json')) {
        try {
          await _files.deleteFile(filePath);
        } on Object {
          // A later reconciliation pass retries coarse cleanup failures.
        }
      }
    }
  }

  void _validateRequest(DraftWriteRequest request) {
    // Path construction also validates the opaque ID without logging it.
    _paths.draftDirectory(request.id);
    final contentMatchesKind = switch ((request.kind, request.content)) {
      (LocalPostDraftKind.standard, StandardDraftContent()) => true,
      (LocalPostDraftKind.project, ProjectDraftContent()) => true,
      _ => false,
    };
    if (request.owner != _owner || !contentMatchesKind) {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.invalidRequest,
      );
    }
  }

  Future<T> _synchronized<T>(String key, Future<T> Function() operation) async {
    final previous = _draftTails[key] ?? Future<void>.value();
    final release = Completer<void>();
    final tail = previous.then((_) {
      return release.future;
    });
    _draftTails[key] = tail;
    await previous;
    try {
      return await operation();
    } finally {
      release.complete();
      if (identical(_draftTails[key], tail)) {
        final removed = _draftTails.remove(key);
        if (removed != null) unawaited(removed);
      }
    }
  }

  @override
  String toString() => 'FileLocalPostDraftRepository(<redacted>)';
}
