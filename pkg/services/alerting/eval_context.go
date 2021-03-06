package alerting

import (
	"context"
	"fmt"
	"time"

	"github.com/go-wyvern/grafana/pkg/bus"
	"github.com/go-wyvern/grafana/pkg/log"
	m "github.com/go-wyvern/grafana/pkg/models"
	"github.com/go-wyvern/grafana/pkg/setting"
)

type EvalContext struct {
	Firing          bool
	IsTestRun       bool
	EvalMatches     []*EvalMatch
	Logs            []*ResultLogEntry
	Error           error
	ConditionEvals  string
	StartTime       time.Time
	EndTime         time.Time
	Rule            *Rule
	log             log.Logger
	dashboardSlug   string
	ImagePublicUrl  string
	ImageOnDiskPath string
	NoDataFound     bool
	PrevAlertState  m.AlertStateType

	Ctx context.Context
}

func NewEvalContext(alertCtx context.Context, rule *Rule) *EvalContext {
	return &EvalContext{
		Ctx:            alertCtx,
		StartTime:      time.Now(),
		Rule:           rule,
		Logs:           make([]*ResultLogEntry, 0),
		EvalMatches:    make([]*EvalMatch, 0),
		log:            log.New("alerting.evalContext"),
		PrevAlertState: rule.State,
	}
}

type StateDescription struct {
	Color string
	Text  string
	Data  string
}

func (c *EvalContext) GetDataSource() (*m.DataSource, error) {
	getDsInfo := &m.GetDataSourceByIdQuery{
		Id:    c.Rule.DataSourceId,
		OrgId: c.Rule.OrgId,
	}

	if err := bus.Dispatch(getDsInfo); err != nil {
		return nil, fmt.Errorf("Could not find datasource %v", err)
	}
	return getDsInfo.Result, nil
}

func (c *EvalContext) GetDashboard() (*m.Dashboard, error) {
	query := m.GetDashboardQuery{Id: c.Rule.DashboardId, OrgId: c.Rule.OrgId}
	if err := bus.Dispatch(&query); err != nil {
		return nil, fmt.Errorf("Could not find dashboard %v", err)
	}
	return query.Result, nil
}

func (c *EvalContext) GetStateModel() *StateDescription {
	switch c.Rule.State {
	case m.AlertStateOK:
		return &StateDescription{
			Color: "#36a64f",
			Text:  "OK",
		}
	case m.AlertStateNoData:
		return &StateDescription{
			Color: "#888888",
			Text:  "No Data",
		}
	case m.AlertStateAlerting:
		return &StateDescription{
			Color: "#D63232",
			Text:  "Alerting",
		}
	default:
		panic("Unknown rule state " + c.Rule.State)
	}
}

func (c *EvalContext) ShouldUpdateAlertState() bool {
	return c.Rule.State != c.PrevAlertState
}

func (a *EvalContext) GetDurationMs() float64 {
	return float64(a.EndTime.Nanosecond()-a.StartTime.Nanosecond()) / float64(1000000)
}

func (c *EvalContext) GetNotificationTitle() string {
	return "[" + c.GetStateModel().Text + "] " + c.Rule.Name
}

func (c *EvalContext) GetDashboardSlug() (string, error) {
	if c.dashboardSlug != "" {
		return c.dashboardSlug, nil
	}

	slugQuery := &m.GetDashboardSlugByIdQuery{Id: c.Rule.DashboardId}
	if err := bus.Dispatch(slugQuery); err != nil {
		return "", err
	}

	c.dashboardSlug = slugQuery.Result
	return c.dashboardSlug, nil
}

func (c *EvalContext) GetRuleUrl() (string, error) {
	if c.IsTestRun {
		return setting.AppUrl, nil
	}

	if slug, err := c.GetDashboardSlug(); err != nil {
		return "", err
	} else {
		ruleUrl := fmt.Sprintf("%sdashboard/db/%s?fullscreen&edit&tab=alert&panelId=%d&orgId=%d", setting.AppUrl, slug, c.Rule.PanelId, c.Rule.OrgId)
		return ruleUrl, nil
	}
}
