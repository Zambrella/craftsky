import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_detail_fields.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/rich_text/widgets/faceted_text.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

typedef ProjectLinkLauncher = ExternalLinkLauncher;

enum ProjectCardVariant { summary, detail }

class ProjectCard extends StatelessWidget {
  const new({
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
        launchUrl: launchUrl ?? launchExternalLink,
      ),
    };
  }
}

class _ProjectSummary extends StatelessWidget {
  const new({required this.project});

  final Project project;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final title = _nonBlank(project.common.title);
    final status = _nonBlank(project.common.status);
    final patterns = project.common.effectivePatterns.where(
      (pattern) =>
          _patternValue(pattern, l10n.projectPatternSelfDraftedLabel) != null,
    );
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
              craft: project.common.craftType,
              tone: _ProjectChipTone.outlined,
            ),
            for (final craft
                in project.common.additionalCrafts ?? const <ProjectCraft>[])
              _ProjectChip(
                label: _craftTypeLabel(craft.craftType),
                craft: craft.craftType,
                tone: _ProjectChipTone.outlined,
              ),
          ],
        ),
        if (patterns.isNotEmpty) ...[
          SizedBox(height: spacing.sp3),
          const CraftskyDivider(),
          SizedBox(height: spacing.sp2),
          for (final pattern in patterns)
            _ProjectPatternMetadataRow(pattern: pattern),
        ],
      ],
    );
  }
}

class _ProjectDetail extends StatelessWidget {
  const new({required this.project, required this.launchUrl});

  final Project project;
  final ProjectLinkLauncher launchUrl;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final title = _nonBlank(project.common.title);
    final rows = _detailRows(project);
    final tags = [
      for (final tag in project.common.tags ?? const <String>[])
        ?_nonBlank(tag),
    ];
    final patterns = project.common.effectivePatterns.where(
      (pattern) =>
          _patternValue(pattern, l10n.projectPatternSelfDraftedLabel) != null ||
          _patternDetailRows(pattern).isNotEmpty,
    );

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
              craft: project.common.craftType,
              tone: _ProjectChipTone.outlined,
            ),
            for (final craft
                in project.common.additionalCrafts ?? const <ProjectCraft>[])
              _ProjectChip(
                label: _craftTypeLabel(craft.craftType),
                craft: craft.craftType,
                tone: _ProjectChipTone.outlined,
              ),
          ],
        ),
        if (patterns.isNotEmpty || rows.isNotEmpty) ...[
          SizedBox(height: spacing.sp3),
          const CraftskyDivider(),
          SizedBox(height: spacing.sp2),
          for (final pattern in patterns) ...[
            if (_patternValue(pattern, l10n.projectPatternSelfDraftedLabel) !=
                null)
              _ProjectPatternMetadataRow(pattern: pattern),
            for (final row in _patternDetailRows(pattern))
              _ProjectMetadataRow(
                label: row.label,
                value: row.value,
                linkUri: row.linkUri,
                launchUrl: launchUrl,
              ),
            if (patterns.length > 1) SizedBox(height: spacing.sp2),
          ],
          for (final row in rows)
            _ProjectMetadataRow(
              label: row.label,
              value: row.value,
              linkUri: row.linkUri,
              launchUrl: launchUrl,
            ),
        ],
        if (tags.isNotEmpty) ...[
          SizedBox(height: spacing.sp3),
          const CraftskyDivider(),
          SizedBox(height: spacing.sp2),
          _ProjectTags(tags: tags),
          SizedBox(height: spacing.sp2),
        ],
      ],
    );
  }
}

class _ProjectTags extends StatelessWidget {
  const new({required this.tags});

  final List<String> tags;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'TAGS',
          style: theme.textTheme.labelSmall?.copyWith(
            color: theme.colorScheme.outline,
          ),
        ),
        SizedBox(height: spacing.sp1),
        Wrap(
          spacing: spacing.sp1,
          runSpacing: spacing.sp1,
          children: [
            for (final tag in tags)
              _ProjectChip(label: tag, tone: _ProjectChipTone.outlined),
          ],
        ),
      ],
    );
  }
}

enum _ProjectChipTone { finished, wip, outlined }

class _ProjectChip extends StatelessWidget {
  const new({required this.label, required this.tone, this.craft});

  final String label;
  final _ProjectChipTone tone;
  final String? craft;

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
        child: craft == null
            ? Text(
                label,
                style: theme.textTheme.labelSmall?.copyWith(
                  color: foreground,
                  letterSpacing: 0,
                ),
              )
            : CraftIconLabel(
                craft: craft!,
                label: label,
                iconSize: 15,
                gap: spacing.sp1,
                flexibleLabel: true,
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
  const new({
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
  const new({
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
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  Future<void> _confirmAndOpenLink(BuildContext context) async {
    await confirmAndLaunchExternalLink(
      context,
      uri: uri,
      launchUrl: launchUrl,
    );
  }
}

class _ProjectPatternMetadataRow extends StatelessWidget {
  const new({required this.pattern});

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
    final selfDrafted = pattern.selfDrafted == true;
    final selfDraftedLabel = AppLocalizations.of(
      context,
    ).projectPatternSelfDraftedLabel;

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
                if (selfDrafted && (name != null || trailingCredits.isNotEmpty))
                  Text(' · ', style: valueStyle),
                if (selfDrafted) Text(selfDraftedLabel, style: valueStyle),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _ProjectDetailRow {
  const new({
    required this.label,
    required this.value,
    this.linkUri,
  });

  final String label;
  final String value;
  final Uri? linkUri;
}

List<_ProjectDetailRow> _detailRows(Project project) {
  final details = project.details;
  if (details == null) return const [];
  final craft = project.common.craftType;
  final values =
      const ProjectDetailsMapper().encode(details) as Map<String, dynamic>;
  final type = values[r'$type'];
  if (type != null && type != ProjectDetailFields.typeForCraft(craft)) {
    return const [];
  }

  final item = [
    if (ProjectOptionCatalogs.projectTypesForCraft(craft).isNotEmpty)
      for (final key in const ['projectType', 'projectSubtype'])
        if (values[key] case final String value when value.trim().isNotEmpty)
          ProjectOptionCatalogs.projectValueLabel(value.trim()),
  ].join(' > ');

  return [
    if (item.isNotEmpty)
      _ProjectDetailRow(
        label: craft == ProjectOptionCatalogs.dyeingCraftToken
            ? 'Dyed item'
            : 'Project type',
        value: item,
      ),
    for (final field in ProjectDetailFields.authoringForCraft(craft))
      if (_detailFieldValue(field, craft, values) case final value?)
        _ProjectDetailRow(label: field.label, value: value),
  ];
}

List<_ProjectDetailRow> _patternDetailRows(ProjectPattern pattern) {
  return [
    if (_nonBlank(pattern.difficulty) case final value?)
      _ProjectDetailRow(
        label: 'Difficulty',
        value: _optionLabel(value, ProjectOptionCatalogs.patternDifficulties),
      ),
    ..._optionalMetadata(_linkMetadata(pattern.url)),
  ];
}

String? _detailFieldValue(
  ProjectDetailField field,
  String craft,
  Map<String, dynamic> values,
) {
  final options = field.choicesFor(craft);
  final value = ProjectDetailFields.authoringValue(craft, field.key, values);
  return switch (value) {
    final String text when text.trim().isNotEmpty => _optionLabel(
      text.trim(),
      options,
    ),
    final List<Object?> entries => _nonBlank(
      [
        for (final entry in entries)
          if (entry case final String text when text.trim().isNotEmpty)
            _optionLabel(text.trim(), options),
      ].join(', '),
    ),
    _ => null,
  };
}

double _metadataLabelWidth(BuildContext context) {
  final scaled = MediaQuery.textScalerOf(context).scale(112);
  return scaled.clamp(112.0, 168.0);
}

List<_ProjectDetailRow> _optionalMetadata(_ProjectDetailRow? row) {
  return row == null ? const [] : [row];
}

_ProjectDetailRow? _linkMetadata(String? value) {
  final uri = normalizeExternalLinkUri(value);
  if (uri == null) return null;
  return _ProjectDetailRow(
    label: 'Link',
    value: displayExternalLink(uri),
    linkUri: uri,
  );
}

String _craftTypeLabel(String value) => _optionLabel(
  value,
  ProjectOptionCatalogs.allCraftTypes,
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

String? _patternValue(ProjectPattern? pattern, String selfDraftedLabel) {
  if (pattern == null) return null;
  final name = _nonBlank(pattern.name);
  final designer = _nonBlank(pattern.designer);
  final publisher = _nonBlank(pattern.publisher);
  final value = switch ((name, designer, publisher)) {
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
  if (pattern.selfDrafted != true) return value;
  return value == null ? selfDraftedLabel : '$value · $selfDraftedLabel';
}

String? _nonBlank(String? value) {
  final trimmed = value?.trim();
  return trimmed == null || trimmed.isEmpty ? null : trimmed;
}
