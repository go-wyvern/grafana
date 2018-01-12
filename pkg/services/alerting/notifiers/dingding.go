package notifiers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-wyvern/grafana/pkg/bus"
	"github.com/go-wyvern/grafana/pkg/components/simplejson"
	"github.com/go-wyvern/grafana/pkg/log"
	m "github.com/go-wyvern/grafana/pkg/models"
	"github.com/go-wyvern/grafana/pkg/services/alerting"
)

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:        "dingding",
		Name:        "DingDing",
		Description: "Sends HTTP POST request to DingDing",
		Factory:     NewDingDingNotifier,
		OptionsTemplate: `
      <h3 class="page-heading">DingDing settings</h3>
      <div class="gf-form">
        <span class="gf-form-label width-10">Url</span>
        <input type="text" required class="gf-form-input max-width-26" ng-model="ctrl.model.settings.url" placeholder="https://oapi.dingtalk.com/robot/send?access_token=xxxxxxxxx"></input>
      </div>
    `,
	})

}

func NewDingDingNotifier(model *m.AlertNotification) (alerting.Notifier, error) {
	url := model.Settings.Get("url").MustString()
	if url == "" {
		return nil, alerting.ValidationError{Reason: "Could not find url property in settings"}
	}

	return &DingDingNotifier{
		NotifierBase: NewNotifierBase(model.Id, model.IsDefault, model.Name, model.Type, model.Settings),
		Url:          url,
		log:          log.New("alerting.notifier.dingding"),
	}, nil
}

func (this *DingDingNotifier) ShouldNotify(context *alerting.EvalContext) bool {
	return defaultShouldNotify(context)
}

type DingDingNotifier struct {
	NotifierBase
	Url string
	log log.Logger
}

func (this *DingDingNotifier) Notify(evalContext *alerting.EvalContext) error {
	this.log.Info("Sending dingding")
	var kibanaUrl, env string

	messageUrl, err := evalContext.GetRuleUrl()
	if err != nil {
		this.log.Error("Failed to get messageUrl", "error", err, "dingding", this.Name)
		messageUrl = ""
	}
	this.log.Info("messageUrl:" + messageUrl)

	// message := evalContext.Rule.Message
	// picUrl := evalContext.ImagePublicUrl
	decription := evalContext.Rule.Name
	// statusText := fmt.Sprintf(`- 状态： %s \n`, evalContext.GetStateModel().Text)
	// message = strings.Replace(message, "\"", "\\\"", -1)
	// message = strings.Replace(message, "\t", "", -1)
	// message = strings.Replace(message, "\n", `\n`, -1)

	datasource, err := evalContext.GetDataSource()
	if err != nil {
		this.log.Error("Failed to get datasource", "error", err, "dingding", this.Name)
		return err
	}
	dashboard, err := evalContext.GetDashboard()
	if err != nil {
		this.log.Error("Failed to get dashboard", "error", err, "dingding", this.Name)
		return err
	}
	if strings.Contains(datasource.Url, "pingxx") {
		env = "pingxx"
	} else {
		env = "pinpula"
	}
	if env == "pingxx" {
		kibanaUrl = "http://elk.system.pingxx.com/app/kibana#/discover?_g=(refreshInterval:(display:Off,pause:!f,value:0),time:(from:now-15m,mode:quick,to:now))&_a=(columns:!(_source),index:'%s*',interval:auto,query:(query_string:(analyze_wildcard:!t,query:'%s')),sort:!('@timestamp',desc))"
	} else {
		kibanaUrl = "http://elk.system.pinpula.com/app/kibana#/discover?_g=(refreshInterval:(display:Off,pause:!f,value:0),time:(from:now-15m,mode:quick,to:now))&_a=(columns:!(_source),index:'%s*',interval:auto,query:(query_string:(analyze_wildcard:!t,query:'%s')),sort:!('@timestamp',desc))"
	}

	indexParts := strings.Split(strings.TrimLeft(datasource.Database, "["), "]")
	indexBase := indexParts[0]
	query := strings.Replace(evalContext.Rule.Query, "\"", "\\\"", -1)
	kibanaUrl = fmt.Sprintf(kibanaUrl, indexBase, evalContext.Rule.Query)
	resUri, err := url.Parse(kibanaUrl)
	if err != nil {
		this.log.Error("url parse error", "error", err, "dingding", this.Name)
		return err
	}
	kibanaUrl = resUri.String()

	text := `- 状态：%s
	- 模块名称：%s
	- 异常点描述：%s
	- 环境：%s
	- 查询index：%s
	- 查询query：%s
	- 图表：[dashboard](%s)
	- 日志：[kibana](%s)
	`
	dashboard.Title = strings.Replace(dashboard.Title, "\b", "", -1)
	text = fmt.Sprintf(text, evalContext.GetStateModel().Text, dashboard.Title, decription, env, indexBase+"*", query, messageUrl, kibanaUrl)
	text = strings.Replace(text, "\n", `\n`, -1)
	text = strings.Replace(text, "\t", "", -1)
	// text = strings.Replace(text, "\\\b", " ", -1)

	jsonb := fmt.Sprintf(`{
		"msgtype": "markdown",
		"markdown": {
			"title": "%s",
			"text": "%s"
		},
		"at": {
			"isAtAll": false
		}
	}`, dashboard.Title, text)

	bodyJSON, err := simplejson.NewJson([]byte(jsonb))

	if err != nil {
		this.log.Error("Failed to create Json data", "error", err, "dingding", this.Name)
	}

	body, _ := bodyJSON.MarshalJSON()

	cmd := &m.SendWebhookSync{
		Url:  this.Url,
		Body: string(body),
	}

	if err := bus.DispatchCtx(evalContext.Ctx, cmd); err != nil {
		this.log.Error("Failed to send DingDing", "error", err, "dingding", this.Name)
		return err
	}

	return nil
}
