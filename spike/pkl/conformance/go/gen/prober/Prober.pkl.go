// Code generated from Pkl module `spike.contract.urlshortener.Prober`. DO NOT EDIT.
package prober

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"spike.invalid/pklconf/gen/fragments"
	"spike.invalid/pklconf/gen/service"
)

type Prober interface {
	service.Service

	GetInterval() string

	GetKeyPrefix() *string

	GetStatSettle() *string

	GetUrls() Urls

	GetRedirect() Redirect

	GetTls() *fragments.TlsFields
}

var _ Prober = ProberImpl{}

// Always-on synthetic traffic: walk the SAME journeys a real caller does (create a short link, resolve it, watch its counter move), in a loop, over the two Services this release already serves. Separate from the e2e suite: the suite proves a release IS healthy once; this proves it STAYS healthy, so a bake window has signal to read even when nothing real is happening.
type ProberImpl struct {
	service.ServiceImpl

	// How often the loop repeats one full pass of every journey, as a Go duration string, for example "10s".
	Interval string `pkl:"interval"`

	// Marks every long URL and key this prober invents, the same role examples/url-shortener/e2e/suite's own testDataPrefix plays for the e2e suite: a person reading the urls table or the archive bucket by hand can tell synthetic traffic from a real caller's at a glance.
	KeyPrefix *string `pkl:"keyPrefix"`

	// How long the "stat" journey keeps watching a link's click count AFTER it first reads exactly 1, as a Go duration string, and fails if it moves again. A consumer that fails to acknowledge a message has it redelivered after its ack wait, so a click counted twice looks right at the first read and wrong one ack wait later; this window has to outlast at least one redelivery, so it defaults to twice the consumer's 30s ack wait ("60s"). "0s" turns the hold off.
	StatSettle *string `pkl:"statSettle"`

	// The service that owns the URL tables. An address and nothing else: which protocol the caller speaks is in its code, and whether the connection is authenticated is a platform decision this component does not make for itself.
	Urls Urls `pkl:"urls"`

	// The service that resolves a short key. An address and nothing else, on the same terms as `urls` above.
	Redirect Redirect `pkl:"redirect"`

	Tls *fragments.TlsFields `pkl:"tls"`
}

// How often the loop repeats one full pass of every journey, as a Go duration string, for example "10s".
func (rcv ProberImpl) GetInterval() string {
	return rcv.Interval
}

// Marks every long URL and key this prober invents, the same role examples/url-shortener/e2e/suite's own testDataPrefix plays for the e2e suite: a person reading the urls table or the archive bucket by hand can tell synthetic traffic from a real caller's at a glance.
func (rcv ProberImpl) GetKeyPrefix() *string {
	return rcv.KeyPrefix
}

// How long the "stat" journey keeps watching a link's click count AFTER it first reads exactly 1, as a Go duration string, and fails if it moves again. A consumer that fails to acknowledge a message has it redelivered after its ack wait, so a click counted twice looks right at the first read and wrong one ack wait later; this window has to outlast at least one redelivery, so it defaults to twice the consumer's 30s ack wait ("60s"). "0s" turns the hold off.
func (rcv ProberImpl) GetStatSettle() *string {
	return rcv.StatSettle
}

// The service that owns the URL tables. An address and nothing else: which protocol the caller speaks is in its code, and whether the connection is authenticated is a platform decision this component does not make for itself.
func (rcv ProberImpl) GetUrls() Urls {
	return rcv.Urls
}

// The service that resolves a short key. An address and nothing else, on the same terms as `urls` above.
func (rcv ProberImpl) GetRedirect() Redirect {
	return rcv.Redirect
}

func (rcv ProberImpl) GetTls() *fragments.TlsFields {
	return rcv.Tls
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Prober
func LoadFromPath(ctx context.Context, path string) (ret Prober, err error) {
	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		return ret, err
	}
	defer func() {
		cerr := evaluator.Close()
		if err == nil {
			err = cerr
		}
	}()
	ret, err = Load(ctx, evaluator, pkl.FileSource(path))
	return ret, err
}

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Prober
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Prober, error) {
	var ret ProberImpl
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
