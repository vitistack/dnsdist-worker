package routes

import "net/http"

const (
	ROOT = "/"

	DNSDIST            = ROOT + "dnsdist"
	DNSDIST_READY      = DNSDIST + "/ready"
	POST_DNSDIST_READY = http.MethodPost + " " + DNSDIST_READY

	SPOOFS        = ROOT + "spoofs"
	POST_SPOOFS   = http.MethodPost + " " + SPOOFS
	DELETE_SPOOFS = http.MethodDelete + " " + SPOOFS + "/{name}"
)
