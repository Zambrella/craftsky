-- Keep the operator's explanation for either appeal outcome. Previously only
-- changed appeals retained it indirectly in their reversed-effect events.
ALTER TABLE moderation_appeals ADD COLUMN resolution_rationale TEXT;
UPDATE moderation_appeals a
SET resolution_rationale = (
    SELECT e.rationale FROM moderation_effect_events e
    WHERE e.case_event_id = a.resolved_event_id AND e.rationale IS NOT NULL
    ORDER BY e.created_at, e.id LIMIT 1
)
WHERE a.status = 'changed';
