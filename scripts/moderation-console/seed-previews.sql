-- LOCAL DEVELOPMENT ONLY. Benign preview fixtures; no operators, grants,
-- restricted evidence, decisions, credentials or PDS calls. Existing cases and
-- walkthrough results are preserved. Generated textile JPEGs use dev media.
BEGIN;
INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at)
VALUES('did:example:console-rich-preview','active',1,1,'initial',now()) ON CONFLICT DO NOTHING;
INSERT INTO craftsky_profiles(did,crafts,record_cid)
VALUES('did:example:console-rich-preview',ARRAY['knitting','weaving'],'synthetic-rich-profile') ON CONFLICT DO NOTHING;
INSERT INTO bluesky_profiles(did,display_name,description,record_cid)
VALUES('did:example:console-rich-preview','Preview practice account','A benign local account for reviewing textile projects.','synthetic-rich-profile') ON CONFLICT DO NOTHING;
INSERT INTO craftsky_posts(uri,did,rkey,cid,text,images,record,created_at,indexed_at,profile_sort_at)
VALUES('at://did:example:console-rich-preview/social.craftsky.feed.post/preview','did:example:console-rich-preview','preview','synthetic-rich-post','A pair of textile samples. Benign local preview practice only.','[{"cid":"devmedia:console-scarf","mime":"image/jpeg","alt":"A green textile sample"},{"cid":"devmedia:console-weaving","mime":"image/jpeg","alt":"A woven textile sample"}]','{}',now(),now(),now()) ON CONFLICT DO NOTHING;
INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state)
VALUES('at://did:example:console-rich-preview/social.craftsky.feed.post/preview','post','synthetic-rich-post','clear') ON CONFLICT DO NOTHING;
INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,subject_collection,subject_rkey,subject_uri,subject_cid_snapshot,owner_did,safe_snapshot,origin)
VALUES('65000000-0000-4000-8000-000000000001','post:at://did:example:console-rich-preview/social.craftsky.feed.post/preview','post','did:example:console-rich-preview','social.craftsky.feed.post','preview','at://did:example:console-rich-preview/social.craftsky.feed.post/preview','synthetic-rich-post','did:example:console-rich-preview','{}','operatorCreated'),
('65000000-0000-4000-8000-000000000002','account:did:example:console-rich-preview','account','did:example:console-rich-preview',NULL,NULL,NULL,'synthetic-rich-profile','did:example:console-rich-preview','{}','operatorCreated') ON CONFLICT DO NOTHING;
-- Additional benign quote/thread, business and changed-version examples.
INSERT INTO craftsky_business_profiles(owner_did,uri,cid,source_revision,raw_record)
VALUES('did:example:console-rich-preview','at://did:example:console-rich-preview/social.craftsky.business.profile/self','synthetic-business-profile','fixture','{"businessTypes":["maker"],"offerings":["finished-goods"],"tagline":"Textiles made with care","hoursNote":"Weekdays, by appointment","serviceArea":"UK","location":{"country":"GB","locality":"York"},"primaryAction":{"type":"visit-website","destination":"https://example.com/studio"},"products":[{"title":"Woven scarf","price":{"amount":"25","currency":"GBP"},"uri":"https://example.com/scarf"}]}') ON CONFLICT DO NOTHING;
INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state)
VALUES('at://did:example:console-rich-preview/social.craftsky.business.profile/self','businessProfile','synthetic-business-profile','clear') ON CONFLICT DO NOTHING;
INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at,indexed_at,profile_sort_at)
SELECT 'at://did:example:console-rich-preview/social.craftsky.feed.post/'||name,'did:example:console-rich-preview',name,'synthetic-'||name,
CASE name WHEN 'thread-root' THEN 'Thread root: show us your textile samples.' WHEN 'reply-parent' THEN 'Reply parent: which stitch did you use?' WHEN 'quoted' THEN 'Quoted post: a simple weaving pattern.' WHEN 'conversation' THEN 'A reply with a quoted pattern. Benign context preview practice.' ELSE 'This is the edited current version. The earlier version was not retained.' END,
'{}',now(),now(),now() FROM (VALUES('thread-root'),('reply-parent'),('quoted'),('conversation'),('edited')) AS fixtures(name) ON CONFLICT DO NOTHING;
-- Initialise references only on the new, untouched synthetic conversation row.
UPDATE craftsky_posts SET quote_uri='at://did:example:console-rich-preview/social.craftsky.feed.post/quoted',quote_cid='synthetic-quoted',reply_parent_uri='at://did:example:console-rich-preview/social.craftsky.feed.post/reply-parent',reply_parent_cid='synthetic-reply-parent',reply_root_uri='at://did:example:console-rich-preview/social.craftsky.feed.post/thread-root',reply_root_cid='synthetic-thread-root'
WHERE uri='at://did:example:console-rich-preview/social.craftsky.feed.post/conversation' AND cid='synthetic-conversation' AND quote_uri IS NULL;
INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state)
SELECT 'at://did:example:console-rich-preview/social.craftsky.feed.post/'||name,'post','synthetic-'||name,'clear'
FROM (VALUES('thread-root'),('reply-parent'),('quoted'),('conversation'),('edited')) AS fixtures(name) ON CONFLICT DO NOTHING;
INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,subject_collection,subject_rkey,subject_uri,subject_cid_snapshot,owner_did,safe_snapshot,origin)
SELECT id::uuid,'post:at://did:example:console-rich-preview/social.craftsky.feed.post/'||name,'post','did:example:console-rich-preview','social.craftsky.feed.post',name,'at://did:example:console-rich-preview/social.craftsky.feed.post/'||name,captured,'did:example:console-rich-preview','{}','operatorCreated'
FROM (VALUES('65000000-0000-4000-8000-000000000003','conversation','synthetic-conversation'),('65000000-0000-4000-8000-000000000004','edited','synthetic-edited-before')) AS fixtures(id,name,captured) ON CONFLICT DO NOTHING;

COMMIT;
