package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/vitistack/dnsdist-worker/internal/apiclient"
	"github.com/vitistack/dnsdist-worker/internal/model"
	"github.com/vitistack/dnsdist-worker/internal/worker"
	"github.com/vitistack/gslb-operator/pkg/bslog"
	"github.com/vitistack/gslb-operator/pkg/rest/request"
)

type DNSDISTHandler struct {
	gslbClient *apiclient.GSLBClient
	worker     *worker.DNSDISTWorker
}

func NewDNSDistHandler(worker *worker.DNSDISTWorker) (*DNSDISTHandler, error) {
	client, err := apiclient.NewGSLBClient()
	if err != nil {
		return nil, fmt.Errorf("could not create upstream gslb-client: %w", err)
	}
	return &DNSDISTHandler{
		gslbClient: client,
		worker:     worker,
	}, nil
}


func (dh *DNSDISTHandler) NewServerReady(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
	ready := model.DNSDISTServerReady{}
	err := request.JSONDECODE(r.Body, &ready)
	if err != nil {
		bslog.Error("could not decode dnsdist ready", slog.String("reason", err.Error()), slog.String("remoteHost", r.RemoteAddr))
		return
	}


	dh.worker.NewServer(&ready)

	items, err := dh.gslbClient.FetchSpoofs()
	if err != nil {
		bslog.Error("could not fetch spoofs from upstream configuration", slog.String("reason", err.Error()))
		return
	}

	for _, spoof := range items {
		err := dh.worker.CreateServerSpoof(ready.Name, spoof)
		if err != nil {
			bslog.Error("unable to create spoof",
				slog.Group(
					"spoof",
					slog.String("fqdn", spoof.FQDN),
					slog.String("datacenter", spoof.DC),
					slog.String("ip", spoof.IP),
				),
			)
		}
	}
}
