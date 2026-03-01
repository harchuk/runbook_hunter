package router

import (
	"sort"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/config"
)

type ResolvedDestination struct {
	ID         string
	Type       string
	Telegram   *config.TelegramDestination
	Mattermost *config.MattermostDestination
}

type Engine struct{}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) ComputeRouteKey(labels map[string]string, cfg config.Config) string {
	rules := append([]config.RoutingRule(nil), cfg.Routing.Rules...)
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Priority < rules[j].Priority })
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if labelsMatch(rule.MatchLabels, labels) {
			if rule.RouteKey != "" {
				return rule.RouteKey
			}
			if rule.Name != "" {
				return rule.Name
			}
			return "default"
		}
	}
	return "default"
}

func (e *Engine) SelectDestinations(labels map[string]string, cfg config.Config, forcedIDs []string) []ResolvedDestination {
	ids := forcedIDs
	if len(ids) == 0 {
		ids = e.routeIDs(labels, cfg)
	}
	if len(ids) == 0 {
		return nil
	}
	result := make([]ResolvedDestination, 0, len(ids))
	for _, id := range ids {
		if telegram, ok := findTelegram(cfg.Destinations.Telegram, id); ok && telegram.Enabled {
			t := telegram
			result = append(result, ResolvedDestination{ID: id, Type: "telegram", Telegram: &t})
			continue
		}
		if mattermost, ok := findMattermost(cfg.Destinations.Mattermost, id); ok && mattermost.Enabled {
			m := mattermost
			result = append(result, ResolvedDestination{ID: id, Type: "mattermost", Mattermost: &m})
		}
	}
	return result
}

func (e *Engine) routeIDs(labels map[string]string, cfg config.Config) []string {
	rules := append([]config.RoutingRule(nil), cfg.Routing.Rules...)
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Priority < rules[j].Priority })
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if labelsMatch(rule.MatchLabels, labels) {
			return append([]string(nil), rule.Destinations...)
		}
	}
	return nil
}

func labelsMatch(selector, labels map[string]string) bool {
	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}

func findTelegram(destinations []config.TelegramDestination, id string) (config.TelegramDestination, bool) {
	for _, item := range destinations {
		if item.ID == id {
			return item, true
		}
	}
	return config.TelegramDestination{}, false
}

func findMattermost(destinations []config.MattermostDestination, id string) (config.MattermostDestination, bool) {
	for _, item := range destinations {
		if item.ID == id {
			return item, true
		}
	}
	return config.MattermostDestination{}, false
}
