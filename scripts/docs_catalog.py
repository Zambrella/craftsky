"""Dependency-free change catalogue and bounded historical-document retrieval."""
import argparse
from collections import defaultdict
from dataclasses import dataclass
import hashlib
import html
import json
from pathlib import Path, PurePosixPath
import re
import shlex
import sys


IMPLEMENTATIONS = {'proposed', 'partial', 'implemented', 'superseded', 'abandoned', 'unknown'}
ARCHIVES = {'active', 'archived'}
FIELDS = {'id', 'title', 'source', 'implementation', 'archive', 'components', 'topics',
          'keywords', 'summary', 'currentDocs', 'code', 'evidence', 'supersededBy', 'followUps'}
SLUG = re.compile(r'[a-z0-9]+(?:-[a-z0-9]+)*\Z')
EXCLUDED = {'README.md', 'INDEX.md', 'catalog'}
STOP_WORDS = {'a', 'an', 'and', 'are', 'do', 'does', 'for', 'how', 'in', 'is', 'it',
              'of', 'on', 'the', 'to', 'what', 'when', 'where', 'why', 'with'}


class CatalogueError(ValueError):
    pass


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise CatalogueError(f'duplicate metadata field: {key}')
        result[key] = value
    return result


def tokens(text):
    return set(re.findall(r'\w+', text.casefold())) - STOP_WORDS


def field_tokens(text):
    # Keep exact symbols searchable, and also expose their constituent words.
    separated = re.sub(r'([a-z0-9])([A-Z])', r'\1 \2', text).replace('_', ' ')
    return tokens(text) | tokens(separated)


def anchor(text):
    text = re.sub(r'!?\[([^\]]*)\]\([^)]*\)', r'\1', text)
    text = re.sub(r'<[^>]*>', '', text)
    text = html.unescape(text).strip().lower()
    return re.sub(r'[^\w\- ]', '', text).replace(' ', '-')


@dataclass
class Heading:
    title: str
    slug: str
    level: int
    line: int
    start: int
    parents: list
    end_line: int = 0
    end: int = 0

    @property
    def selector(self):
        # '@' never occurs in a generated Markdown anchor, so the fallback
        # cannot collide with a literal heading such as 'line-1'.
        return self.slug if self.slug and len(self.slug) <= 200 else f'@line-{self.line}'


def headings(text):
    """ATX/setext headings, outside fenced code, with duplicate-anchor suffixes."""
    lines = text.splitlines(keepends=True)
    offsets = [0]
    for line in lines:
        offsets.append(offsets[-1] + len(line))
    found, used, stack = [], set(), []
    fence, fence_length = None, 0
    previous_plain = False
    for i, line in enumerate(lines):
        stripped = line.rstrip('\r\n')
        marker = re.match(r'^ {0,3}(`{3,}|~{3,})(.*)$', stripped)
        if fence:
            if marker and marker[1][0] == fence and len(marker[1]) >= fence_length and not marker[2].strip():
                fence = None
            previous_plain = False
            continue
        if marker:
            fence, fence_length = marker[1][0], len(marker[1])
            previous_plain = False
            continue
        atx = re.match(r'^ {0,3}(#{1,6})(?:[ \t]+(.*?)|)[ \t]*$', stripped)
        setext = re.match(r'^ {0,3}(=+|-+)[ \t]*$', stripped)
        if atx:
            title = re.sub(r'[ \t]+#+[ \t]*$', '', atx[2] or '').strip()
            level, line_index = len(atx[1]), i
        elif setext and previous_plain:
            title, level, line_index = lines[i - 1].strip(), 1 if setext[1][0] == '=' else 2, i - 1
        else:
            previous_plain = bool(stripped.strip()) and not stripped.startswith(('    ', '\t'))
            continue
        base = anchor(title)
        slug, suffix = base, 0
        while slug in used:
            suffix += 1
            slug = f'{base}-{suffix}'
        used.add(slug)
        while stack and stack[-1].level >= level:
            stack.pop()
        item = Heading(title, slug, level, line_index + 1, offsets[line_index], [p.title for p in stack])
        found.append(item)
        stack.append(item)
        previous_plain = False
    for i, item in enumerate(found):
        following = next((candidate for candidate in found[i + 1:] if candidate.level <= item.level), None)
        item.end = following.start if following else len(text)
        item.end_line = following.line - 1 if following else len(lines)
    return found


@dataclass
class Card:
    metadata: dict
    body: str
    path: str
    digest: str

    def record(self):
        return dict(self.metadata, card=self.path, cardSha256=self.digest)


class Catalogue:
    def __init__(self, root):
        self.root = Path(root).resolve()
        self.changes = self.root / 'docs/changes'
        self.card_dir = self.changes / 'catalog'
        self._documents = {}

    def resolve(self, value, *, directory=True):
        if not isinstance(value, str) or not value or '\\' in value:
            raise CatalogueError(f'invalid repository path: {value!r}')
        path = PurePosixPath(value)
        if path.is_absolute() or '..' in path.parts or '?' in value:
            raise CatalogueError(f'path must stay within the repository: {value!r}')
        candidate = self.root / path
        resolved = candidate.resolve()
        if not resolved.is_relative_to(self.root):
            raise CatalogueError(f'path escapes the repository: {value!r}')
        if not candidate.exists() or (not directory and not candidate.is_file()):
            raise CatalogueError(f'missing reference: {value}')
        return candidate

    def document(self, path):
        if path not in self._documents:
            text = path.read_text(encoding='utf-8')
            self._documents[path] = text, headings(text)
        return self._documents[path]

    def reference(self, value, *, markdown=False):
        path, separator, fragment = value.partition('#')
        target = self.resolve(path, directory=not markdown)
        if markdown and target.suffix.lower() != '.md':
            raise CatalogueError(f'expected a Markdown document: {value}')
        if separator:
            if not fragment or not target.is_file() or target.suffix.lower() != '.md':
                raise CatalogueError(f'invalid heading reference: {value}')
            if fragment not in {item.slug for item in self.document(target)[1]}:
                raise CatalogueError(f'missing heading: {value}')
        return target

    def inventory(self):
        sources = {}
        for path in sorted(self.changes.iterdir()):
            if path.name in EXCLUDED or not (path.is_dir() or path.suffix == '.md'):
                continue
            if path.is_symlink():
                raise CatalogueError(f'change source must not be a symlink: {path.name}')
            identifier = path.name if path.is_dir() else path.stem
            if identifier in sources:
                raise CatalogueError(f'duplicate source ID: {identifier}')
            sources[identifier] = path.relative_to(self.root).as_posix()
        return sources

    def load(self, *, validate=False):
        cards = {}
        if not self.card_dir.exists():
            raise CatalogueError('no change summaries found in docs/changes/catalog')
        for path in sorted(self.card_dir.glob('*.md')):
            if path.is_symlink() or not path.resolve().is_relative_to(self.root):
                raise CatalogueError(f'card must be a regular repository file: {path.name}')
            raw = path.read_text(encoding='utf-8')
            if len(raw) > 12000:
                raise CatalogueError(f'{path.name}: card exceeds 12000 characters')
            parts = raw.split('\n---\n', 1) if raw.startswith('---\n') else []
            if len(parts) != 2:
                raise CatalogueError(f'{path.name}: expected JSON frontmatter between --- lines')
            try:
                metadata = json.loads(parts[0][4:], object_pairs_hook=unique_object)
            except CatalogueError as error:
                raise CatalogueError(f'{path.name}: {error}') from error
            except (ValueError, TypeError) as error:
                raise CatalogueError(f'{path.name}: invalid JSON frontmatter') from error
            self.validate_metadata(metadata, path)
            body = parts[1].strip()
            if not body or len(body.split()) > 300:
                raise CatalogueError(f'{path.name}: summary body must contain 1–300 words')
            identifier = metadata['id']
            if identifier in cards:
                raise CatalogueError(f'duplicate card ID: {identifier}')
            cards[identifier] = Card(metadata, body, path.relative_to(self.root).as_posix(), hashlib.sha256(raw.encode()).hexdigest())
        if validate:
            expected = self.inventory()
            missing = sorted(set(expected) - set(cards))
            orphaned = sorted(set(cards) - set(expected))
            if missing or orphaned:
                raise CatalogueError(f'catalogue inventory mismatch; missing={missing}, orphaned={orphaned}')
            for identifier, card in cards.items():
                if card.metadata['source'] != expected[identifier]:
                    raise CatalogueError(f'{identifier}: source does not match its inventory entry')
                for field in ('currentDocs', 'evidence'):
                    for value in card.metadata[field]:
                        self.reference(value, markdown=True)
                for value in card.metadata['code']:
                    self.reference(value)
                self.validate_body_links(card)
                for successor in card.metadata['supersededBy']:
                    if successor not in cards:
                        raise CatalogueError(f'{identifier}: unknown superseding change {successor}')
            self.validate_supersession(cards)
        return cards

    def validate_metadata(self, item, path):
        if not isinstance(item, dict) or set(item) != FIELDS:
            raise CatalogueError(f'{path.name}: metadata fields must be {", ".join(sorted(FIELDS))}')
        for field, maximum in (('id', 160), ('title', 180), ('source', 400), ('summary', 400)):
            if not isinstance(item[field], str) or not item[field].strip() or len(item[field]) > maximum or '\n' in item[field]:
                raise CatalogueError(f'{path.name}: invalid {field}')
        if not SLUG.fullmatch(item['id']) or path.stem != item['id']:
            raise CatalogueError(f'{path.name}: ID must be a slug matching the filename')
        if not isinstance(item['implementation'], str) or not isinstance(item['archive'], str) or item['implementation'] not in IMPLEMENTATIONS or item['archive'] not in ARCHIVES:
            raise CatalogueError(f'{path.name}: invalid implementation/archive status')
        for field in ('components', 'topics', 'keywords', 'currentDocs', 'code', 'evidence', 'supersededBy', 'followUps'):
            values = item[field]
            if not isinstance(values, list) or len(values) > 30 or any(not isinstance(v, str) or not v.strip() or '\n' in v or len(v) > 500 for v in values) or len(values) != len(set(values)):
                raise CatalogueError(f'{path.name}: invalid {field} list')
        for field, maximum in (('components', 8), ('topics', 12)):
            if not item[field] or len(item[field]) > maximum or any(not SLUG.fullmatch(v) or len(v) > 60 for v in item[field]):
                raise CatalogueError(f'{path.name}: {field} must contain lowercase slugs')
        source = self.resolve(item['source'])
        if source.is_symlink() or not source.resolve().is_relative_to(self.changes.resolve()) or source.parent != self.changes or source.name in EXCLUDED:
            raise CatalogueError(f'{path.name}: source must be an original top-level change package/document')
        if not source.is_dir() and (not source.is_file() or source.suffix != '.md'):
            raise CatalogueError(f'{path.name}: source must be a package directory or Markdown file')
        source_id = source.name if source.is_dir() else source.stem
        if source_id != item['id']:
            raise CatalogueError(f'{path.name}: ID does not match its original source')

    def validate_body_links(self, card):
        body = re.sub(r'(?ms)^ {0,3}(`{3,}|~{3,}).*?^ {0,3}\1\s*$', '', card.body)
        for target in re.findall(r'!?\[[^\]\n]*\]\(([^)\n]+)\)', body):
            target = target.split(' "', 1)[0].strip('<>')
            if re.match(r'^[a-zA-Z][a-zA-Z0-9+.-]*:', target):
                continue
            relative, separator, fragment = target.partition('#')
            candidate = self.root / card.path
            if relative:
                candidate = candidate.parent / relative
            resolved = candidate.resolve()
            if not resolved.is_relative_to(self.root):
                raise CatalogueError(f'{card.path}: summary link escapes the repository')
            value = resolved.relative_to(self.root).as_posix() + ('#' + fragment if separator else '')
            self.reference(value)

    @staticmethod
    def validate_supersession(cards):
        visiting, visited = set(), set()

        def visit(identifier):
            if identifier in visiting:
                raise CatalogueError(f'cyclic supersession involving {identifier}')
            if identifier in visited:
                return
            visiting.add(identifier)
            for successor in cards[identifier].metadata['supersededBy']:
                visit(successor)
            visiting.remove(identifier)
            visited.add(identifier)
        for identifier in cards:
            visit(identifier)

    def original_documents(self, card):
        source = self.resolve(card.metadata['source'])
        documents = sorted(source.rglob('*.md')) if source.is_dir() else [source]
        for document in documents:
            if not document.resolve().is_relative_to(source.resolve() if source.is_dir() else source.parent.resolve()):
                raise CatalogueError('original document escapes its change package')
        return documents

    def source_digest(self, card):
        """Fingerprint only during maintenance, never during bounded discovery."""
        source = self.resolve(card.metadata['source'])
        digest = hashlib.sha256()
        for document in self.original_documents(card):
            name = document.relative_to(source).as_posix() if source.is_dir() else document.name
            # Git checkouts may use CRLF; line-ending conversion is not a new
            # historical amendment and must not invalidate another host's index.
            name_bytes, content = name.encode('utf-8'), document.read_text(encoding='utf-8').encode('utf-8')
            for part in (name_bytes, content):
                digest.update(len(part).to_bytes(8, 'big'))
                digest.update(part)
        return digest.hexdigest()

    def select_document(self, card, value):
        path = PurePosixPath(value)
        if path.is_absolute() or '..' in path.parts or '\\' in value:
            raise CatalogueError('document must be a relative path within the selected change')
        source = self.resolve(card.metadata['source'])
        if source.is_dir():
            selected = source / path
            boundary = source
        else:
            if value != source.name:
                raise CatalogueError('document must name the selected standalone change file')
            selected, boundary = source, source.parent
        if not selected.resolve().is_relative_to(boundary.resolve()) or selected not in self.original_documents(card):
            raise CatalogueError('document is not an original Markdown file in the selected change')
        return selected

    def section_rows(self, card, document=None, query=None):
        if query and len(query) > 400:
            raise CatalogueError('section query exceeds 400 characters; use a few keywords')
        documents = [self.select_document(card, document)] if document else self.original_documents(card)
        source = self.resolve(card.metadata['source'])
        rows = []
        terms = tokens(query or '')
        for path in documents:
            text, items = self.document(path)
            if not items:
                items = [Heading('Document', 'document', 1, 1, 0, [], len(text.splitlines()), len(text))]
            for item in items:
                if terms and not all(term in ' '.join([item.title] + item.parents).casefold() for term in terms):
                    continue
                rows.append({'document': path.relative_to(source).as_posix() if source.is_dir() else path.name,
                             'path': path.relative_to(self.root).as_posix(), 'section': item.selector,
                             'heading': item.title[:200], 'parents': [p[:200] for p in item.parents],
                             'startLine': item.line, 'endLine': item.end_line})
        return rows

    def read_section(self, card, document, section, offset, maximum):
        path = self.select_document(card, document)
        text, items = self.document(path)
        if not items:
            items = [Heading('Document', 'document', 1, 1, 0, [], len(text.splitlines()), len(text))]
        item = next((item for item in items if item.selector == section), None)
        if item is None:
            raise CatalogueError('unknown section; use scripts/docs sections to find its selector')
        content = text[item.start:item.end]
        if offset > len(content):
            raise CatalogueError('offset is beyond the selected section')
        part = content[offset:offset + maximum]
        next_offset = offset + len(part) if offset + len(part) < len(content) else None
        start_line = item.line + content[:offset].count('\n')
        end_line = start_line + part.count('\n') - int(part.endswith('\n')) if part else start_line
        return {'id': card.metadata['id'], 'document': document, 'path': path.relative_to(self.root).as_posix(),
                'section': item.selector, 'heading': item.title[:200], 'startLine': start_line,
                'endLine': end_line, 'offset': offset, 'nextOffset': next_offset,
                'totalChars': len(content), 'text': part}


def table(value):
    return value.replace('|', '\\|').replace('\n', ' ')


def artifacts(cards, catalogue):
    ordered = sorted(cards.values(), key=lambda card: card.metadata['id'])
    jsonl = ''.join(json.dumps(dict(card.record(), sourceSha256=catalogue.source_digest(card)), ensure_ascii=False, sort_keys=True) + '\n' for card in ordered)
    lines = ['# Change archive index', '',
             'Generated by `scripts/docs update`; edit the summaries in [catalog/](catalog/).', '',
             'Use [the discovery guide](README.md) for bounded search and section retrieval. '
             'Statuses describe recorded implementation, not deployment or present-day authority.', '',
             f'{len(ordered)} changes. Original documents and assets remain at their existing paths.', '',
             '## Follow-ups and uncertain status', '',
             'These are package-specific historical limits and questions. Check later evidence and '
             'current code before treating them as outstanding work.', '']
    pending = [card for card in ordered if card.metadata['followUps'] or card.metadata['implementation'] in {'proposed', 'partial', 'unknown'}]
    if not pending:
        lines += ['No outstanding work is recorded in the catalogue.', '']
    for card in pending:
        item = card.metadata
        lines.append(f'- [{table(item["title"])}](catalog/{item["id"]}.md) '
                     f'({item["implementation"]}, {item["archive"]}): ' +
                     table('; '.join(item['followUps']) or 'Implementation status needs review.'))
    groups = defaultdict(list)
    for card in ordered:
        groups[card.metadata['topics'][0]].append(card)
    for topic, members in sorted(groups.items()):
        lines += ['', f'## {topic}', '', '| Change | Implementation | Archive | Other topics | Summary |', '| --- | --- | --- | --- | --- |']
        for card in members:
            item = card.metadata
            lines.append(f'| [{table(item["title"])}](catalog/{item["id"]}.md) | '
                         f'{item["implementation"]} | {item["archive"]} | {", ".join(item["topics"][1:])} | {table(item["summary"])} |')
    return {'docs/changes/INDEX.md': '\n'.join(lines) + '\n', 'docs/changes/catalog.jsonl': jsonl}


def search(cards, query, filters, limit):
    if len(query) > 400:
        raise CatalogueError('search query exceeds 400 characters; use a few keywords')
    terms = tokens(query)
    if not terms:
        raise CatalogueError('search needs at least one nontrivial keyword')
    ranked = []
    for card in cards.values():
        item = card.metadata
        if any(value and (value not in item[key] if isinstance(item[key], list) else value != item[key]) for key, value in filters.items()):
            continue
        fields = [(item['id'], 8), (item['title'], 12), (' '.join(item['topics']), 10),
                  (' '.join(item['keywords']), 10), (' '.join(item['components']), 6),
                  (item['summary'], 5), (card.body, 1)]
        fields = [(field_tokens(value), weight) for value, weight in fields]
        if not all(any(term in value for value, _ in fields) for term in terms):
            continue
        score = sum(weight for value, weight in fields for term in terms if term in value)
        if item['implementation'] == 'superseded':
            score *= 0.7
        ranked.append((-score, item['id'], card))
    ranked.sort(key=lambda row: row[:2])
    results = []
    for _, _, card in ranked[:limit]:
        results.append({key: card.record()[key] for key in ('id', 'title', 'summary', 'implementation', 'archive', 'components', 'topics', 'card')})
    return {'query': query, 'total': len(ranked), 'hasMore': len(ranked) > limit, 'results': results}


def bounded_number(minimum, maximum=None):
    def parse(value):
        try:
            result = int(value)
        except ValueError as error:
            raise argparse.ArgumentTypeError('expected an integer') from error
        if result < minimum or maximum is not None and result > maximum:
            raise argparse.ArgumentTypeError(f'expected {minimum}..{maximum}' if maximum else f'expected at least {minimum}')
        return result
    return parse


def parser():
    result = argparse.ArgumentParser(description=__doc__)
    result.add_argument('--root', type=Path, default=Path(__file__).resolve().parents[1], help=argparse.SUPPRESS)
    commands = result.add_subparsers(dest='command', required=True)
    for name in ('update', 'check'):
        commands.add_parser(name)
    find = commands.add_parser('search')
    find.add_argument('query', nargs='+')
    for option in ('component', 'topic'):
        find.add_argument('--' + option)
    find.add_argument('--implementation', choices=sorted(IMPLEMENTATIONS))
    find.add_argument('--archive', choices=sorted(ARCHIVES))
    find.add_argument('--limit', type=bounded_number(1, 10), default=5)
    for name in ('show', 'sections', 'read'):
        command = commands.add_parser(name)
        command.add_argument('id')
        if name in ('sections', 'read'):
            command.add_argument('--document', required=name == 'read')
            command.add_argument('--offset', type=bounded_number(0), default=0)
        if name == 'sections':
            command.add_argument('--query')
            command.add_argument('--limit', type=bounded_number(1, 20), default=10)
        if name == 'read':
            command.add_argument('--section', required=True)
            command.add_argument('--max-chars', type=bounded_number(100, 12000), default=6000)
        command.add_argument('--json', action='store_true')
    find.add_argument('--json', action='store_true')
    return result


def main(argv=None):
    args = parser().parse_args(argv)
    try:
        catalogue = Catalogue(args.root)
        cards = catalogue.load(validate=args.command in {'check', 'update'})
        if args.command in {'check', 'update'}:
            stale = []
            for relative, expected in artifacts(cards, catalogue).items():
                path = catalogue.root / relative
                if path.is_symlink() or not path.parent.resolve().is_relative_to(catalogue.root):
                    raise CatalogueError(f'generated output must stay in the repository: {relative}')
                if args.command == 'update':
                    path.write_text(expected, encoding='utf-8')
                elif not path.exists() or path.read_text(encoding='utf-8') != expected:
                    stale.append(relative)
            if stale:
                raise CatalogueError('stale generated indexes: ' + ', '.join(stale) + '; review affected summaries, then run scripts/docs update')
            print(f'{len(cards)} change summaries verified' + ('; indexes updated.' if args.command == 'update' else '; indexes current.'))
            return 0
        if args.command == 'search':
            value = search(cards, ' '.join(args.query), {'components': args.component, 'topics': args.topic,
                           'implementation': args.implementation, 'archive': args.archive}, args.limit)
        else:
            if args.id not in cards:
                raise CatalogueError(f'unknown change ID: {args.id[:160]}')
            card = cards[args.id]
            if args.command == 'show':
                value = dict(card.record(), text=card.body)
            elif args.command == 'sections':
                rows = catalogue.section_rows(card, args.document, args.query)
                if args.offset > len(rows):
                    raise CatalogueError('offset is beyond the section listing')
                value = {'id': args.id, 'total': len(rows), 'offset': args.offset,
                         'nextOffset': args.offset + args.limit if args.offset + args.limit < len(rows) else None,
                         'sections': rows[args.offset:args.offset + args.limit]}
            else:
                value = catalogue.read_section(card, args.document, args.section, args.offset, args.max_chars)
        if args.json:
            print(json.dumps(value, ensure_ascii=False, indent=2))
        elif args.command == 'search':
            if not value['results']:
                print('No matching changes. Try fewer keywords or different terminology.')
            for item in value['results']:
                print(f'{item["id"]} [{item["implementation"]}; {item["archive"]}]\n  {item["title"]}\n  {item["summary"]}\n  {item["card"]}')
            if value['hasMore']:
                print(f'{value["total"]} matches; showing {len(value["results"])}. Refine the query or filters.')
        elif args.command == 'show':
            print(json.dumps(card.metadata, ensure_ascii=False, indent=2) + '\n\n' + card.body)
        elif args.command == 'sections':
            for item in value['sections']:
                parents = ' > '.join(item['parents'])
                print(f'{item["path"]}:{item["startLine"]}-{item["endLine"]}\n  {item["section"]}: {item["heading"]}' + (f'\n  Under: {parents}' if parents else ''))
            print(f'{value["total"]} matching sections.' + (f' Continue with --offset {value["nextOffset"]}.' if value['nextOffset'] is not None else ''))
        else:
            print(f'{value["path"]}:{value["startLine"]}-{value["endLine"]} ({value["heading"]})\n')
            print(value['text'], end='' if value['text'].endswith('\n') else '\n')
            if value['nextOffset'] is not None:
                print('\n[Section continues; use ' + shlex.join(['scripts/docs', 'read', args.id, '--document', args.document, '--section', args.section, '--offset', str(value['nextOffset']), '--max-chars', str(args.max_chars)]) + ']')
        return 0
    except (CatalogueError, OSError, UnicodeError, ValueError, TypeError) as error:
        print(f'docs: {error}', file=sys.stderr)
        return 1
