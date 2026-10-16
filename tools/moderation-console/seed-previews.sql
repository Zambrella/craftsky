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
COMMIT;
