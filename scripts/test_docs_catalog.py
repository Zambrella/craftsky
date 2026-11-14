"""Regression checks for truthful catalogue validation and bounded discovery."""
import contextlib
import io
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import docs_catalog as docs


class CatalogueTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        (self.root / 'docs/changes/catalog').mkdir(parents=True)
        (self.root / 'docs/current.md').write_text('# Current guide\n\nMaintained guidance.\n')
        (self.root / 'app').mkdir()
        self.catalogue = docs.Catalogue(self.root)

    def card(self, identifier='2026-01-01-session-refresh', **overrides):
        source = self.root / 'docs/changes' / identifier
        source.mkdir(exist_ok=True)
        document = source / '01-record.md'
        if not document.exists():
            document.write_text('# Original plan\n\nProposed behavior.\n\n## Evidence\n\nLocal tests passed.\n')
        data = {'id': identifier, 'title': 'Session refresh races', 'source': f'docs/changes/{identifier}',
                'implementation': 'implemented', 'archive': 'archived', 'components': ['appview'],
                'topics': ['authentication', 'recovery'], 'keywords': ['OAuth', 'refresh token', 'race'],
                'summary': 'Preserves session integrity during concurrent token refresh.',
                'currentDocs': ['docs/current.md#current-guide'], 'code': ['app'],
                'evidence': [f'docs/changes/{identifier}/01-record.md#evidence'],
                'supersededBy': [], 'followUps': []}
        data.update(overrides)
        self.write_card(data)
        return source, data

    def write_card(self, data, body=None, filename=None):
        path = self.root / 'docs/changes/catalog' / (filename or (data['id'] + '.md'))
        body = body or '# Outcome\n\nThe implemented change keeps account refresh synchronized. Local test evidence does not prove production deployment.\n'
        path.write_text('---\n' + json.dumps(data, indent=2) + '\n---\n' + body)
        return path

    def invoke(self, *args):
        output, error = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(output), contextlib.redirect_stderr(error):
            status = docs.main(['--root', str(self.root), *args])
        return status, output.getvalue(), error.getvalue()

    def test_generated_indexes_are_deterministic_and_detect_card_and_inventory_drift(self):
        _, data = self.card()
        self.assertEqual(self.invoke('update')[0], 0)
        original = (self.root / 'docs/changes/catalog.jsonl').read_bytes()
        self.assertEqual(self.invoke('check')[0], 0)
        self.assertEqual(self.invoke('update')[0], 0)
        self.assertEqual((self.root / 'docs/changes/catalog.jsonl').read_bytes(), original)
        self.write_card(data, '# Outcome\n\nUpdated recovery evidence.\n')
        status, _, error = self.invoke('check')
        self.assertEqual(status, 1)
        self.assertIn('stale generated indexes', error)
        self.assertEqual(self.invoke('update')[0], 0)
        (self.root / 'docs/changes/2026-02-01-new-change').mkdir()
        status, _, error = self.invoke('check')
        self.assertEqual(status, 1)
        self.assertIn('missing=', error)

    def test_index_has_one_primary_topic_row_and_exposes_archived_unfinished_work(self):
        self.card(implementation='partial', followUps=['Verify device behavior.'])
        self.assertEqual(self.invoke('update')[0], 0)
        index = (self.root / 'docs/changes/INDEX.md').read_text()
        self.assertEqual(index.count('| [Session refresh races]'), 1)
        self.assertIn('## authentication', index)
        self.assertIn('| recovery |', index)
        self.assertIn('(partial, archived): Verify device behavior.', index)

    def test_original_edits_and_new_documents_invalidate_the_recorded_source_snapshot(self):
        source, _ = self.card()
        self.assertEqual(self.invoke('update')[0], 0)
        original = json.loads((self.root / 'docs/changes/catalog.jsonl').read_text())['sourceSha256']
        document = source / '01-record.md'
        document.write_text(document.read_text() + '\nLate amendment changes the old requirement.\n')
        status, _, error = self.invoke('check')
        self.assertEqual(status, 1)
        self.assertIn('review affected summaries', error)
        self.assertEqual(self.invoke('update')[0], 0)
        changed = json.loads((self.root / 'docs/changes/catalog.jsonl').read_text())['sourceSha256']
        self.assertNotEqual(original, changed)
        (source / '02-follow-up.md').write_text('# New follow-up\n\nPending device proof.\n')
        self.assertEqual(self.invoke('check')[0], 1)

    def test_source_snapshot_survives_checkout_line_ending_conversion(self):
        source, _ = self.card()
        self.assertEqual(self.invoke('update')[0], 0)
        document = source / '01-record.md'
        document.write_bytes(document.read_bytes().replace(b'\n', b'\r\n'))
        self.assertEqual(self.invoke('check')[0], 0)

    def test_search_ranks_metadata_and_filters_without_reading_original_documents(self):
        self.card()
        _, data = self.card('2026-02-01-feed', title='Feed repair', components=['web-client'], topics=['feeds'],
                            keywords=['pagination'], summary='Repairs feed ordering.')
        self.write_card(data, '# Outcome\n\nHistorical OAuth refresh race discussion.\n')
        with mock.patch.object(docs.Catalogue, 'document', side_effect=AssertionError('source text must not be read')):
            status, output, _ = self.invoke('search', 'OAuth refresh race', '--json')
        self.assertEqual(status, 0)
        result = json.loads(output)
        self.assertEqual(result['total'], 2)
        self.assertEqual(result['results'][0]['id'], '2026-01-01-session-refresh')
        self.assertNotIn('evidence', result['results'][0])
        self.assertNotIn('text', result['results'][0])
        status, output, _ = self.invoke('search', 'refresh', '--component', 'web-client', '--json')
        self.assertEqual(status, 0)
        self.assertEqual(json.loads(output)['results'][0]['id'], '2026-02-01-feed')
        self.assertEqual(self.invoke('search', 'missing terminology')[1].strip(), 'No matching changes. Try fewer keywords or different terminology.')
        self.assertEqual(self.invoke('search', 'refresh', '--implementation', 'partial', '--json')[0], 0)
        self.assertEqual(json.loads(self.invoke('search', 'refresh', '--archive', 'active', '--json')[1])['total'], 0)

    def test_search_matches_symbol_words_without_incidental_substrings(self):
        _, image = self.card('2026-01-01-image-editing', title='Basic image editing',
                             keywords=['WithDiagnosticOccurrence', 'account_session_lease'])
        self.card('2026-02-01-reposts', title='Repost credits', keywords=['image crediting'])
        result = json.loads(self.invoke('search', 'image editing', '--json')[1])
        self.assertEqual([row['id'] for row in result['results']], [image['id']])
        for query in ('WithDiagnosticOccurrence', 'diagnostic occurrence', 'account_session_lease', 'account lease'):
            result = json.loads(self.invoke('search', query, '--json')[1])
            self.assertEqual(result['results'][0]['id'], image['id'])

    def test_superseded_search_matches_rank_below_equally_relevant_outcomes(self):
        self.card('2026-01-01-old-session', implementation='superseded')
        _, latest = self.card('2026-02-01-current-session')
        result = json.loads(self.invoke('search', 'refresh', '--json')[1])
        self.assertEqual(result['results'][0]['id'], latest['id'])

    def test_heading_maps_exclude_fences_and_support_setext_duplicates_and_parent_context(self):
        source, _ = self.card()
        document = source / '01-record.md'
        document.write_text('# Plan\n\n```md\n## Ignored\n```\n\nEvidence\n--------\n\n## Evidence\n\n### Last review\nDetails.\n')
        status, output, _ = self.invoke('sections', source.name, '--document', document.name, '--json')
        self.assertEqual(status, 0)
        rows = json.loads(output)['sections']
        self.assertEqual([item['section'] for item in rows], ['plan', 'evidence', 'evidence-1', 'last-review'])
        self.assertEqual(rows[-1]['parents'], ['Plan', 'Evidence'])
        status, output, _ = self.invoke('sections', source.name, '--query', 'last review', '--json')
        self.assertEqual(status, 0)
        self.assertEqual(json.loads(output)['total'], 1)
        result = json.loads(self.invoke('sections', source.name, '--limit', '2', '--json')[1])
        self.assertEqual(result['nextOffset'], 2)
        self.assertEqual(json.loads(self.invoke('sections', source.name, '--offset', '2', '--json')[1])['sections'][0]['section'], 'evidence-1')

    def test_section_windows_are_exact_even_when_a_single_line_exceeds_the_limit(self):
        source, _ = self.card()
        text = '# Enormous\n' + ('aé🙂' * 15000) + '\n\n## Tail\nEnd.\n'
        (source / '01-record.md').write_text(text)
        expected = text
        parts, offset = [], 0
        while True:
            status, output, _ = self.invoke('read', source.name, '--document', '01-record.md', '--section', 'enormous', '--offset', str(offset), '--max-chars', '12000', '--json')
            self.assertEqual(status, 0)
            value = json.loads(output)
            self.assertLessEqual(len(value['text']), 12000)
            self.assertLessEqual(value['startLine'], value['endLine'])
            parts.append(value['text'])
            offset = value['nextOffset']
            if offset is None:
                break
        self.assertEqual(''.join(parts), expected)
        self.assertEqual(self.invoke('read', source.name, '--document', '01-record.md', '--section', 'absent')[0], 1)
        self.assertEqual(self.invoke('read', source.name, '--document', '01-record.md', '--section', 'enormous', '--offset', '9999999')[0], 1)

    def test_standalone_and_headingless_documents_share_the_catalogue(self):
        source, data = self.card('2026-01-01-note')
        (source / '01-record.md').unlink()
        source.rmdir()
        loose = self.root / 'docs/changes/2026-01-01-note.md'
        loose.write_text('Standalone historical note.\n')
        data.update(source='docs/changes/2026-01-01-note.md', evidence=['docs/changes/2026-01-01-note.md'])
        self.write_card(data)
        self.assertEqual(self.invoke('update')[0], 0)
        rows = json.loads(self.invoke('sections', data['id'], '--json')[1])['sections']
        self.assertEqual(rows[0]['section'], 'document')
        result = json.loads(self.invoke('read', data['id'], '--document', loose.name, '--section', 'document', '--json')[1])
        self.assertEqual(result['text'], loose.read_text())

    def test_missing_files_anchors_duplicate_ids_and_invalid_field_types_fail_closed(self):
        _, data = self.card()
        variants = [dict(data, evidence=['docs/current.md#missing']), dict(data, code=['app/missing.dart']),
                    dict(data, components='appview'), dict(data, keywords=['repeat', 'repeat']),
                    dict(data, followUps=[{'bad': 'nested'}]), dict(data, implementation=['implemented'])]
        for variant in variants:
            with self.subTest(variant=variant):
                self.write_card(variant)
                status, _, error = self.invoke('update')
                self.assertEqual(status, 1)
                self.assertNotIn('Traceback', error)
        self.write_card(data)
        self.write_card(data, filename='duplicate.md')
        self.assertEqual(self.invoke('check')[0], 1)

    def test_duplicate_metadata_fields_and_oversized_queries_fail_without_echoing_input(self):
        source, data = self.card()
        path = self.write_card(data)
        raw = path.read_text().replace('"title": "Session refresh races",', '"title": "Earlier title", "title": "Session refresh races",')
        path.write_text(raw)
        status, _, error = self.invoke('show', source.name)
        self.assertEqual(status, 1)
        self.assertIn('duplicate metadata field: title', error)
        self.write_card(data)
        huge = 'UNBOUNDED_QUERY_CANARY ' * 1000
        for arguments in [('search', huge, '--json'), ('sections', source.name, '--query', huge, '--json')]:
            status, output, error = self.invoke(*arguments)
            self.assertEqual(status, 1)
            self.assertLess(len(output + error), 300)
            self.assertNotIn('UNBOUNDED_QUERY_CANARY', output + error)

    def test_supersession_must_resolve_and_be_acyclic_including_self_links(self):
        _, first = self.card()
        _, second = self.card('2026-02-01-successor')
        first['supersededBy'] = ['absent']
        self.write_card(first)
        self.assertIn('unknown superseding change', self.invoke('update')[2])
        first['supersededBy'] = [first['id']]
        self.write_card(first)
        self.assertIn('cyclic supersession', self.invoke('update')[2])
        first['supersededBy'] = [second['id']]
        second['supersededBy'] = [first['id']]
        self.write_card(first)
        self.write_card(second)
        self.assertIn('cyclic supersession', self.invoke('update')[2])

    def test_source_identity_is_checked_before_search_show_or_read(self):
        source, data = self.card()
        other, _ = self.card('2026-02-01-other')
        data['source'] = f'docs/changes/{other.name}'
        self.write_card(data)
        for arguments in [('show', source.name), ('search', 'refresh'),
                          ('read', source.name, '--document', '01-record.md', '--section', 'original-plan')]:
            status, _, error = self.invoke(*arguments)
            self.assertEqual(status, 1)
            self.assertIn('ID does not match', error)

    def test_huge_heading_selectors_do_not_collide_with_literal_line_headings(self):
        source, _ = self.card()
        (source / '01-record.md').write_text('# ' + 'x' * 400 + '\n\n## line-1\nContents.\n')
        rows = json.loads(self.invoke('sections', source.name, '--json')[1])['sections']
        self.assertEqual([row['section'] for row in rows], ['@line-1', 'line-1'])
        self.assertLessEqual(len(rows[0]['heading']), 200)
        status, output, _ = self.invoke('read', source.name, '--document', '01-record.md', '--section', '@line-1', '--json')
        self.assertEqual(status, 0)
        self.assertIn('Contents.', json.loads(output)['text'])

    def test_generated_output_symlinks_cannot_overwrite_other_files(self):
        self.card()
        sentinel = self.root / 'preserve.txt'
        sentinel.write_text('SENTINEL')
        (self.root / 'docs/changes/INDEX.md').symlink_to(sentinel)
        status, _, error = self.invoke('update')
        self.assertEqual(status, 1)
        self.assertIn('generated output', error)
        self.assertEqual(sentinel.read_text(), 'SENTINEL')

    def test_references_and_source_reads_reject_traversal_absolute_paths_and_symlink_escape(self):
        source, data = self.card()
        outside = Path(self.temporary.name).parent / (self.root.name + '-outside.md')
        outside.write_text('# Private\nPRIVATE_FILE_CANARY\n')
        self.addCleanup(lambda: outside.unlink(missing_ok=True))
        (source / 'escape.md').symlink_to(outside)
        for document in ['../current.md', '/etc/passwd', 'escape.md']:
            status, output, error = self.invoke('read', source.name, '--document', document, '--section', 'private')
            self.assertEqual(status, 1)
            self.assertNotIn('PRIVATE_FILE_CANARY', output + error)
        data['currentDocs'] = ['../' + outside.name]
        self.write_card(data)
        self.assertEqual(self.invoke('update')[0], 1)
        data['currentDocs'] = ['docs/current.md']
        self.write_card(data)
        (self.root / 'docs/current.md').unlink()
        (self.root / 'docs/current.md').symlink_to(outside)
        self.assertIn('escapes the repository', self.invoke('update')[2])

    def test_broken_summary_links_and_oversized_cards_fail_validation(self):
        _, data = self.card()
        self.write_card(data, '# Outcome\n\n[Missing](../missing.md)\n')
        self.assertIn('missing reference', self.invoke('update')[2])
        self.write_card(data, '# Outcome\n\n' + 'word ' * 301)
        self.assertIn('1–300 words', self.invoke('update')[2])
        self.write_card(data, '# Outcome\n\n' + 'x' * 12001)
        self.assertIn('12000 characters', self.invoke('update')[2])

    def test_cli_runs_from_any_directory_and_rejects_unbounded_output_arguments(self):
        source, _ = self.card()
        script = Path(__file__).with_name('docs')
        result = subprocess.run([sys.executable, str(script), '--root', str(self.root), 'show', source.name, '--json'],
                                cwd=self.temporary.name, text=True, capture_output=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(json.loads(result.stdout)['id'], source.name)
        for arguments in [('search', 'refresh', '--limit', '999'),
                          ('sections', source.name, '--limit', '999'),
                          ('read', source.name, '--document', '01-record.md', '--section', 'original-plan', '--max-chars', '999999')]:
            with contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit) as caught:
                docs.main(['--root', str(self.root), *arguments])
            self.assertEqual(caught.exception.code, 2)


if __name__ == '__main__':
    unittest.main()
