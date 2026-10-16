package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/operatorconsole"
)

func ConsoleOperatorCommandHandler(pool *pgxpool.Pool, operation string) http.Handler {
	table, param := "safety_operators", "operatorId"
	if operation == "allow" {
		table = ""
	}
	return ConsoleCommandHandler(pool, table, param, "id", func(tx pgx.Tx) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, _ := ctxkeys.GetModerator(r.Context())
			store := operatorconsole.NewStore(pool, time.Now).InTransaction(tx, actor.ActorID)
			id, _ := uuid.Parse(r.PathValue("operatorId"))
			status := 200
			var err error
			result := map[string]any{"id": id, "operatorId": id}
			switch operation {
			case "allow":
				var body struct {
					DID           syntax.DID `json:"did"`
					PermissionSet string     `json:"permissionSet"`
				}
				if decodeSafetyCommand(r, &body) != nil {
					err = operatorconsole.ErrInvalid
					break
				}
				var op operatorconsole.Operator
				op, err = store.Allow(r.Context(), body.DID, body.PermissionSet)
				result = map[string]any{"id": op.ID, "operatorId": op.ID}
				status = 201
			case "set":
				var body struct {
					PermissionSet string `json:"permissionSet"`
				}
				if decodeSafetyCommand(r, &body) != nil {
					err = operatorconsole.ErrInvalid
					break
				}
				err = store.SetPermissionSet(r.Context(), id, body.PermissionSet)
			default:
				if decodeSafetyCommand(r, &struct{}{}) != nil {
					err = operatorconsole.ErrInvalid
					break
				}
				if operation == "disable" {
					err = store.Disable(r.Context(), id)
				} else if operation == "revoke" {
					err = store.RevokeSessions(r.Context(), id)
				} else {
					err = operatorconsole.ErrInvalid
				}
			}
			if err != nil {
				var pgErr *pgconn.PgError
				switch {
				case errors.Is(err, operatorconsole.ErrInvalid):
					adminError(w, r, 400, "invalid_request", "invalid operator command")
				case errors.Is(err, operatorconsole.ErrLastAdmin):
					adminError(w, r, 422, "last_admin_required", "another active administrator is required")
				case errors.Is(err, operatorconsole.ErrDenied):
					adminError(w, r, 409, "operator_unavailable", "operator unavailable")
				case errors.As(err, &pgErr) && pgErr.Code == "23505":
					adminError(w, r, 409, "operator_already_listed", "operator already listed")
				default:
					consoleCommandFailure(w, r, err)
				}
				return
			}
			writeSafetyCommandJSON(w, status, result)
		})
	})
}

func ConsoleAssignmentCommandHandler(pool *pgxpool.Pool, release bool) http.Handler {
	return ConsoleCommandHandler(pool, "safety_incidents", "incidentReference", "id", func(tx pgx.Tx) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, _ := ctxkeys.GetModerator(r.Context())
			store := operatorconsole.NewStore(pool, time.Now).InTransaction(tx, actor.ActorID)
			incident, _ := uuid.Parse(r.PathValue("incidentReference"))
			var id uuid.UUID
			var err error
			status := 201
			if release {
				status = 200
				id, err = uuid.Parse(r.PathValue("assignmentId"))
				if err == nil && decodeSafetyCommand(r, &struct{}{}) == nil {
					err = store.ReleaseAssignment(r.Context(), incident, id)
				} else {
					err = operatorconsole.ErrInvalid
				}
			} else {
				var body struct {
					OperatorID uuid.UUID `json:"operatorId"`
					Role       string    `json:"role"`
				}
				if decodeSafetyCommand(r, &body) != nil {
					err = operatorconsole.ErrInvalid
				} else {
					id, err = store.AssignIncident(r.Context(), incident, body.OperatorID, body.Role)
				}
			}
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.Is(err, operatorconsole.ErrInvalid) || errors.As(err, &pgErr) && pgErr.Code == "23505" {
					adminError(w, r, 422, "invalid_assignment", "invalid incident assignment")
				} else {
					consoleCommandFailure(w, r, err)
				}
				return
			}
			writeSafetyCommandJSON(w, status, map[string]any{"id": id})
		})
	})
}
