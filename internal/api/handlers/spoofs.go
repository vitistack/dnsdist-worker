package handlers

import (
	"log/slog"
	"net/http"

	"github.com/vitistack/dnsdist-worker/internal/worker"
	"github.com/vitistack/gslb-operator/pkg/bslog"
	"github.com/vitistack/gslb-operator/pkg/models/spoofs"
	"github.com/vitistack/gslb-operator/pkg/rest/request"
	"github.com/vitistack/gslb-operator/pkg/rest/response"
)

type SpoofsHandler struct {
	worker *worker.DNSDISTWorker
}

func NewSpoofsHandler(worker *worker.DNSDISTWorker) *SpoofsHandler {
	return &SpoofsHandler{
		worker: worker,
	}
}

func (sh *SpoofsHandler) CreateSpoof(w http.ResponseWriter, r *http.Request) {
	spoof := &spoofs.Spoof{}
	err := request.JSONDECODE(r.Body, spoof)
	if err != nil {
		bslog.Error("could not decode request-body", slog.String("reason", err.Error()), slog.Any("request_id", r.Context().Value("id")))
		response.Err(w, response.ErrInvalidInput, "unable to parse request body")
		return
	}

	err = sh.worker.CreateSpoof(spoof)
	if err != nil {
		bslog.Error("unable to create spoof", slog.String("reason", err.Error()), slog.Any("request_id", r.Context().Value("id")))
		response.Err(w, response.ErrInternalError, "unable to create spoof")
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (sh *SpoofsHandler) DeleteSpoof(w http.ResponseWriter, r *http.Request) {
	ruleName := r.PathValue("name")
	if ruleName == "" {
		bslog.Info("skipping request due to invalid request input", slog.String("reason", "missing pathvalue name"), slog.Any("request_id", r.Context().Value("id")))
		response.Err(w, response.ErrInvalidInput, "missing rule name")
		return
	}

	err := sh.worker.RemoveSpoof(ruleName)
	if err != nil {
		bslog.Error("unable to create spoof", slog.String("reason", err.Error()), slog.Any("request_id", r.Context().Value("id")))
		response.Err(w, response.ErrInternalError, "unable to create spoof")
		return
	}

}
