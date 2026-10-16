-- LOCAL DEVELOPMENT ONLY. Benign fixtures for the cookie-authenticated case UI.
-- Run once against a migrated craftsky_dev database. Does not reset existing
-- decisions, create operators, grant access or contact a PDS.
BEGIN;
INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at)
VALUES('did:example:console-stage3','active',1,1,'initial',now()),('did:example:console-stage3-viewer','active',1,1,'initial',now()) ON CONFLICT DO NOTHING;
INSERT INTO craftsky_profiles(did,record_cid)
VALUES('did:example:console-stage3','synthetic-stage3-profile'),('did:example:console-stage3-viewer','synthetic-stage3-viewer') ON CONFLICT DO NOTHING;
INSERT INTO account_language_preferences(account_did,primary_language,content_languages)
VALUES('did:example:console-stage3-viewer','en',ARRAY['en']) ON CONFLICT DO NOTHING;
INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at,indexed_at,profile_sort_at)
VALUES('at://did:example:console-stage3/social.craftsky.feed.post/review-one','did:example:console-stage3','review-one','synthetic-stage3-cid','A hand-knit scarf, made slowly. Synthetic moderation walkthrough only.','{}',now(),now(),now())
ON CONFLICT DO NOTHING;
INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state)
VALUES('at://did:example:console-stage3/social.craftsky.feed.post/review-one','post','synthetic-stage3-cid','clear') ON CONFLICT DO NOTHING;
INSERT INTO bluesky_profiles(did,display_name,description,record_cid)
VALUES('did:example:console-stage3-spare','Practice account','A benign synthetic account for console review.','synthetic-stage3-profile'),('did:example:console-stage3-open','Open practice account','A benign synthetic case left open for UI review.','synthetic-stage3-open'),
-- The existing development-only handle resolver requires a mirrored profile.
-- These local identities let consumer HTTP reads verify hide/restore behavior.
('did:example:console-stage3','Synthetic craft account','Benign local moderation fixture.','synthetic-stage3-author'),
('did:example:console-stage3-viewer','Synthetic viewer','Benign local moderation fixture.','synthetic-stage3-viewer') ON CONFLICT DO NOTHING;
INSERT INTO moderation_reports(id,reporter_did,subject_type,subject_did,subject_collection,subject_rkey,subject_uri,subject_cid_snapshot,reason_type,details,forwarding_status,forwarding_prepared_at)
VALUES('console-stage3-report','did:example:console-stage3-reporter','post','did:example:console-stage3','social.craftsky.feed.post','review-one','at://did:example:console-stage3/social.craftsky.feed.post/review-one','synthetic-stage3-cid','harassment','Benign synthetic report for the stage 3 browser walkthrough. No real allegation.','prepared_not_submitted',now()) ON CONFLICT DO NOTHING;
INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,subject_collection,subject_rkey,subject_uri,subject_cid_snapshot,owner_did,safe_snapshot)
VALUES('63000000-0000-4000-8000-000000000001','post:at://did:example:console-stage3/social.craftsky.feed.post/review-one','post','did:example:console-stage3','social.craftsky.feed.post','review-one','at://did:example:console-stage3/social.craftsky.feed.post/review-one','synthetic-stage3-cid','did:example:console-stage3','{"type":"post","did":"did:example:console-stage3","uri":"at://did:example:console-stage3/social.craftsky.feed.post/review-one","cid":"synthetic-stage3-cid"}') ON CONFLICT DO NOTHING;
INSERT INTO moderation_case_reports(case_id,report_id)
VALUES('63000000-0000-4000-8000-000000000001','console-stage3-report') ON CONFLICT DO NOTHING;
INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,owner_did,safe_snapshot,origin)
VALUES('63000000-0000-4000-8000-000000000002','account:did:example:console-stage3-spare','account','did:example:console-stage3-spare','did:example:console-stage3-spare','{"type":"account","did":"did:example:console-stage3-spare"}','operatorCreated'),('63000000-0000-4000-8000-000000000004','account:did:example:console-stage3-open','account','did:example:console-stage3-open','did:example:console-stage3-open','{"type":"account","did":"did:example:console-stage3-open"}','operatorCreated') ON CONFLICT DO NOTHING;
INSERT INTO moderation_case_provenance(id,case_id,actor_id,source_system,reason,created_at)
VALUES('63000000-0000-4000-8000-000000000003','63000000-0000-4000-8000-000000000002','local-fixture','development-fixture','Benign local review practice',now()) ON CONFLICT DO NOTHING;
COMMIT;
