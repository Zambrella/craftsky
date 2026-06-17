package api

import (
	"errors"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
)

const pdsSessionExpiredMessage = "PDS session expired; please sign in again"

func writePDSError(w http.ResponseWriter, fallbackStatus int, fallbackCode, fallbackMessage, runID string, err error) {
	if errors.Is(err, auth.ErrPDSSessionExpired) {
		envelope.WriteError(w, http.StatusUnauthorized,
			"pds_session_expired", pdsSessionExpiredMessage, runID, nil)
		return
	}
	envelope.WriteError(w, fallbackStatus, fallbackCode, fallbackMessage, runID, nil)
}
