package config

import (
	"fmt"
	"regexp"

	"github.com/allegro/akubra/metrics"
)

// ClientTransportDetail properties
type ClientTransportDetail struct {
	// MaxIdleConns see: https://golang.org/pkg/net/http/#Transport
	// Default 0 (no limit)
	MaxIdleConns int `yaml:"MaxIdleConns" validate:"min=0"`
	// MaxIdleConnsPerHost see: https://golang.org/pkg/net/http/#Transport
	// Default 100
	MaxIdleConnsPerHost int `yaml:"MaxIdleConnsPerHost" validate:"min=0"`
	// IdleConnTimeout see: https://golang.org/pkg/net/http/#Transport
	// Default 0 (no limit)
	IdleConnTimeout metrics.Interval `yaml:"IdleConnTimeout"`
	// ResponseHeaderTimeout see: https://golang.org/pkg/net/http/#Transport
	// Default 5s (no limit)
	ResponseHeaderTimeout metrics.Interval `yaml:"ResponseHeaderTimeout"`
	// DisableKeepAlives see: https://golang.org/pkg/net/http/#Transport
	// Default false
	DisableKeepAlives bool `yaml:"DisableKeepAlives"`
}

// ClientTransportTriggers properties
type ClientTransportTriggers struct {
	Method     string `yaml:"Method" validate:"max=64"`
	Path       string `yaml:"Path" validate:"max=64"`
	QueryParam string `yaml:"QueryParam" validate:"max=64"`
}

// TriggersCompiledRules properties
type TriggersCompiledRules struct {
	MethodRegexp     *regexp.Regexp
	PathRegexp       *regexp.Regexp
	QueryParamRegexp *regexp.Regexp
	IsCompiled       bool
}

// Transport properties
type Transport struct {
	Name                  string                  `yaml:"Name"`
	Triggers              ClientTransportTriggers `yaml:"Triggers"`
	TriggersCompiledRules TriggersCompiledRules
	Details               ClientTransportDetail `yaml:"Details"`
}

// Transports map with Transport
type Transports []Transport

// compileRule
func (t *Transport) compileRule(regexpRule string) (compiledRule *regexp.Regexp, err error) {
	if len(regexpRule) > 0 {
		compiledRule, err = regexp.Compile(regexpRule)
	}
	return
}

// transportFlags for internal matching func
type transportFlags struct {
	declared bool
	matched  bool
	empty    bool
}

// compileRules prepares precompiled regular expressions for rules
func (t *Transport) compileRules() error {
	if !t.TriggersCompiledRules.IsCompiled {
		//TODO: var triggersCompiledRules TriggersCompiledRules


		if len(t.Triggers.Method) > 0 {
			var err error
			t.TriggersCompiledRules.MethodRegexp, err = t.compileRule(t.Triggers.Method)
			if err != nil {
				return fmt.Errorf("compileRule for Client->Transport->Trigger->Method error: %q", err)
			}
		}
		if len(t.Triggers.Path) > 0 {
			var err error
			t.TriggersCompiledRules.PathRegexp, err = t.compileRule(t.Triggers.Path)
			if err != nil {
				return fmt.Errorf("compileRule for Client->Transport->Trigger->Path error: %q", err)
			}
		}
		if len(t.Triggers.QueryParam) > 0 {
			var err error
			t.TriggersCompiledRules.QueryParamRegexp, err = t.compileRule(t.Triggers.QueryParam)
			if err != nil {
				return fmt.Errorf("compileRule for Client->Transport->Trigger->QueryParam error: %q", err)
			}
		}
		t.TriggersCompiledRules.IsCompiled = true
	}
	return nil
}

// GetMatchedTransport returns first details matching with rules from Triggers by arguments: method, path, queryParam
func (t *Transports) GetMatchedTransport(method, path, queryParam string) (matchedTransport Transport, matchedTransportName string, ok bool) {
	for _, transport := range *t {
		err := transport.compileRules()
		if err != nil {
			fmt.Errorf("could't get matched transport - problem with compiling rules")
			return matchedTransport, matchedTransportName, false
		}
		methodFlag, pathFlag, queryParamFlag := matchTransportFlags(transport, method, path, queryParam)

		if methodFlag.matched && pathFlag.matched && queryParamFlag.matched {
			return transport, transport.Name, true
		}
		if methodFlag.empty && pathFlag.empty && queryParamFlag.empty && len(matchedTransportName) == 0 {
			matchedTransport = transport
			matchedTransportName = transport.Name
		}
	}
	return
}

// matchTransportFlags matches method, path and query for Transport
func matchTransportFlags(transport Transport, method, path, queryParam string) (transportFlags, transportFlags, transportFlags) {
	var methodFlag, pathFlag, queryParamFlag transportFlags

	methodFlag.declared, pathFlag.declared, queryParamFlag.declared =
		len(transport.Triggers.Method) > 0, len(transport.Triggers.Path) > 0, len(transport.Triggers.QueryParam) > 0

	if methodFlag.declared {
		methodFlag.matched = transport.TriggersCompiledRules.MethodRegexp.MatchString(method)
	} else {
		methodFlag.empty = true
		methodFlag.matched = true
	}
	if pathFlag.declared {
		pathFlag.matched = transport.TriggersCompiledRules.PathRegexp.MatchString(path)
	} else {
		pathFlag.empty = true
		pathFlag.matched = true
	}
	if queryParamFlag.declared {
		queryParamFlag.matched = transport.TriggersCompiledRules.QueryParamRegexp.MatchString(queryParam)
	} else {
		queryParamFlag.empty = true
		queryParamFlag.matched = true
	}
	return methodFlag, pathFlag, queryParamFlag
}
