package api

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/safetyincident"
	"time"
)

type ConsoleWorkReader struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewConsoleWorkReader(pool *pgxpool.Pool, now func() time.Time) *ConsoleWorkReader {
	if now == nil {
		now = time.Now
	}
	return &ConsoleWorkReader{pool: pool, now: now}
}
func (s *ConsoleWorkReader) SafeWork(ctx context.Context) ([]safetyincident.WorkItem, error) {
	rows, err := s.pool.Query(ctx, `SELECT reference,kind,state,priority,created_at,deadline,owner,cover FROM (
 SELECT i.id::text reference,'imageMatch' kind,i.state,CASE WHEN i.priority='immediate' THEN 1 WHEN i.reporting_deadline<=now() OR i.priority='urgent' THEN 2 ELSE 3 END priority,i.detected_at created_at,i.reporting_deadline deadline,
 COALESCE(i.assigned_to,(SELECT 'operator:'||a.operator_id FROM safety_operator_assignments a WHERE a.incident_id=i.id AND a.assignment_role='owner' AND a.released_at IS NULL ORDER BY a.assigned_at DESC,a.id DESC LIMIT 1),'') owner,
 COALESCE((SELECT 'operator:'||a.operator_id FROM safety_operator_assignments a WHERE a.incident_id=i.id AND a.assignment_role='cover' AND a.released_at IS NULL ORDER BY a.assigned_at DESC,a.id DESC LIMIT 1),'') cover FROM safety_incidents i WHERE i.state IN ('detected','confirmed','caseCreated','reporting')
 UNION ALL
 SELECT id::text,workflow_class,state,CASE WHEN workflow_class='credibleThreat' THEN 1 ELSE 2 END,created_at,deadline_at,responsible_actor,'' FROM safety_workflows WHERE state IN ('open','inProgress')
 UNION ALL
 SELECT i.id::text,'externalIntake',i.intake_kind,CASE WHEN i.urgency='immediate' THEN 1 WHEN i.urgency IN ('urgent','priority') THEN 2 ELSE 5 END,i.created_at,NULL::timestamptz,i.owner_actor_id,'' FROM external_safety_intakes i
 WHERE NOT EXISTS(SELECT 1 FROM safety_workflows w WHERE w.intake_id=i.id) AND NOT EXISTS(SELECT 1 FROM moderation_case_provenance p WHERE p.intake_id=i.id) AND NOT EXISTS(SELECT 1 FROM external_safety_appeal_links a WHERE a.intake_id=i.id)
 ) work ORDER BY priority,deadline NULLS LAST,created_at,reference LIMIT 101`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []safetyincident.WorkItem{}
	for rows.Next() {
		var item safetyincident.WorkItem
		if err := rows.Scan(&item.Reference, &item.Kind, &item.State, &item.Priority, &item.CreatedAt, &item.Deadline, &item.Owner, &item.Cover); err != nil {
			return nil, err
		}
		item.Alert = item.Deadline != nil && !s.now().Before(*item.Deadline)
		items = append(items, item)
	}
	return items, rows.Err()
}
