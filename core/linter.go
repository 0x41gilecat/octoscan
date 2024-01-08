package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"octoscan/common"
	"octoscan/core/rules"

	"github.com/rhysd/actionlint"
)

type OctoLinter struct {
	actionlint.Linter
}

// not optimal but I can't pass other parameters to OnRulesCreated
var (
	FilterExternalTriggers = false
	FilterRun              = false
	Internetavailable      = true
	DebugRules             = false
	rulesSwitch            = map[string]bool{
		"dangerous-action":       true,
		"dangerous-checkout":     true,
		"expression-injection":   true,
		"dangerous-write":        true,
		"local-action":           true,
		"oidc-action":            true,
		"repo-jacking":           true,
		"shellcheck":             true,
		"credentials":            true,
		"debug-external-trigger": true,
	}
)

func FilterRules(include bool, rulesFiltered []string) {
	for name := range rulesSwitch {
		rulesSwitch[name] = (include == common.IsStringInArray(rulesFiltered, name))
	}
}

func OnRulesCreated(rules []actionlint.Rule) []actionlint.Rule {
	res := filterUnWantedRules(rules)
	res = append(res, offlineRules()...)

	if Internetavailable {
		res = append(res, onlineRules()...)
	}

	return res
}

func filterUnWantedRules(rules []actionlint.Rule) []actionlint.Rule {
	res := []actionlint.Rule{}

	for _, r := range rules {
		// only keep credential and shellcheck rule
		if r.Name() == "credentials" && rulesSwitch["credentials"] {
			res = append(res, r)
		}

		if r.Name() == "shellcheck" && rulesSwitch["shellcheck"] {
			res = append(res, r)
		}
	}

	return res
}

func offlineRules() []actionlint.Rule {

	var res = []actionlint.Rule{}

	if rulesSwitch["dangerous-action"] {
		res = append(res, rules.NewRuleDangerousAction(FilterExternalTriggers))
	}

	if rulesSwitch["dangerous-checkout"] {
		res = append(res, rules.NewRuleDangerousCheckout(FilterExternalTriggers))
	}

	if rulesSwitch["expression-injection"] {
		res = append(res, rules.NewRuleExpressionInjection(FilterExternalTriggers, FilterRun))
	}

	if rulesSwitch["dangerous-write"] {
		res = append(res, rules.NewRuleDangerousWrite(FilterExternalTriggers))
	}

	if rulesSwitch["local-action"] {
		res = append(res, rules.NewRuleLocalAction())
	}

	if rulesSwitch["oidc-action"] {
		res = append(res, rules.NewRuleOIDCAction())
	}

	if DebugRules {
		if rulesSwitch["debug-external-trigger"] {
			res = append(res, rules.NewRuleDebugExternalTrigger())
		}
	}

	return res
}

func onlineRules() []actionlint.Rule {
	var res = []actionlint.Rule{}

	if rulesSwitch["repo-jacking"] {
		res = append(res, rules.NewRuleRepoJacking())
	}

	return res
}

// LintRepositoryRecurse search for all GitHub project recursively and run the linter
func (l *OctoLinter) LintRepositoryRecurse(dir string) {
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		w := filepath.Join(path, ".github", "workflows")
		if s, err := os.Stat(w); err == nil && s.IsDir() {
			l.LintRepository(w)

			return fs.SkipDir
		}

		return nil
	}); err != nil {
		common.Log.Error(fmt.Errorf("could not read files in %q: %w", "./", err))
	}
}
