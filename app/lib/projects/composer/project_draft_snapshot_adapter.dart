import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/composer/draft_media_write_adapter.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/models/project.dart';

/// Converts the project composer's partial form state to and from draft-safe
/// values. Only fields owned by the composer are persisted.
class ProjectDraftSnapshotAdapter {
  const ProjectDraftSnapshotAdapter();

  DraftWriteRequest toWriteRequest({
    required String id,
    required AccountKey owner,
    required String body,
    required List<String> languages,
    required DraftScheduleIntent schedule,
    required Map<String, dynamic> formValues,
    required List<ComposerImageDraft> images,
    int? existingRevision,
    DateTime? existingCreatedAt,
  }) {
    return DraftWriteRequest(
      id: id,
      owner: owner,
      kind: LocalPostDraftKind.project,
      createdAt: existingCreatedAt,
      expectedRevision: existingRevision,
      content: ProjectDraftContent(
        body: body,
        languages: List.unmodifiable(languages),
        knownProjectFieldValues: encodeKnownFields(formValues),
      ),
      schedule: schedule,
      orderedMedia: draftMediaWritesFromComposer(images),
    );
  }

  Map<String, dynamic> encodeKnownFields(Map<String, dynamic> values) {
    return <String, dynamic>{
      for (final field in _knownFields)
        if (values.containsKey(field))
          field: _encodeValue(field, values[field]),
    };
  }

  Map<String, dynamic> decodeKnownFields(Map<String, dynamic> values) {
    return <String, dynamic>{
      for (final field in _knownFields)
        if (values.containsKey(field))
          field: _decodeValue(field, values[field]),
    };
  }

  Object? _encodeValue(String field, Object? value) {
    if (field == ProjectComposerFields.materials) {
      if (value is! List<ProjectMaterial>) return const <Object?>[];
      return [
        for (final material in value)
          <String, dynamic>{
            'text': material.text,
            if (material.facets != null)
              'facets': [
                for (final facet in material.facets!)
                  Map<String, dynamic>.from(facet),
              ],
          },
      ];
    }
    if (value == null || value is String || value is num || value is bool) {
      return value;
    }
    if (value is List) {
      return [
        for (final item in value)
          if (item == null || item is String || item is num || item is bool)
            item,
      ];
    }
    return null;
  }

  Object? _decodeValue(String field, Object? value) {
    if (_stringListFields.contains(field)) {
      if (value == null) return null;
      if (value is! List) return const <String>[];
      return List<String>.unmodifiable(value.whereType<String>());
    }
    if (field != ProjectComposerFields.materials) return value;
    if (value is! List) return const <ProjectMaterial>[];
    return <ProjectMaterial>[
      for (final item in value)
        if (item is Map && item['text'] is String)
          ProjectMaterial(
            text: item['text']! as String,
            facets: _decodeFacets(item['facets']),
          ),
    ];
  }

  List<Map<String, dynamic>>? _decodeFacets(Object? value) {
    if (value is! List) return null;
    final facets = <Map<String, dynamic>>[
      for (final item in value)
        if (item is Map) Map<String, dynamic>.from(item),
    ];
    return facets.isEmpty ? null : facets;
  }

  static const _knownFields = <String>{
    ProjectComposerFields.craftType,
    ProjectComposerFields.status,
    ProjectComposerFields.title,
    ProjectComposerFields.materials,
    ProjectComposerFields.colours,
    ProjectComposerFields.designTags,
    ProjectComposerFields.patternUrl,
    ProjectComposerFields.patternName,
    ProjectComposerFields.patternDifficulty,
    ProjectComposerFields.patternDesigner,
    ProjectComposerFields.patternPublisher,
    ProjectComposerFields.sewingProjectType,
    ProjectComposerFields.sewingProjectSubtype,
    ProjectComposerFields.sewingSizeMade,
    ProjectComposerFields.sewingFitNotes,
    ProjectComposerFields.knittingProjectType,
    ProjectComposerFields.knittingProjectSubtype,
    ProjectComposerFields.knittingYarnWeight,
    ProjectComposerFields.knittingNeedleSize,
    ProjectComposerFields.knittingGaugeStitches,
    ProjectComposerFields.knittingGaugeRows,
    ProjectComposerFields.knittingGaugeMeasurement,
    ProjectComposerFields.knittingGaugeUnit,
    ProjectComposerFields.knittingFinishedSize,
    ProjectComposerFields.crochetProjectType,
    ProjectComposerFields.crochetProjectSubtype,
    ProjectComposerFields.crochetYarnWeight,
    ProjectComposerFields.crochetHookSize,
    ProjectComposerFields.crochetGaugeStitches,
    ProjectComposerFields.crochetGaugeRows,
    ProjectComposerFields.crochetGaugeMeasurement,
    ProjectComposerFields.crochetGaugeUnit,
    ProjectComposerFields.crochetFinishedSize,
    ProjectComposerFields.quiltingProjectType,
    ProjectComposerFields.quiltingProjectSubtype,
    ProjectComposerFields.quiltingSize,
    ProjectComposerFields.quiltingPiecingTechnique,
    ProjectComposerFields.quiltingMethod,
  };

  static const _stringListFields = <String>{
    ProjectComposerFields.colours,
    ProjectComposerFields.designTags,
  };
}
