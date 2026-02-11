package worker

import (
	"reflect"
	"testing"

	"github.com/vitistack/gslb-operator/pkg/models/spoofs"
)

// Header + spoof rules
const testRulesBasic = `#   Name                             Matches Rule                                                     Action
0                                          0 All                                                      log to /var/log/dnsdist/queries.log
1   test.nhn.no.:DC1                       0 qname==test.nhn.no.                                      spoof in 10.10.0.2`

// Multiple spoofs
const testRulesMultiple = `#   Name                             Matches Rule                                                     Action
0                                          0 All                                                      log to /var/log/dnsdist/queries.log
1   viti-demo.gslb.test.dns.nhn.no.:BGO1                                  0 qname==viti-demo.gslb.test.dns.nhn.no.                                     spoof in 10.10.0.2
2   viti-demo.gslb.test.dns.nhn.no.:OSL1                                  0 qname==viti-demo.gslb.test.dns.nhn.no.                                     spoof in 10.10.0.3
3   api.example.com.:DC1                                                  5 qname==api.example.com.                                                    spoof in 192.168.1.100`

// With different actions (non-spoof)
const testRulesMixed = `#   Name                             Matches Rule                                                     Action
0   block-rule                             0 qname==malware.com.                                      drop
1   viti-demo.gslb.test.dns.nhn.no.:BGO1                                  0 qname==viti-demo.gslb.test.dns.nhn.no.                                     spoof in 10.10.0.2
2   allow-rule                            12 qname==trusted.com.                                      allow`

// With high match counts
const testRulesWithMatches = `#   Name                             Matches Rule                                                     Action
0                                       1234 All                                                      log to /var/log/dnsdist/queries.log
1   prod.app.example.com.:NYC1                                    99999 qname==prod.app.example.com.                                             spoof in 172.16.0.1
2   test.nhn.no.:BGO1                                                     0 qname==test.nhn.no.                                                     spoof in 10.10.0.2`

// Empty/minimal
const testRulesEmpty = `#   Name                             Matches Rule                                                     Action`

// Only non-spoof rules
const testRulesNoSpoofs = `#   Name                             Matches Rule                                                     Action
0                                          0 All                                                      log to /var/log/dnsdist/queries.log
1   block-malware                          0 qname==badsite.com.                                      drop`

func TestDNSDISTWorker_ParseRuleSet(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		ruleset string
		want    []spoofs.Spoof
		wantErr bool
	}{
		{
			name:    "basic-single-spoof",
			wantErr: false,
			ruleset: testRulesBasic,
			want: []spoofs.Spoof{
				{
					FQDN: "test.nhn.no.",
					DC:   "DC1",
					IP:   "10.10.0.2",
				},
			},
		},
		{
			name:    "multiple-spoofs",
			wantErr: false,
			ruleset: testRulesMultiple,
			want: []spoofs.Spoof{
				{
					FQDN: "viti-demo.gslb.test.dns.nhn.no.",
					DC:   "BGO1",
					IP:   "10.10.0.2",
				},
				{
					FQDN: "viti-demo.gslb.test.dns.nhn.no.",
					DC:   "OSL1",
					IP:   "10.10.0.3",
				},
				{
					FQDN: "api.example.com.",
					DC:   "DC1",
					IP:   "192.168.1.100",
				},
			},
		},
		{
			name:    "mixed-actions-only-spoof-extracted",
			wantErr: false,
			ruleset: testRulesMixed,
			want: []spoofs.Spoof{
				{
					FQDN: "viti-demo.gslb.test.dns.nhn.no.",
					DC:   "BGO1",
					IP:   "10.10.0.2",
				},
			},
		},
		{
			name:    "with-high-match-counts",
			wantErr: false,
			ruleset: testRulesWithMatches,
			want: []spoofs.Spoof{
				{
					FQDN: "prod.app.example.com.",
					DC:   "NYC1",
					IP:   "172.16.0.1",
				},
				{
					FQDN: "test.nhn.no.",
					DC:   "BGO1",
					IP:   "10.10.0.2",
				},
			},
		},
		{
			name:    "empty-rules",
			wantErr: false,
			ruleset: testRulesEmpty,
			want:    []spoofs.Spoof{},
		},
		{
			name:    "no-spoof-rules",
			wantErr: false,
			ruleset: testRulesNoSpoofs,
			want:    []spoofs.Spoof{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w DNSDISTWorker
			got, gotErr := w.ParseRuleSet(tt.ruleset)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ParseRuleSet() failed: %v", gotErr)
				}
				return
			}

			if tt.wantErr {
				t.Fatal("ParseRuleSet() succeeded unexpectedly")
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got: %v, but expected: %v", got, tt.want)
			}
		})
	}
}
