import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/pages/drafts_page.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('shows draft previews, kinds, thumbnails, and damaged state', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(800, 1200);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final owner = AccountKey('did:plc:alice');
    final items = [
      _draft(
        'standard',
        owner,
        const StandardDraftContent(
          text: 'A knitted blue cardigan',
          languages: ['en'],
        ),
        media: const [_media],
      ),
      _draft(
        'project',
        owner,
        const ProjectDraftContent(
          body: 'Project notes',
          languages: ['en'],
          knownProjectFieldValues: {'title': 'Cardigan project'},
        ),
        kind: LocalPostDraftKind.project,
      ),
      LocalPostDraft.unavailable(id: 'damaged', owner: owner),
    ];
    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: DraftsPageContent(
            items: items,
            onRefresh: () async {},
            onEdit: (_) async {},
            onDelete: (_) async {},
            thumbnailBuilder: (draftId, mediaId) => Text(
              'thumbnail:$draftId:$mediaId',
            ),
          ),
        ),
      ),
    );

    expect(find.text('Drafts'), findsOneWidget);
    expect(find.text('A knitted blue cardigan'), findsOneWidget);
    expect(find.text('Cardigan project'), findsOneWidget);
    expect(find.text('Project notes'), findsOneWidget);
    expect(find.text('Draft unavailable'), findsOneWidget);
    expect(find.textContaining('Standard ·'), findsOneWidget);
    expect(find.textContaining('Project ·'), findsOneWidget);
    expect(find.text('thumbnail:standard:${_media.mediaId}'), findsOneWidget);
    expect(find.byTooltip('Edit draft'), findsNWidgets(2));
    expect(find.byTooltip('Delete draft'), findsNWidgets(3));
  });
}

LocalPostDraft _draft(
  String id,
  AccountKey owner,
  LocalDraftContent content, {
  LocalPostDraftKind kind = LocalPostDraftKind.standard,
  List<DraftMediaDescriptor> media = const [],
}) => LocalPostDraft(
  id: id,
  owner: owner,
  kind: kind,
  createdAt: DateTime.utc(2026, 8, 3, 10),
  updatedAt: DateTime.utc(2026, 8, 3, 11),
  content: content,
  schedule: const DraftScheduleIntent.now(),
  media: media,
);

const _media = DraftMediaDescriptor(
  mediaId: '00000000-0000-4000-8000-000000000002',
  storageRevision: '00000000-0000-4000-8000-000000000003',
  storageFileName: 'image.jpg',
  displayFileName: 'image.jpg',
  mimeType: 'image/jpeg',
  byteLength: 1,
  sha256:
      '0123456789abcdef0123456789abcdef'
      '0123456789abcdef0123456789abcdef',
  width: 1,
  height: 1,
  altText: '',
  order: 0,
);
