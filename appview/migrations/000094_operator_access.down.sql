DROP TABLE safety_operator_events;
DROP FUNCTION prevent_operator_event_mutation();
ALTER TABLE safety_operators DROP COLUMN retool_user_id;
ALTER TABLE safety_operator_permissions DROP CONSTRAINT safety_operator_permissions_permission_check;
DELETE FROM safety_operator_permissions WHERE permission IN (
    'moderation.read', 'moderation.manage', 'safety.read', 'intake.manage', 'eligibility.review', 'operator.manage'
);
ALTER TABLE safety_operator_permissions ADD CONSTRAINT safety_operator_permissions_permission_check CHECK (permission IN (
    'incident.readSafe', 'incident.confirm', 'evidence.access', 'evidence.preserve',
    'hold.manage', 'authority.report', 'disclosure.approve', 'scan.retry', 'workflow.manage'
));
