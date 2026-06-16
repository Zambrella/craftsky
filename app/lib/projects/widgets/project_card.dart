import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/shared/rich_text/widgets/faceted_text.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart' as url_launcher;

typedef ProjectLinkLauncher = Future<bool> Function(Uri uri);

enum ProjectCardVariant { summary, detail }

class ProjectCard extends StatelessWidget {
  const ProjectCard({
    required this.project,
    this.variant = ProjectCardVariant.summary,
    this.launchUrl,
    super.key,
  });

  final Project project;
  final ProjectCardVariant variant;
  final ProjectLinkLauncher? launchUrl;

  @override
  Widget build(BuildContext context) {
    return switch (variant) {
      ProjectCardVariant.summary => _ProjectSummary(project: project),
      ProjectCardVariant.detail => _ProjectDetail(
        project: project,
        launchUrl: launchUrl ?? _defaultLaunchUrl,
      ),
    };
  }
}

class _ProjectSummary extends StatelessWidget {
  const _ProjectSummary({required this.project});

  final Project project;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final title = _nonBlank(project.common.title);
    final status = _nonBlank(project.common.status);
    final pattern = _patternValue(project.common.pattern);
    final size = _sizeMetadata(project.details);
    final headlineStyle = theme.textTheme.headlineSmall;
    final titleStyle = theme.textTheme.displaySmall?.copyWith(
      fontSize: headlineStyle?.fontSize,
      fontWeight: headlineStyle?.fontWeight,
      height: headlineStyle?.height,
      letterSpacing: headlineStyle?.letterSpacing,
      color: headlineStyle?.color,
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (title != null) ...[
          Text(title, style: titleStyle),
          SizedBox(height: spacing.sp2),
        ],
        Wrap(
          spacing: spacing.sp2,
          runSpacing: spacing.sp1,
          children: [
            if (status != null)
              _ProjectChip(
                label: _statusLabel(status),
                tone: _statusTone(status),
              ),
            _ProjectChip(
              label: _craftTypeLabel(project.common.craftType),
              tone: _ProjectChipTone.outlined,
            ),
          ],
        ),
        if (pattern != null || size != null) ...[
          SizedBox(height: spacing.sp3),
          const CraftskyDivider(),
          SizedBox(height: spacing.sp2),
          if (pattern != null)
            _ProjectPatternMetadataRow(pattern: project.common.pattern!),
          if (size case final row?)
            _ProjectMetadataRow(label: row.label, value: row.value),
        ],
      ],
    );
  }
}

class _ProjectDetail extends StatelessWidget {
  const _ProjectDetail({required this.project, required this.launchUrl});

  final Project project;
  final ProjectLinkLauncher launchUrl;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final title = _nonBlank(project.common.title);
    final rows = _detailRows(project);
    final chipSections = _chipSections(project.common);
    final hasPattern = _patternValue(project.common.pattern) != null;

    return Column(
      key: const ValueKey('project-detail-card'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (title != null) ...[
          Text(title, style: theme.textTheme.headlineSmall),
          SizedBox(height: spacing.sp2),
        ],
        Wrap(
          spacing: spacing.sp2,
          runSpacing: spacing.sp1,
          children: [
            if (_nonBlank(project.common.status) case final status?)
              _ProjectChip(
                label: _statusLabel(status),
                tone: _statusTone(status),
              ),
            _ProjectChip(
              label: _craftTypeLabel(project.common.craftType),
              tone: _ProjectChipTone.outlined,
            ),
          ],
        ),
        if (hasPattern || rows.isNotEmpty) ...[
          SizedBox(height: spacing.sp3),
          const CraftskyDivider(),
          SizedBox(height: spacing.sp2),
          if (hasPattern)
            _ProjectPatternMetadataRow(pattern: project.common.pattern!),
          for (final row in rows)
            _ProjectMetadataRow(
              label: row.label,
              value: row.value,
              linkUri: row.linkUri,
              launchUrl: launchUrl,
            ),
        ],
        if (chipSections.isNotEmpty) ...[
          SizedBox(height: spacing.sp3),
          const CraftskyDivider(),
          SizedBox(height: spacing.sp2),
          for (final section in chipSections) ...[
            _ProjectChipSection(section: section),
            SizedBox(height: spacing.sp2),
          ],
        ],
      ],
    );
  }
}

class _ProjectChipSection extends StatelessWidget {
  const _ProjectChipSection({required this.section});

  final _ProjectChipSectionData section;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          section.label.toUpperCase(),
          style: theme.textTheme.labelSmall?.copyWith(
            color: theme.colorScheme.outline,
          ),
        ),
        SizedBox(height: spacing.sp1),
        Wrap(
          spacing: spacing.sp1,
          runSpacing: spacing.sp1,
          children: [
            for (final value in section.values)
              _ProjectChip(label: value, tone: _ProjectChipTone.outlined),
          ],
        ),
      ],
    );
  }
}

enum _ProjectChipTone { finished, wip, outlined }

class _ProjectChip extends StatelessWidget {
  const _ProjectChip({required this.label, required this.tone});

  final String label;
  final _ProjectChipTone tone;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final colors = theme.colorScheme;
    final (background, foreground, borderColor) = switch (tone) {
      _ProjectChipTone.finished => (
        swatches.done,
        colors.onSurface,
        colors.onSurface,
      ),
      _ProjectChipTone.wip => (
        swatches.wip,
        colors.onSurface,
        colors.onSurface,
      ),
      _ProjectChipTone.outlined => (
        Colors.transparent,
        colors.onSurface,
        colors.onSurface,
      ),
    };

    return DecoratedBox(
      decoration: BoxDecoration(
        color: background,
        border: Border.all(color: borderColor, width: 1.2),
        borderRadius: BorderRadius.circular(radii.rPill),
      ),
      child: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: spacing.sp2,
          vertical: spacing.sp1,
        ),
        child: Text(
          label,
          style: theme.textTheme.labelSmall?.copyWith(
            color: foreground,
            letterSpacing: 0,
          ),
        ),
      ),
    );
  }
}

class _ProjectMetadataRow extends StatelessWidget {
  const _ProjectMetadataRow({
    required this.label,
    required this.value,
    this.linkUri,
    this.launchUrl,
  });

  final String label;
  final String value;
  final Uri? linkUri;
  final ProjectLinkLauncher? launchUrl;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Padding(
      padding: EdgeInsets.only(bottom: spacing.sp1),
      child: Row(
        children: [
          SizedBox(
            width: _metadataLabelWidth(context),
            child: Text(
              label.toUpperCase(),
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
            ),
          ),
          Expanded(
            child: linkUri == null
                ? Text(
                    value,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurface,
                      fontWeight: FontWeight.w600,
                    ),
                  )
                : _ProjectLinkValue(
                    label: value,
                    uri: linkUri!,
                    launchUrl: launchUrl!,
                  ),
          ),
        ],
      ),
    );
  }
}

class _ProjectLinkValue extends StatelessWidget {
  const _ProjectLinkValue({
    required this.label,
    required this.uri,
    required this.launchUrl,
  });

  final String label;
  final Uri uri;
  final ProjectLinkLauncher launchUrl;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return InkWell(
      onTap: () => _confirmAndOpenLink(context),
      child: Text(
        label,
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.primary,
          fontWeight: FontWeight.w700,
        ),
      ),
    );
  }

  Future<void> _confirmAndOpenLink(BuildContext context) async {
    final confirmed = await _showOpenLinkDialog(context, uri);
    if (!confirmed) return;
    await launchUrl(uri);
  }
}

class _ProjectPatternMetadataRow extends StatelessWidget {
  const _ProjectPatternMetadataRow({required this.pattern});

  final ProjectPattern pattern;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final valueStyle = theme.textTheme.bodySmall?.copyWith(
      color: theme.colorScheme.onSurface,
      fontWeight: FontWeight.w600,
    );
    final name = _nonBlank(pattern.name);
    final designer = _nonBlank(pattern.designer);
    final publisher = _nonBlank(pattern.publisher);
    final trailingCredits = [designer, publisher].whereType<String>().toList();

    return Padding(
      padding: EdgeInsets.only(bottom: spacing.sp1),
      child: Row(
        children: [
          SizedBox(
            width: _metadataLabelWidth(context),
            child: Text(
              'PATTERN',
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
            ),
          ),
          Expanded(
            child: Wrap(
              crossAxisAlignment: WrapCrossAlignment.center,
              children: [
                if (name != null)
                  FacetedText(
                    text: name,
                    facets: pattern.nameFacets,
                    style: valueStyle,
                  ),
                if (name != null && trailingCredits.isNotEmpty)
                  Text(' by ', style: valueStyle),
                if (designer != null)
                  FacetedText(
                    text: designer,
                    facets: pattern.designerFacets,
                    style: valueStyle,
                  ),
                if (designer != null && publisher != null)
                  Text(', ', style: valueStyle),
                if (publisher != null)
                  FacetedText(
                    text: publisher,
                    facets: pattern.publisherFacets,
                    style: valueStyle,
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _ProjectSizeMetadata {
  const _ProjectSizeMetadata({
    required this.label,
    required this.value,
    this.linkUri,
  });

  final String label;
  final String value;
  final Uri? linkUri;
}

class _ProjectChipSectionData {
  const _ProjectChipSectionData({required this.label, required this.values});

  final String label;
  final List<String> values;
}

List<_ProjectSizeMetadata> _detailRows(Project project) {
  return [
    if (_nonBlank(project.common.duration) case final value?)
      _ProjectSizeMetadata(label: 'Duration', value: value),
    if (_nonBlank(project.common.pattern?.difficulty) case final value?)
      _ProjectSizeMetadata(
        label: 'Difficulty',
        value: _optionLabel(value, ProjectOptionCatalogs.patternDifficulties),
      ),
    ..._optionalMetadata(_linkMetadata(project.common.pattern?.url)),
    ..._craftDetailRows(project.common.craftType, project.details),
  ];
}

List<_ProjectSizeMetadata> _craftDetailRows(
  String craftType,
  ProjectDetails? details,
) {
  return switch (details) {
    SewingProjectDetails(
      :final projectType,
      :final projectSubtype,
      :final sizeMade,
      :final fitNotes,
    ) =>
      [
        ..._projectTypeRows(craftType, projectType, projectSubtype),
        if (_nonBlank(sizeMade) case final value?)
          _ProjectSizeMetadata(label: 'Size made', value: value),
        if (_nonBlank(fitNotes) case final value?)
          _ProjectSizeMetadata(label: 'Fit notes', value: value),
      ],
    KnittingProjectDetails(
      :final projectType,
      :final projectSubtype,
      :final yarnWeight,
      :final needleSizeMm,
      :final gauge,
      :final finishedSize,
    ) =>
      [
        ..._projectTypeRows(craftType, projectType, projectSubtype),
        if (_nonBlank(yarnWeight) case final value?)
          _ProjectSizeMetadata(
            label: 'Yarn weight',
            value: _optionLabel(value, ProjectOptionCatalogs.yarnWeights),
          ),
        if (_nonBlank(needleSizeMm) case final value?)
          _ProjectSizeMetadata(label: 'Needle size', value: value),
        if (_gaugeValue(gauge) case final value?)
          _ProjectSizeMetadata(label: 'Gauge', value: value),
        if (_nonBlank(finishedSize) case final value?)
          _ProjectSizeMetadata(label: 'Finished size', value: value),
      ],
    CrochetProjectDetails(
      :final projectType,
      :final projectSubtype,
      :final yarnWeight,
      :final hookSizeMm,
      :final gauge,
      :final finishedSize,
    ) =>
      [
        ..._projectTypeRows(craftType, projectType, projectSubtype),
        if (_nonBlank(yarnWeight) case final value?)
          _ProjectSizeMetadata(
            label: 'Yarn weight',
            value: _optionLabel(value, ProjectOptionCatalogs.yarnWeights),
          ),
        if (_nonBlank(hookSizeMm) case final value?)
          _ProjectSizeMetadata(label: 'Hook size', value: value),
        if (_gaugeValue(gauge) case final value?)
          _ProjectSizeMetadata(label: 'Gauge', value: value),
        if (_nonBlank(finishedSize) case final value?)
          _ProjectSizeMetadata(label: 'Finished size', value: value),
      ],
    QuiltingProjectDetails(
      :final projectType,
      :final projectSubtype,
      :final size,
      :final piecingTechnique,
      :final quiltingMethod,
    ) =>
      [
        ..._projectTypeRows(craftType, projectType, projectSubtype),
        if (_nonBlank(size) case final value?)
          _ProjectSizeMetadata(label: 'Size', value: value),
        if (_nonBlank(piecingTechnique) case final value?)
          _ProjectSizeMetadata(
            label: 'Piecing technique',
            value: _optionLabel(
              value,
              ProjectOptionCatalogs.quiltingPiecingTechniques,
            ),
          ),
        if (_nonBlank(quiltingMethod) case final value?)
          _ProjectSizeMetadata(
            label: 'Quilting method',
            value: _optionLabel(value, ProjectOptionCatalogs.quiltingMethods),
          ),
      ],
    UnknownProjectDetails(:final raw) => [
      for (final entry in raw.entries)
        if (entry.value case final String value when _nonBlank(value) != null)
          _ProjectSizeMetadata(
            label: _tokenFallbackLabel(entry.key),
            value: _nonBlank(value)!,
          ),
    ],
    _ => const <_ProjectSizeMetadata>[],
  };
}

List<_ProjectSizeMetadata> _projectTypeRows(
  String craftType,
  String? projectType,
  String? projectSubtype,
) {
  final typeLabel = _nonBlank(projectType) == null
      ? null
      : _optionLabel(
          _nonBlank(projectType)!,
          ProjectOptionCatalogs.projectTypesForCraft(craftType),
        );
  final subtypeLabel = _nonBlank(projectSubtype) == null
      ? null
      : _optionLabel(
          _nonBlank(projectSubtype)!,
          ProjectOptionCatalogs.projectSubtypesForCraft(craftType),
        );
  final value = [typeLabel, subtypeLabel].whereType<String>().join(' > ');
  return [
    if (value.isNotEmpty)
      _ProjectSizeMetadata(label: 'Project type', value: value),
  ];
}

double _metadataLabelWidth(BuildContext context) {
  final scaled = MediaQuery.textScalerOf(context).scale(112);
  return scaled.clamp(112.0, 168.0);
}

List<_ProjectSizeMetadata> _optionalMetadata(_ProjectSizeMetadata? row) {
  return row == null ? const [] : [row];
}

_ProjectSizeMetadata? _linkMetadata(String? value) {
  final uri = _normalizeLinkUri(value);
  if (uri == null) return null;
  return _ProjectSizeMetadata(
    label: 'Link',
    value: _displayLink(uri),
    linkUri: uri,
  );
}

Uri? _normalizeLinkUri(String? value) {
  final trimmed = _nonBlank(value);
  if (trimmed == null) return null;
  final parsed = Uri.tryParse(trimmed);
  if (parsed == null) return null;
  final withScheme = parsed.hasScheme
      ? parsed
      : Uri.tryParse('https://$trimmed');
  if (withScheme == null) return null;
  if (withScheme.scheme != 'http' && withScheme.scheme != 'https') return null;
  if (withScheme.host.isEmpty) return null;
  return withScheme;
}

String _displayLink(Uri uri) {
  final buffer = StringBuffer(uri.host);
  var path = uri.path;
  if (path.endsWith('/') && path.length > 1) {
    path = path.substring(0, path.length - 1);
  }
  if (path.isNotEmpty && path != '/') {
    buffer.write(path);
  }
  return buffer.toString();
}

Future<bool> _defaultLaunchUrl(Uri uri) {
  return url_launcher.launchUrl(
    uri,
    mode: url_launcher.LaunchMode.externalApplication,
  );
}

Future<bool> _showOpenLinkDialog(BuildContext context, Uri uri) async {
  final theme = Theme.of(context);
  final spacing = theme.extension<SpacingTheme>()!;
  final durations = theme.extension<DurationTheme>()!;
  final result = await showGeneralDialog<bool>(
    context: context,
    barrierDismissible: true,
    barrierLabel: MaterialLocalizations.of(context).modalBarrierDismissLabel,
    barrierColor: Colors.black54,
    transitionDuration: durations.modal,
    pageBuilder: (dialogContext, _, _) => CraftskyDialog(
      title: 'Open link?',
      body: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Text('This will open outside Craftsky.'),
          SizedBox(height: spacing.sp3),
          SelectableText(
            uri.toString(),
            style: theme.textTheme.bodySmall?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(dialogContext).pop(false),
          child: const Text('Cancel'),
        ),
        ChunkyButton(
          onPressed: () => Navigator.of(dialogContext).pop(true),
          child: const Text('Open link'),
        ),
      ],
    ),
    transitionBuilder: (_, animation, _, child) {
      final curved = CurvedAnimation(
        parent: animation,
        curve: durations.easePop,
      );
      return FadeTransition(
        opacity: curved,
        child: ScaleTransition(
          scale: Tween<double>(begin: 0.96, end: 1).animate(curved),
          child: child,
        ),
      );
    },
  );
  return result ?? false;
}

List<_ProjectChipSectionData> _chipSections(ProjectCommon common) {
  return [
    if (_listLabels(common.materials) case final values when values.isNotEmpty)
      _ProjectChipSectionData(label: 'Materials', values: values),
    if (_listLabels(common.colors, ProjectOptionCatalogs.colours)
        case final values when values.isNotEmpty)
      _ProjectChipSectionData(label: 'Colours', values: values),
    if (_listLabels(common.designTags, ProjectOptionCatalogs.designTags)
        case final values when values.isNotEmpty)
      _ProjectChipSectionData(label: 'Design tags', values: values),
    if (_listLabels(common.tags) case final values when values.isNotEmpty)
      _ProjectChipSectionData(label: 'Tags', values: values),
  ];
}

List<String> _listLabels(List<String>? values, [List<ProjectOption>? options]) {
  return [
    for (final value in values ?? const <String>[])
      if (_nonBlank(value) case final clean?)
        options == null ? clean : _optionLabel(clean, options),
  ];
}

String? _gaugeValue(ProjectGauge? gauge) {
  if (gauge == null) return null;
  final rows = gauge.rows == null ? '' : ' / ${gauge.rows} rows';
  return '${gauge.stitches} sts$rows per ${gauge.measurement} ${gauge.unit}';
}

String _craftTypeLabel(String value) => _optionLabel(
  value,
  ProjectOptionCatalogs.craftTypes,
);

String _statusLabel(String value) => _optionLabel(
  value,
  ProjectOptionCatalogs.statuses,
);

_ProjectChipTone _statusTone(String value) {
  return switch (value) {
    ProjectOptionCatalogs.finishedStatusToken => _ProjectChipTone.finished,
    ProjectOptionCatalogs.wipStatusToken => _ProjectChipTone.wip,
    _ => _ProjectChipTone.outlined,
  };
}

String _optionLabel(String value, List<ProjectOption> options) {
  for (final option in options) {
    if (option.value == value) return option.label;
  }
  return _tokenFallbackLabel(value);
}

String _tokenFallbackLabel(String value) {
  final token = value.contains('#') ? value.split('#').last : value;
  if (token.isEmpty) return value;
  final words = token
      .replaceAllMapped(RegExp('([a-z0-9])([A-Z])'), (match) {
        return '${match.group(1)} ${match.group(2)}';
      })
      .replaceAll(RegExp('[-_]+'), ' ')
      .trim()
      .split(RegExp(r'\s+'));
  return words
      .where((word) => word.isNotEmpty)
      .map((word) => '${word[0].toUpperCase()}${word.substring(1)}')
      .join(' ');
}

String? _patternValue(ProjectPattern? pattern) {
  if (pattern == null) return null;
  final name = _nonBlank(pattern.name);
  final designer = _nonBlank(pattern.designer);
  final publisher = _nonBlank(pattern.publisher);
  return switch ((name, designer, publisher)) {
    (final String name, final String designer, final String publisher) =>
      '$name by $designer, $publisher',
    (final String name, final String designer, null) => '$name by $designer',
    (final String name, null, final String publisher) => '$name by $publisher',
    (final String name, null, null) => name,
    (null, final String designer, final String publisher) =>
      '$designer, $publisher',
    (null, final String designer, null) => designer,
    (null, null, final String publisher) => publisher,
    _ => null,
  };
}

_ProjectSizeMetadata? _sizeMetadata(ProjectDetails? details) {
  return switch (details) {
    SewingProjectDetails(:final sizeMade) when _nonBlank(sizeMade) != null =>
      _ProjectSizeMetadata(label: 'Size', value: _nonBlank(sizeMade)!),
    KnittingProjectDetails(:final finishedSize)
        when _nonBlank(finishedSize) != null =>
      _ProjectSizeMetadata(
        label: 'Finished size',
        value: _nonBlank(finishedSize)!,
      ),
    CrochetProjectDetails(:final finishedSize)
        when _nonBlank(finishedSize) != null =>
      _ProjectSizeMetadata(
        label: 'Finished size',
        value: _nonBlank(finishedSize)!,
      ),
    QuiltingProjectDetails(:final size) when _nonBlank(size) != null =>
      _ProjectSizeMetadata(label: 'Size', value: _nonBlank(size)!),
    _ => null,
  };
}

String? _nonBlank(String? value) {
  final trimmed = value?.trim();
  return trimmed == null || trimmed.isEmpty ? null : trimmed;
}
