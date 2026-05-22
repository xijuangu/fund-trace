package notifier

import (
	"fmt"
	"sync"
	"time"

	"fund-trace/internal/model"

	"github.com/gen2brain/beeep"
)

// Notifier manages alert checking and desktop notifications with cooldown.
type Notifier struct {
	mu        sync.Mutex
	lastAlert map[string]time.Time
	cooldown  time.Duration
}

// New creates a new Notifier with the given cooldown between repeated alerts.
func New(cooldown time.Duration) *Notifier {
	return &Notifier{
		lastAlert: make(map[string]time.Time),
		cooldown:  cooldown,
	}
}

// coalesceKey generates a unique key for fund alert cooldown tracking.
func coalesceKey(fundCode string, alertType model.AlertType) string {
	return fmt.Sprintf("%s_%d", fundCode, int(alertType))
}

// stockCoalesceKey generates a unique key for stock alert cooldown tracking.
func stockCoalesceKey(kind model.AssetKind, market, code string, alertType model.AlertType) string {
	return fmt.Sprintf("stock:%d:%s:%s_%d", int(kind), market, code, int(alertType))
}

// CheckAlerts compares real-time fund data against configured alerts.
// Returns descriptions of triggered alerts. Respects cooldown.
func (n *Notifier) CheckAlerts(funds []model.RealTimeFund, alerts []model.Alert) []model.Alert {
	if n == nil || len(funds) == 0 || len(alerts) == 0 {
		return nil
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now()
	var triggered []model.Alert

	for _, alert := range alerts {
		if !alert.Enabled || alert.Kind == model.AssetKindStock {
			continue
		}
		// Find the fund data
		var fund *model.RealTimeFund
		for i := range funds {
			if funds[i].Code == alert.FundCode {
				fund = &funds[i]
				break
			}
		}
		if fund == nil || !fund.Available {
			continue
		}

		// Check cooldown
		key := coalesceKey(alert.FundCode, alert.Type)
		if lastTime, ok := n.lastAlert[key]; ok {
			if now.Sub(lastTime) < n.cooldown {
				continue
			}
		}

		// Check threshold
		hit := false
		switch alert.Type {
		case model.AlertDrop:
			if fund.DailyChangePct <= alert.ThresholdPct {
				hit = true
			}
		case model.AlertRise:
			if fund.DailyChangePct >= alert.ThresholdPct {
				hit = true
			}
		}

		if hit {
			n.lastAlert[key] = now
			triggered = append(triggered, alert)
		}
	}
	return triggered
}

// CheckStockAlerts compares real-time stock quote data against configured alerts.
// Returns descriptions of triggered alerts. Respects cooldown.
func (n *Notifier) CheckStockAlerts(quotes []model.Quote, alerts []model.Alert) []model.Alert {
	if n == nil || len(quotes) == 0 || len(alerts) == 0 {
		return nil
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now()
	var triggered []model.Alert

	for _, alert := range alerts {
		if !alert.Enabled || alert.Kind != model.AssetKindStock {
			continue
		}

		var quote *model.Quote
		for i := range quotes {
			if quotes[i].Kind == alert.Kind && quotes[i].Market == alert.Market && quotes[i].Code == alert.Code {
				quote = &quotes[i]
				break
			}
		}
		if quote == nil || !quote.Available {
			continue
		}

		key := stockCoalesceKey(alert.Kind, alert.Market, alert.Code, alert.Type)
		if lastTime, ok := n.lastAlert[key]; ok {
			if now.Sub(lastTime) < n.cooldown {
				continue
			}
		}

		hit := false
		switch alert.Type {
		case model.AlertDrop:
			if quote.ChangePct <= alert.ThresholdPct {
				hit = true
			}
		case model.AlertRise:
			if quote.ChangePct >= alert.ThresholdPct {
				hit = true
			}
		}

		if hit {
			n.lastAlert[key] = now
			triggered = append(triggered, alert)
		}
	}
	return triggered
}

// SendAlert sends a desktop notification via beeep.
// Returns nil even if notification fails (best-effort on non-macOS).
func (n *Notifier) SendAlert(title, message string) error {
	if err := beeep.Notify(title, message, ""); err != nil {
		return nil
	}
	return nil
}

// NotifyTriggered sends desktop notifications for all triggered alerts.
func (n *Notifier) NotifyTriggered(triggered []model.Alert, nameMap map[string]string) {
	for _, alert := range triggered {
		if alert.Kind == model.AssetKindStock {
			assetLabel := fmt.Sprintf("%s%s", alert.Market, alert.Code)
			if name, ok := nameMap[model.QuoteKey(alert.Kind, alert.Market, alert.Code)]; ok && name != "" {
				assetLabel = name
			}
			var alertType string
			switch alert.Type {
			case model.AlertDrop:
				alertType = "跌幅"
			case model.AlertRise:
				alertType = "涨幅"
			default:
				alertType = "价格变动"
			}
			title := fmt.Sprintf("股票告警: %s", assetLabel)
			msg := fmt.Sprintf("%s%s %s 达到 %.1f%% 阈值", alert.Market, alert.Code, alertType, alert.ThresholdPct)
			n.SendAlert(title, msg)
			continue
		}

		name := alert.FundCode
		if n, ok := nameMap[alert.FundCode]; ok {
			name = n
		}
		var alertType string
		switch alert.Type {
		case model.AlertDrop:
			alertType = "跌幅"
		case model.AlertRise:
			alertType = "涨幅"
		default:
			alertType = "价格变动"
		}
		title := fmt.Sprintf("基金告警: %s", name)
		msg := fmt.Sprintf("%s %s 达到 %.1f%% 阈值", alert.FundCode, alertType, alert.ThresholdPct)
		n.SendAlert(title, msg)
	}
}
