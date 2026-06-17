import 'package:dart_mappable/dart_mappable.dart';

part 'project.mapper.dart';

const knittingProjectDetailsType = 'social.craftsky.project.knitting#details';
const crochetProjectDetailsType = 'social.craftsky.project.crochet#details';
const sewingProjectDetailsType = 'social.craftsky.project.sewing#details';
const quiltingProjectDetailsType = 'social.craftsky.project.quilting#details';

@MappableClass(ignoreNull: true, includeCustomMappers: [ProjectDetailsMapper()])
class Project with ProjectMappable {
  const Project({required this.common, this.details});

  final ProjectCommon common;
  @MappableField(hook: ProjectDetailsFieldHook())
  final ProjectDetails? details;

  Map<String, dynamic> toCreateMap() => _omitEmptyProjectArrays(toMap());
}

@MappableClass(ignoreNull: true)
class ProjectCommon with ProjectCommonMappable {
  const ProjectCommon({
    required this.craftType,
    this.status,
    this.title,
    this.duration,
    this.pattern,
    this.materials,
    this.colors,
    this.designTags,
    this.tags,
  });

  final String craftType;
  final String? status;
  final String? title;
  final String? duration;
  final ProjectPattern? pattern;
  final List<ProjectMaterial>? materials;
  final List<String>? colors;
  final List<String>? designTags;
  final List<String>? tags;
}

@MappableClass(ignoreNull: true)
class ProjectMaterial with ProjectMaterialMappable {
  const ProjectMaterial({required this.text, this.facets});

  final String text;
  final List<Map<String, dynamic>>? facets;
}

@MappableClass(ignoreNull: true)
class ProjectPattern with ProjectPatternMappable {
  const ProjectPattern({
    this.url,
    this.name,
    this.nameFacets,
    this.difficulty,
    this.designer,
    this.designerFacets,
    this.publisher,
    this.publisherFacets,
  });

  final String? url;
  final String? name;
  final List<Map<String, dynamic>>? nameFacets;
  final String? difficulty;
  final String? designer;
  final List<Map<String, dynamic>>? designerFacets;
  final String? publisher;
  final List<Map<String, dynamic>>? publisherFacets;
}

@MappableClass(ignoreNull: true)
class ProjectGauge with ProjectGaugeMappable {
  const ProjectGauge({
    required this.stitches,
    required this.measurement,
    required this.unit,
    this.rows,
  });

  final int stitches;
  final int? rows;
  final int measurement;
  final String unit;
}

sealed class ProjectDetails {
  const ProjectDetails();
}

@MappableClass(ignoreNull: true)
final class KnittingProjectDetails extends ProjectDetails
    with KnittingProjectDetailsMappable {
  const KnittingProjectDetails({
    this.projectType,
    this.projectSubtype,
    this.yarnWeight,
    this.needleSizeMm,
    this.gauge,
    this.finishedSize,
  });

  final String? projectType;
  final String? projectSubtype;
  final String? yarnWeight;
  final String? needleSizeMm;
  final ProjectGauge? gauge;
  final String? finishedSize;
}

@MappableClass(ignoreNull: true)
final class CrochetProjectDetails extends ProjectDetails
    with CrochetProjectDetailsMappable {
  const CrochetProjectDetails({
    this.projectType,
    this.projectSubtype,
    this.yarnWeight,
    this.hookSizeMm,
    this.gauge,
    this.finishedSize,
  });

  final String? projectType;
  final String? projectSubtype;
  final String? yarnWeight;
  final String? hookSizeMm;
  final ProjectGauge? gauge;
  final String? finishedSize;
}

@MappableClass(ignoreNull: true)
final class SewingProjectDetails extends ProjectDetails
    with SewingProjectDetailsMappable {
  const SewingProjectDetails({
    this.projectType,
    this.projectSubtype,
    this.sizeMade,
    this.fitNotes,
  });

  final String? projectType;
  final String? projectSubtype;
  final String? sizeMade;
  final String? fitNotes;
}

@MappableClass(ignoreNull: true)
final class QuiltingProjectDetails extends ProjectDetails
    with QuiltingProjectDetailsMappable {
  const QuiltingProjectDetails({
    this.projectType,
    this.projectSubtype,
    this.size,
    this.piecingTechnique,
    this.quiltingMethod,
  });

  final String? projectType;
  final String? projectSubtype;
  final String? size;
  final String? piecingTechnique;
  final String? quiltingMethod;
}

@MappableClass(ignoreNull: true)
final class UnknownProjectDetails extends ProjectDetails
    with UnknownProjectDetailsMappable {
  const UnknownProjectDetails({required this.raw, this.type});

  final String? type;
  final Map<String, dynamic> raw;
}

class ProjectDetailsMapper extends SimpleMapper<ProjectDetails> {
  const ProjectDetailsMapper();

  @override
  ProjectDetails decode(Object value) {
    final map = Map<String, dynamic>.from(value as Map);
    return switch (map[r'$type']) {
      knittingProjectDetailsType => KnittingProjectDetailsMapper.fromMap(map),
      crochetProjectDetailsType => CrochetProjectDetailsMapper.fromMap(map),
      sewingProjectDetailsType => SewingProjectDetailsMapper.fromMap(map),
      quiltingProjectDetailsType => QuiltingProjectDetailsMapper.fromMap(map),
      final String type => UnknownProjectDetails(
        type: type,
        raw: _withoutDiscriminator(map),
      ),
      _ => UnknownProjectDetails(raw: _withoutDiscriminator(map)),
    };
  }

  @override
  Object encode(ProjectDetails self) {
    return switch (self) {
      KnittingProjectDetails() => {
        ...self.toMap(),
        r'$type': knittingProjectDetailsType,
      },
      CrochetProjectDetails() => {
        ...self.toMap(),
        r'$type': crochetProjectDetailsType,
      },
      SewingProjectDetails() => {
        ...self.toMap(),
        r'$type': sewingProjectDetailsType,
      },
      QuiltingProjectDetails() => {
        ...self.toMap(),
        r'$type': quiltingProjectDetailsType,
      },
      UnknownProjectDetails(:final type, :final raw) => {
        r'$type': ?type,
        ...raw,
      },
    };
  }
}

class ProjectDetailsFieldHook extends MappingHook {
  const ProjectDetailsFieldHook();

  @override
  Object? beforeEncode(Object? value) {
    if (value is ProjectDetails) {
      return const ProjectDetailsMapper().encode(value);
    }
    return value;
  }
}

Map<String, dynamic> _withoutDiscriminator(Map<String, dynamic> map) {
  final raw = Map<String, dynamic>.from(map);
  return raw..remove(r'$type');
}

Map<String, dynamic> _omitEmptyProjectArrays(Map<String, dynamic> map) {
  final common = map['common'];
  if (common case final Map<String, dynamic> commonMap) {
    for (final key in const ['materials', 'colors', 'designTags', 'tags']) {
      if (commonMap[key] case final List<Object?> value when value.isEmpty) {
        commonMap.remove(key);
      }
    }
  }
  return map;
}
