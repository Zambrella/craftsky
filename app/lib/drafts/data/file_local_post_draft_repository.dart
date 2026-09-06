import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/draft_file_store.dart';
import 'package:craftsky_app/drafts/data/draft_storage_paths.dart';
import 'package:craftsky_app/drafts/data/draft_update_plan.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/data/video_draft_quota.dart';
import 'package:craftsky_app/drafts/data/video_draft_stream_store.dart';
import 'package:craftsky_app/drafts/models/draft_manifest_codec.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/models/video_draft_descriptor.dart';
import 'package:crypto/crypto.dart';
import 'package:path/path.dart' as p;
import 'package:uuid/uuid.dart';

final class FileLocalPostDraftRepository
    implements LocalPostDraftRepository, LocalVideoDraftRepository {
  FileLocalPostDraftRepository({
    required String documentsRoot,
    required AccountKey owner,
    DraftFileStore? fileStore,
    DateTime Function()? clock,
    String Function()? nextId,
    this.videoSourceQuotaBytes = maxVideoDraftSourceBytesPerAccount,
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
  final int videoSourceQuotaBytes;
  final Map<String, Future<void>> _draftTails = {};

  @override
  Future<LocalPostDraft> save(DraftWriteRequest request) =>
      _synchronized('account-video-quota', () => _saveLocked(request));

  Future<LocalPostDraft> _saveLocked(DraftWriteRequest request) async {
    _validateRequest(request);
    final manifestPath = _paths.manifestPath(request.id);
    final draftDirectory = _paths.draftDirectory(request.id);
    final mediaDirectory = _paths.mediaDirectory(request.id);
    await _rejectSymbolicLinks([
      _paths.accountRoot,
      draftDirectory,
      mediaDirectory,
      manifestPath,
    ]);
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
    final pendingPath = _paths.pendingManifestPath(request.id, operationId);
    await _rejectSymbolicLinks([pendingPath]);
    final newlyWritten = <String>[];
    var manifestSwitched = false;

    try {
      await _files.ensureDirectory(draftDirectory);
      await _files.ensureDirectory(mediaDirectory);
      await _rejectSymbolicLinks([draftDirectory, mediaDirectory]);
      for (final write in plan.mediaWrites) {
        final target = _paths.mediaFilePath(
          request.id,
          write.descriptor.storageFileName,
        );
        await _rejectSymbolicLinks([target]);
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

      final nextVideo = await _prepareVideo(
        request: request,
        current: current,
        newlyWritten: newlyWritten,
      );

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
        video: nextVideo,
        revision: (current?.revision ?? 0) + 1,
      );
      await _files.writeBytesFlushed(
        pendingPath,
        Uint8List.fromList(utf8.encode(DraftManifestCodec.encode(next))),
      );
      await _rejectSymbolicLinks([pendingPath, manifestPath]);
      await _files.atomicReplace(
        sourcePath: pendingPath,
        targetPath: manifestPath,
      );
      manifestSwitched = true;

      for (final fileName in plan.cleanupFileNames) {
        try {
          await _files.deleteFile(
            await _verifiedMediaPath(request.id, fileName),
          );
        } on Object {
          // The committed manifest is authoritative; reconciliation retries.
        }
      }
      final retainedVideoFiles = {
        ?nextVideo?.sourceStorageFileName,
        ?nextVideo?.posterStorageFileName,
      };
      for (final fileName in {
        ?current?.video?.sourceStorageFileName,
        ?current?.video?.posterStorageFileName,
      }.difference(retainedVideoFiles)) {
        try {
          await _files.deleteFile(
            await _verifiedMediaPath(request.id, fileName),
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
            await _rejectSymbolicLinks([path]);
            await _files.deleteFile(path);
          } on Object {
            // Startup reconciliation removes unreferenced immutable files.
          }
        }
        try {
          await _rejectSymbolicLinks([pendingPath]);
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
      final manifestPath = _paths.manifestPath(draftId);
      await _rejectSymbolicLinks([
        _paths.accountRoot,
        _paths.draftDirectory(draftId),
        manifestPath,
      ]);
      final draft = await _readManifest(manifestPath);
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
      await _rejectSymbolicLinks([_paths.accountRoot]);
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
        await _verifiedMediaPath(draftId, descriptor.storageFileName),
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
  Stream<List<int>> openVideoSource(String draftId) async* {
    try {
      final draft = await get(draftId);
      final descriptor = draft.video;
      if (descriptor == null ||
          descriptor.availability != DraftVideoAvailability.available) {
        throw const DraftRepositoryException(
          DraftRepositoryFailureReason.unavailable,
        );
      }
      final path = await _verifiedMediaPath(
        draftId,
        descriptor.sourceStorageFileName,
      );
      yield* File(path).openRead();
    } on DraftRepositoryException {
      rethrow;
    } on Object {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.unavailable,
      );
    }
  }

  @override
  Future<Uint8List> readVideoPoster(String draftId) async {
    try {
      final draft = await get(draftId);
      final descriptor = draft.video;
      if (descriptor == null) {
        throw const DraftRepositoryException(
          DraftRepositoryFailureReason.unavailable,
        );
      }
      return _files.readBytes(
        await _verifiedMediaPath(draftId, descriptor.posterStorageFileName),
      );
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
      await _rejectSymbolicLinks([_paths.accountRoot, directory]);
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
          await _verifiedMediaPath(draft.id, descriptor.storageFileName),
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
    final video = draft.video;
    VideoDraftDescriptor? verifiedVideo;
    if (video != null) {
      final sourceAvailable = await _matchesFile(
        draft.id,
        video.sourceStorageFileName,
        video.sourceByteLength,
        video.sourceSha256,
      );
      final posterAvailable = await _matchesFile(
        draft.id,
        video.posterStorageFileName,
        video.posterByteLength,
        video.posterSha256,
      );
      final available = sourceAvailable && posterAvailable;
      if (!available) draftAvailable = false;
      verifiedVideo = video.withAvailability(
        available
            ? DraftVideoAvailability.available
            : DraftVideoAvailability.unavailable,
      );
    }
    return draft.withStorageState(
      availability: draftAvailable
          ? LocalPostDraftAvailability.available
          : LocalPostDraftAvailability.unavailable,
      media: media,
      video: verifiedVideo,
    );
  }

  Future<void> _reconcileDraftArtifacts(LocalPostDraft draft) async {
    await _deletePendingManifests(draft.id);
    final mediaDirectory = _paths.mediaDirectory(draft.id);
    try {
      await _rejectSymbolicLinks([mediaDirectory]);
    } on DraftRepositoryException {
      return;
    }
    if (!await _files.directoryExists(mediaDirectory)) return;
    final referenced = {
      for (final descriptor in draft.media) descriptor.storageFileName,
      ?draft.video?.sourceStorageFileName,
      ?draft.video?.posterStorageFileName,
    };
    for (final filePath in await _files.listChildFiles(mediaDirectory)) {
      if (!_isDirectChild(mediaDirectory, filePath)) continue;
      try {
        await _rejectSymbolicLinks([filePath]);
      } on DraftRepositoryException {
        continue;
      }
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
    final draftDirectory = _paths.draftDirectory(draftId);
    try {
      await _rejectSymbolicLinks([draftDirectory]);
    } on DraftRepositoryException {
      return;
    }
    for (final filePath in await _files.listChildFiles(draftDirectory)) {
      if (!_isDirectChild(draftDirectory, filePath)) continue;
      try {
        await _rejectSymbolicLinks([filePath]);
      } on DraftRepositoryException {
        continue;
      }
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
    if (request.owner != _owner ||
        !contentMatchesKind ||
        request.video != null && request.orderedMedia.isNotEmpty) {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.invalidRequest,
      );
    }
  }

  Future<VideoDraftDescriptor?> _prepareVideo({
    required DraftWriteRequest request,
    required LocalPostDraft? current,
    required List<String> newlyWritten,
  }) async {
    final write = request.video;
    if (write == null) return null;
    switch (write) {
      case ExistingStoredVideo():
        final stored = current?.video;
        if (stored == null ||
            stored.storageRevision != write.storageRevision ||
            stored.sourceSha256 != write.expectedSourceSha256 ||
            stored.posterSha256 != write.expectedPosterSha256 ||
            write.altText.length > 1000) {
          throw const DraftRepositoryException(
            DraftRepositoryFailureReason.invalidRequest,
          );
        }
        return VideoDraftDescriptor(
          storageRevision: stored.storageRevision,
          sourceStorageFileName: stored.sourceStorageFileName,
          posterStorageFileName: stored.posterStorageFileName,
          displayFileName: stored.displayFileName,
          mimeType: stored.mimeType,
          sourceByteLength: stored.sourceByteLength,
          sourceSha256: stored.sourceSha256,
          posterByteLength: stored.posterByteLength,
          posterSha256: stored.posterSha256,
          posterMimeType: stored.posterMimeType,
          width: stored.width,
          height: stored.height,
          duration: stored.duration,
          altText: write.altText,
        );
      case PreparedDraftVideo():
        await _checkVideoQuota(
          replacingSourceBytes: current?.video?.sourceByteLength ?? 0,
          nextSourceBytes: write.byteLength,
        );
        final revision = _nextId();
        final sourceExtension = switch (write.mimeType) {
          'video/mp4' => 'mp4',
          'video/quicktime' => 'mov',
          'video/webm' => 'webm',
          _ => throw const DraftRepositoryException(
            DraftRepositoryFailureReason.invalidRequest,
          ),
        };
        final posterExtension = switch (write.posterMimeType) {
          'image/jpeg' => 'jpg',
          'image/png' => 'png',
          _ => throw const DraftRepositoryException(
            DraftRepositoryFailureReason.invalidRequest,
          ),
        };
        final sourceName = 'video-$revision.$sourceExtension';
        final posterName = 'video-poster-$revision.$posterExtension';
        final sourcePath = _paths.mediaFilePath(request.id, sourceName);
        final posterPath = _paths.mediaFilePath(request.id, posterName);
        await _rejectSymbolicLinks([sourcePath, posterPath]);
        final sourceResult = await writeVideoDraftStream(
          source: write.openSource(),
          targetPath: sourcePath,
          maximumBytes: write.byteLength,
        );
        newlyWritten.add(sourcePath);
        if (sourceResult.byteLength != write.byteLength) {
          throw const DraftRepositoryException(
            DraftRepositoryFailureReason.invalidRequest,
          );
        }
        await _files.writeBytesFlushed(posterPath, write.posterBytes);
        newlyWritten.add(posterPath);
        final descriptor = VideoDraftDescriptor(
          storageRevision: revision,
          sourceStorageFileName: sourceName,
          posterStorageFileName: posterName,
          displayFileName: write.displayFileName,
          mimeType: write.mimeType,
          sourceByteLength: sourceResult.byteLength,
          sourceSha256: sourceResult.sha256,
          posterByteLength: write.posterBytes.length,
          posterSha256: sha256.convert(write.posterBytes).toString(),
          posterMimeType: write.posterMimeType,
          width: write.width,
          height: write.height,
          duration: write.duration,
          altText: write.altText,
        )..validate();
        if (!await _matchesFile(
          request.id,
          posterName,
          descriptor.posterByteLength,
          descriptor.posterSha256,
        )) {
          throw const DraftRepositoryException(
            DraftRepositoryFailureReason.storageUnavailable,
          );
        }
        return descriptor;
    }
  }

  Future<void> _checkVideoQuota({
    required int replacingSourceBytes,
    required int nextSourceBytes,
  }) async {
    var used = 0;
    if (await _files.directoryExists(_paths.accountRoot)) {
      for (final directory in await _files.listChildDirectories(
        _paths.accountRoot,
      )) {
        try {
          final manifest = await _readExisting(
            p.join(directory, 'manifest.json'),
          );
          used += manifest?.video?.sourceByteLength ?? 0;
        } on Object {
          // Unreadable drafts are retained and do not surrender quota.
          used = videoSourceQuotaBytes;
          break;
        }
      }
    }
    if (nextSourceBytes <= 0 ||
        used - replacingSourceBytes + nextSourceBytes > videoSourceQuotaBytes) {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.quotaExceeded,
      );
    }
  }

  Future<bool> _matchesFile(
    String draftId,
    String fileName,
    int expectedLength,
    String expectedSha256,
  ) async {
    try {
      final path = await _verifiedMediaPath(draftId, fileName);
      final digestOutput = _DraftDigestSink();
      final digestInput = sha256.startChunkedConversion(digestOutput);
      var length = 0;
      await for (final chunk in File(path).openRead()) {
        length += chunk.length;
        digestInput.add(chunk);
      }
      digestInput.close();
      return length == expectedLength &&
          digestOutput.value.toString() == expectedSha256;
    } on Object {
      return false;
    }
  }

  Future<String> _verifiedMediaPath(String draftId, String fileName) async {
    final mediaDirectory = _paths.mediaDirectory(draftId);
    final filePath = _paths.mediaFilePath(draftId, fileName);
    await _rejectSymbolicLinks([
      _paths.accountRoot,
      _paths.draftDirectory(draftId),
      mediaDirectory,
      filePath,
    ]);
    return filePath;
  }

  Future<void> _rejectSymbolicLinks(Iterable<String> paths) async {
    final checked = <String>{};
    for (final target in paths) {
      for (final path in _paths.protectedComponentsTo(target)) {
        if (checked.add(path) && await _files.isSymbolicLink(path)) {
          throw const DraftRepositoryException(
            DraftRepositoryFailureReason.unavailable,
          );
        }
      }
    }
  }

  bool _isDirectChild(String directory, String candidate) =>
      p.dirname(p.normalize(candidate)) == p.normalize(directory);

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

final class _DraftDigestSink implements Sink<Digest> {
  Digest? _digest;

  Digest get value => _digest ?? (throw StateError('Digest is not complete'));

  @override
  void add(Digest data) => _digest = data;

  @override
  void close() {}
}
