package safetyincident

import "github.com/jackc/pgx/v5"

// InTransaction retains domain policy and object-store boundaries while making
// mutations and the console replay receipt share one database commit.
func (s *WorkflowService) InTransaction(tx pgx.Tx) *WorkflowService {
	if s == nil {
		return nil
	}
	copy := *s
	copy.pool = tx
	return &copy
}
func (s *HoldService) InTransaction(tx pgx.Tx) *HoldService {
	if s == nil {
		return nil
	}
	copy := *s
	copy.pool = tx
	return &copy
}
func (s *CSEAWorkflow) InTransaction(tx pgx.Tx) *CSEAWorkflow {
	if s == nil {
		return nil
	}
	copy := *s
	copy.pool = tx
	return &copy
}
func (s *EvidenceService) InTransaction(tx pgx.Tx) *EvidenceService {
	if s == nil {
		return nil
	}
	copy := *s
	copy.transactional = true
	copy.pending = nil
	copy.pool = tx
	return &copy
}
