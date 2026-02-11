package apiclient

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/vitistack/dnsdist-worker/internal/config"
	"github.com/vitistack/gslb-operator/pkg/auth/jwt"
	"github.com/vitistack/gslb-operator/pkg/bslog"
	"github.com/vitistack/gslb-operator/pkg/models/hash"
	"github.com/vitistack/gslb-operator/pkg/models/pagination"
	"github.com/vitistack/gslb-operator/pkg/models/spoofs"
	"github.com/vitistack/gslb-operator/pkg/rest/request"
	httpclient "github.com/vitistack/gslb-operator/pkg/rest/request/client"
)

type GSLBClient struct {
	builder *request.Builder
	client  httpclient.HTTPClient
}

func NewGSLBClient() (*GSLBClient, error) {
	client, err := httpclient.NewClient(time.Second*5,
		httpclient.WithRequestLogging(slog.Default()),
		httpclient.WithRetry(3),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create http-client: %w", err)
	}

	return &GSLBClient{
		builder: request.NewBuilder(config.GetInstance().API().UpstreamGSLBHost()).SetHeader("User-Agent", config.GetInstance().JWT().User()),
		client:  *client,
	}, nil
}

// fetches all the spoofs from the upstream gslb-operator
func (c *GSLBClient) FetchSpoofs() ([]spoofs.Spoof, error) {
	token, err := jwt.GetInstance().GetServiceToken()
	if err != nil {
		return nil, fmt.Errorf("could not fetch service token: %w", err)
	}

	builder := c.builder.
		GET().
		SetHeader("Authorization", token).
		URL("/spoofs").
		WithURLParams(pagination.NewPaginationParams())

	req, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch spoofs: %w", err)
	}

	spoofsPage := &spoofs.SpoofResponse{}
	err = request.JSONDECODE(resp.Body, spoofsPage)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshall response: %w", err)
	}

	items := make([]spoofs.Spoof, 0)
	items = append(items, spoofsPage.Items...)
	for spoofsPage.Next != nil {
		paginationParams := pagination.NewPaginationParams()
		paginationParams.Page = *spoofsPage.Next

		req, err := builder.WithURLParams(paginationParams).Build()
		if err != nil {
			return nil, fmt.Errorf("could not create request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			bslog.Error("could not fetch spoofs", slog.String("reason", err.Error()))
			return nil, fmt.Errorf("could not fetch spoofs: %w", err)
		}

		spoofsPage := &spoofs.SpoofResponse{}
		err = request.JSONDECODE(resp.Body, spoofsPage)
		if err != nil {
			bslog.Error("could not unmarshall response", slog.String("reason", err.Error()))
			return nil, fmt.Errorf("could not unmarshall response: %w", err)
		}
		items = append(items, spoofsPage.Items...)
	}

	return items, nil
}

func (c *GSLBClient) FetchSpoofsHash() (string, error) {
	token, err := jwt.GetInstance().GetServiceToken()
	if err != nil {
		return "", fmt.Errorf("could not fetch service token: %w", err)
	}

	req, err := c.builder.GET().SetHeader("Authorization", token).URL("/spoofs/hash").Build()
	if err != nil {
		return "", fmt.Errorf("could not create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not fetch hash of spoofs: %w", err)
	}

	hash := hash.Hash{}
	err = json.NewDecoder(resp.Body).Decode(&hash)
	if err != nil {
		return "", fmt.Errorf("failed to decode hash response: %w", err)
	}

	return hash.Hash, nil
}
