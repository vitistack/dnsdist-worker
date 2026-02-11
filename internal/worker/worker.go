package worker

import (
	"bufio"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/vitistack/dnsdist-worker/internal/apiclient"
	"github.com/vitistack/dnsdist-worker/internal/model"
	"github.com/vitistack/dnsdist-worker/pkg/dnsdist"
	"github.com/vitistack/gslb-operator/pkg/bslog"
	"github.com/vitistack/gslb-operator/pkg/models/spoofs"
)

const (
	DEFAULT_SYNC_JOB = time.Minute * 1
)

var startup sync.Once

type DNSDISTWorker struct {
	servers    map[string]*dnsdist.Client
	gslbClient *apiclient.GSLBClient
	mu         sync.Mutex
	wg         sync.WaitGroup
}

func NewWorker(servers string) (*DNSDISTWorker, error) {
	var worker *DNSDISTWorker
	initError := fmt.Errorf("impossible to create more than one dnsdist worker") // default error

	startup.Do(func() {
		initError = nil // reset init value when no worker has been created
		gslbClient, err := apiclient.NewGSLBClient()
		if err != nil {
			worker = nil
			initError = fmt.Errorf("could not create DNSDist worker: %w", err)
			return
		}

		worker = &DNSDISTWorker{
			servers:    make(map[string]*dnsdist.Client),
			gslbClient: gslbClient,
			mu:         sync.Mutex{},
			wg:         sync.WaitGroup{},
		}

		err = worker.LoadDNSDISTServersConfig(servers)
		if err != nil {
			worker = nil
			initError = fmt.Errorf("could not load dnsdist servers config: %w", err)
			return
		}

		err = worker.SynchronizeServers()
		if err != nil {
			worker = nil
			initError = fmt.Errorf("startup sequence failed: %w", err)
			return
		}
	})

	return worker, initError
}

func (w *DNSDISTWorker) LoadDNSDISTServersConfig(servers string) error {
	rawConfig, err := os.ReadFile(servers)
	if err != nil {
		return fmt.Errorf("could not read file: %w", err)
	}

	dnsdistServers := []model.DNSDISTServerReady{}
	err = json.Unmarshal(rawConfig, &dnsdistServers)
	if err != nil {
		return fmt.Errorf("error while unmarshalling configuration: %w", err)
	}

	for _, server := range dnsdistServers {
		client, err := dnsdist.NewClient(
			server.Key,
			dnsdist.WithHost(server.Host),
			dnsdist.WithPort(server.Port),
		)
		if err != nil {
			return fmt.Errorf("unable to create dnsdist-client: %w", err)
		}

		w.servers[server.Name] = client
	}

	return nil
}

func (w *DNSDISTWorker) NewServer(server *model.DNSDISTServerReady) error {
	client, err := dnsdist.NewClient(
		server.Key,
		dnsdist.WithHost(server.Host),
		dnsdist.WithPort(server.Port),
	)

	if err != nil {
		return fmt.Errorf("unable to create new dnsdist client: %w", err)
	}

	w.servers[server.Name] = client

	return nil
}

func (w *DNSDISTWorker) CreateServerSpoof(server string, spoof spoofs.Spoof) error {
	client, ok := w.servers[server]
	if !ok {
		return fmt.Errorf("server with name: %s is not registered", server)
	}
	err := client.AddDomainSpoof(fmt.Sprintf("%s:%s", spoof.FQDN, spoof.DC), spoof.FQDN, spoof.IP)
	if err != nil {
		return fmt.Errorf("could not create spoof on %s: %w", server, err)
	}

	return nil
}

func (w *DNSDISTWorker) CreateSpoof(spoof *spoofs.Spoof) error {
	for server, client := range w.servers {
		err := client.AddDomainSpoof(spoof.FQDN+":"+spoof.DC, spoof.FQDN, spoof.IP)
		if err != nil {
			return fmt.Errorf("could not create spoof on server %s: %w", server, err)
		}
	}
	return nil
}

func (w *DNSDISTWorker) RemoveSpoof(ruleName string) error {
	for server, client := range w.servers {
		err := client.RmRuleWithName(ruleName)
		if err != nil {
			return fmt.Errorf("failed to remove spoof %s: %w", server, err)
		}
	}
	return nil
}

func (w *DNSDISTWorker) SynchronizeRemoteConfiguration(ctx context.Context) {
	w.wg.Go(func() {
		for {
			select {
			case <-time.After(DEFAULT_SYNC_JOB):
				w.SynchronizeServers()
			case <-ctx.Done():
				return
			}
		}
	})
}

func (w *DNSDISTWorker) SynchronizeServers() error {
	desiredHash, err := w.gslbClient.FetchSpoofsHash()
	if err != nil {
		return fmt.Errorf("failed to fetch hash: %w", err)
	}

	wg := sync.WaitGroup{}
	for server, client := range w.servers {
		wg.Go(func() {
			rawRuleSet, err := client.ShowRules()
			if err != nil {
				bslog.Error("unable to fetch ruleset from dnsdist server", slog.String("reason", err.Error()))
				return
			}

			data, err := w.ParseRuleSet(rawRuleSet)
			if err != nil {
				bslog.Error("could not synchronize dnsdist server", slog.String("reason", err.Error()))
			}

			slices.SortFunc(data, func(a, b spoofs.Spoof) int {
				return cmp.Compare(fmt.Sprintf("%s:%s", a.FQDN, a.DC), fmt.Sprintf("%s:%s", b.FQDN, b.DC))
			})

			marshalledSpoofs, err := json.Marshal(data)
			if err != nil {
				bslog.Error("unable to marshall spoofs", slog.String("reason", err.Error()))
				return
			}

			rawHash := sha256.Sum256(marshalledSpoofs) // creating bytes representation of spoofs
			hash := hex.EncodeToString(rawHash[:])
			if hash != desiredHash {
				err := w.reconcile(client, data)
				if err != nil {
					bslog.Warn("failed to reconcile server", slog.String("server_name", server))
				}
			}
		})
	}

	wg.Wait()

	return nil
}

// Parses the current ruleset and returns a slice of all the spoof rules that were found
func (w *DNSDISTWorker) ParseRuleSet(ruleset string) ([]spoofs.Spoof, error) {
	reader := strings.NewReader(ruleset)
	lines := bufio.NewScanner(reader)

	pattern, err := regexp.Compile(`[a-zA-Z0-9._-]+:[A-Z0-9]+|spoof|\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	if err != nil {
		return nil, fmt.Errorf("unable to compile regex: %w", err)
	}

	spoofRules := make([]spoofs.Spoof, 0)
	for lines.Scan() {
		line := lines.Text()
		matches := pattern.FindAllString(line, -1)
		if len(matches) < 3 {
			continue
		}
		rule := model.Rule{
			Name:   matches[0],
			Action: matches[1],
		}

		if rule.Action != "spoof" {
			continue
		}

		spoofRules = append(spoofRules,
			spoofs.Spoof{
				FQDN: strings.Split(rule.Name, ":")[0],
				DC:   strings.Split(rule.Name, ":")[1],
				IP:   matches[2],
			})
	}

	return spoofRules, nil
}

// removes all unwanted spoofs
// and adds all wanted spoofs that does not exist
func (w *DNSDISTWorker) reconcile(client *dnsdist.Client, configuredSpoofs []spoofs.Spoof) error {
	gslbspoofs, err := w.gslbClient.FetchSpoofs()
	if err != nil {
		return fmt.Errorf("could not fetch spoofs: %w", err)
	}

	for _, spoof := range configuredSpoofs { // remove all spoofs that should not exist any more
		if !slices.ContainsFunc(gslbspoofs, func(s spoofs.Spoof) bool {
			return s.FQDN+":"+s.DC == spoof.FQDN+":"+spoof.DC
		}) {
			err := client.RmRuleWithName(spoof.FQDN + ":" + spoof.DC)
			if err != nil {
				return fmt.Errorf("could not remove spoof: %w", err)
			}
		}
	}

	for _, spoof := range gslbspoofs { // add all spoofs that does not exist but should
		if !slices.ContainsFunc(configuredSpoofs, func(s spoofs.Spoof) bool {
			return s.FQDN+":"+s.DC == spoof.FQDN+":"+spoof.DC
		}) {
			err := client.AddDomainSpoof(spoof.FQDN+":"+spoof.DC, spoof.FQDN, spoof.IP)
			if err != nil {
				return fmt.Errorf("could not remove spoof: %w", err)
			}
		}
	}

	return nil
}
