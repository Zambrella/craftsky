-- LOCAL DEVELOPMENT ONLY. Benign, metadata-only incident and failed-scan seeds.
-- No operators, grants, restricted bytes, live subjects or PDS calls are created.
-- Idempotent inserts preserve any walkthrough results on subsequent runs.
BEGIN;
INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at)
VALUES('did:example:console-stage4','active',1,1,'initial',now()) ON CONFLICT DO NOTHING;
INSERT INTO craftsky_profiles(did,record_cid) VALUES('did:example:console-stage4','synthetic-stage4-profile') ON CONFLICT DO NOTHING;
INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at,indexed_at,profile_sort_at)
VALUES('at://did:example:console-stage4/social.craftsky.feed.post/synthetic','did:example:console-stage4','synthetic','synthetic-stage4-cid','A benign hand-knit scarf for a local console walkthrough.','{}',now(),now(),now()) ON CONFLICT DO NOTHING;
INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state)
VALUES('at://did:example:console-stage4/social.craftsky.feed.post/synthetic','post','synthetic-stage4-cid','clear') ON CONFLICT DO NOTHING;
INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state)
VALUES('64000000-0000-4000-8000-000000000011','synthetic-stage4-match','fixture','fixture','fixture','match'),
      ('64000000-0000-4000-8000-000000000012','synthetic-stage4-retry','fixture','fixture','fixture','error') ON CONFLICT DO NOTHING;
INSERT INTO safety_incidents(id,scan_result_id,provider_reference,integrity_metadata_reference,detected_at,updated_at)
VALUES('64000000-0000-4000-8000-000000000001','64000000-0000-4000-8000-000000000011','synthetic-stage4-provider','synthetic-stage4-integrity',now(),now()) ON CONFLICT DO NOTHING;
INSERT INTO safety_incident_subjects(incident_id,owner_did,subject_uri,source_cid,subject_kind,image_slot,linked_at)
VALUES('64000000-0000-4000-8000-000000000001','did:example:console-stage4','at://did:example:console-stage4/social.craftsky.feed.post/synthetic','synthetic-stage4-cid','post','image:0',now()) ON CONFLICT DO NOTHING;
INSERT INTO image_scan_jobs(id,scan_result_id,state,attempts,safe_error_category)
VALUES('64000000-0000-4000-8000-000000000013','64000000-0000-4000-8000-000000000012','dead_letter',4,'synthetic_fixture') ON CONFLICT DO NOTHING;
COMMIT;
