import 'dart:typed_data';

import 'package:craftsky_app/scheduled_posts/composer/schedule_composer_state.dart';

enum PrivateStagingPhase { idle, staging, creating, failed, succeeded }

enum PrivateStagingFailureStage { staging, creation }

final class PrivateMediaSource {
  const PrivateMediaSource({
    required this.id,
    required this.bytes,
    required this.mimeType,
  });

  final String id;
  final Uint8List bytes;
  final String mimeType;

  @override
  String toString() => 'PrivateMediaSource [REDACTED]';
}

final class PrivateStagingProgress {
  const PrivateStagingProgress({required this.sent, required this.total});

  final int sent;
  final int total;

  double? get fraction => total > 0 ? (sent / total).clamp(0, 1) : null;
}

final class PrivateStagingState {
  PrivateStagingState._({
    required this.operationId,
    required List<PrivateMediaSource> sources,
    required this.phase,
    required Map<String, PrivateStagingProgress> progress,
    required Map<String, String> stagedMediaIds,
    this.failureStage,
  }) : sources = List.unmodifiable(sources),
       progress = Map.unmodifiable(progress),
       stagedMediaIds = Map.unmodifiable(stagedMediaIds);

  factory PrivateStagingState.initial({
    required String operationId,
    required List<PrivateMediaSource> sources,
  }) {
    return PrivateStagingState._(
      operationId: operationId,
      sources: sources,
      phase: PrivateStagingPhase.idle,
      progress: const {},
      stagedMediaIds: const {},
    );
  }

  final String operationId;
  final List<PrivateMediaSource> sources;
  final PrivateStagingPhase phase;
  final Map<String, PrivateStagingProgress> progress;
  final Map<String, String> stagedMediaIds;
  final PrivateStagingFailureStage? failureStage;

  PrivateStagingState begin(ScheduleChoice choice) {
    if (choice == ScheduleChoice.now) return this;
    return _copyWith(
      phase: sources.isEmpty
          ? PrivateStagingPhase.creating
          : PrivateStagingPhase.staging,
    );
  }

  PrivateStagingState reportProgress(
    String sourceId, {
    required int sent,
    required int total,
  }) {
    _source(sourceId);
    return _copyWith(
      phase: PrivateStagingPhase.staging,
      progress: {
        ...progress,
        sourceId: PrivateStagingProgress(sent: sent, total: total),
      },
    );
  }

  PrivateStagingState markStaged({
    required String sourceId,
    required String mediaId,
  }) {
    _source(sourceId);
    final nextMedia = {...stagedMediaIds, sourceId: mediaId};
    return _copyWith(
      phase: nextMedia.length == sources.length
          ? PrivateStagingPhase.creating
          : PrivateStagingPhase.staging,
      stagedMediaIds: nextMedia,
      clearFailure: true,
    );
  }

  PrivateStagingState fail(PrivateStagingFailureStage stage) {
    return _copyWith(
      phase: PrivateStagingPhase.failed,
      failureStage: stage,
    );
  }

  PrivateStagingState retry() {
    return _copyWith(
      phase: stagedMediaIds.length == sources.length
          ? PrivateStagingPhase.creating
          : PrivateStagingPhase.staging,
      clearFailure: true,
    );
  }

  PrivateMediaSource _source(String id) {
    return sources.firstWhere(
      (source) => source.id == id,
      orElse: () => throw ArgumentError.value(id, 'sourceId'),
    );
  }

  PrivateStagingState _copyWith({
    PrivateStagingPhase? phase,
    Map<String, PrivateStagingProgress>? progress,
    Map<String, String>? stagedMediaIds,
    PrivateStagingFailureStage? failureStage,
    bool clearFailure = false,
  }) {
    return PrivateStagingState._(
      operationId: operationId,
      sources: sources,
      phase: phase ?? this.phase,
      progress: progress ?? this.progress,
      stagedMediaIds: stagedMediaIds ?? this.stagedMediaIds,
      failureStage: clearFailure ? null : failureStage ?? this.failureStage,
    );
  }

  @override
  String toString() => 'PrivateStagingState [REDACTED]';
}
