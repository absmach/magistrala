package pat

import (
	"fmt"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/permissions"
)

const (
	// Alarms operations
	OpAlarmCreate = "create"
	OpAlarmView   = "view"
	OpAlarmUpdate = "update"
	OpAlarmDelete = "delete"
	OpAlarmList   = "list"

	// Reports operations
	OpReportAdd            = "add"
	OpReportView           = "view"
	OpReportUpdate         = "update"
	OpReportRemove         = "remove"
	OpReportList           = "list"
	OpReportEnable         = "enable"
	OpReportDisable        = "disable"
	OpReportGenerate       = "generate"
	OpReportUpdateSchedule = "update_schedule"
	OpReportUpdateTemplate = "update_template"
	OpReportViewTemplate   = "view_template"
	OpReportDeleteTemplate = "delete_template"

	// Rules operations
	OpRuleAdd            = "add"
	OpRuleView           = "view"
	OpRuleUpdate         = "update"
	OpRuleUpdateTags     = "update_tags"
	OpRuleRemove         = "remove"
	OpRuleList           = "list"
	OpRuleEnable         = "enable"
	OpRuleDisable        = "disable"
	OpRuleUpdateSchedule = "update_schedule"
)

type Operation = permissions.Operation

const (
	AlarmsType auth.EntityType = iota + 9
	ReportsType
	RulesType
)

const (
	AlarmsStr  = "alarms"
	ReportsStr = "reports"
	RulesStr   = "rules"
)

func (et auth.EntityType) String() string {
	switch et {
		case AlarmsType:
		return AlarmsStr
	case ReportsType:
		return ReportsStr
	case RulesType:
		return RulesStr
	default:
		return fmt.Sprintf("unknown domain entity type %d", et)
	}
}

func ParseEntityType(et string) (auth.EntityType, error) {
	switch et {
	case AlarmsStr:
		return AlarmsType, nil
	case ReportsStr:
		return ReportsType, nil
	case RulesStr:
		return RulesType, nil
	default:
		return 0, fmt.Errorf("unknown domain entity type %s", et)
	}
}
