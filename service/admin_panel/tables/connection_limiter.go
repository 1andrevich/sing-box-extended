package tables

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	mForm "github.com/GoAdminGroup/go-admin/plugins/admin/modules/form"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/parameter"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"

	"github.com/sagernet/sing-box/log"
	CM "github.com/sagernet/sing-box/service/manager/constant"
)

func ConnectionLimiterTableFactory(manager CM.Manager, logger log.Logger) func(ctx *context.Context) table.Table {
	return func(ctx *context.Context) table.Table {
		connectionLimiterTable := table.NewDefaultTable(ctx, table.Config{
			CanAdd:     true,
			Editable:   true,
			Deletable:  true,
			Exportable: true,
			PrimaryKey: table.PrimaryKey{
				Type: db.Int,
				Name: table.DefaultPrimaryKeyName,
			},
		})
		squads, err := manager.GetSquads(map[string][]string{})
		if err != nil {
			return nil
		}
		squadsByID := make(map[int]string, len(squads))
		squadOptions := make(types.FieldOptions, len(squads))
		for i, squad := range squads {
			squadsByID[squad.ID] = squad.Name
			squadOptions[i] = types.FieldOption{
				Text:  squad.Name,
				Value: strconv.Itoa(squad.ID),
			}
		}
		info := connectionLimiterTable.GetInfo().SetFilterFormLayout(form.LayoutFilter)
		info.AddField("ID", "id", db.Int).
			FieldSortable()
		info.AddField("Squads", "squad_ids", db.Varchar).
			FieldDisplay(func(model types.FieldModel) interface{} {
				values := model.Row["squad_ids"].([]interface{})
				labels := template.HTML("")
				labelTpl := label(ctx).SetType("success")
				labelValues := make([]string, len(values))
				for i, squadID := range values {
					labelValues[i] = squadsByID[int(squadID.(float64))]
				}
				for key, label := range labelValues {
					if key == len(labelValues)-1 {
						labels += labelTpl.SetContent(template.HTML(label)).GetContent()
					} else {
						labels += labelTpl.SetContent(template.HTML(label)).GetContent()
					}
				}
				return labels
			})
		info.AddField("Username", "username", db.Varchar).
			FieldFilterable().
			FieldSortable()
		info.AddField("Outbound", "outbound", db.Varchar).
			FieldFilterable().
			FieldSortable()
		info.AddField("Strategy", "strategy", db.Varchar).
			FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,
				Options: types.FieldOptions{
					{Text: "Connection", Value: "connection"},
				},
			}).
			FieldSortable()
		info.AddField("Connection type", "connection_type", db.Varchar).
			FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,
				Options: types.FieldOptions{
					{Text: "Mux", Value: "mux"},
					{Text: "HWID", Value: "hwid"},
					{Text: "IP", Value: "ip"},
				},
			}).
			FieldSortable()
		info.AddField("Lock type", "lock_type", db.Varchar).
			FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,
				Options: types.FieldOptions{
					{Text: "Manager", Value: "manager"},
				},
			}).
			FieldSortable()
		info.AddField("Count", "count", db.Int).
			FieldSortable()
		info.AddField("Created at", "created_at", db.Datetime).
			FieldDisplay(func(model types.FieldModel) interface{} {
				t, err := time.Parse(time.RFC3339, model.Value)
				if err != nil {
					return model.Value
				}
				return t.Format("2006-01-02 15:04:05")
			}).
			FieldFilterable(types.FilterType{FormType: form.DatetimeRange}).
			FieldSortable()
		info.AddField("Updated at", "updated_at", db.Datetime).
			FieldDisplay(func(model types.FieldModel) interface{} {
				t, err := time.Parse(time.RFC3339, model.Value)
				if err != nil {
					return model.Value
				}
				return t.Format("2006-01-02 15:04:05")
			}).
			FieldFilterable(types.FilterType{FormType: form.DatetimeRange}).
			FieldSortable()

		info.SetGetDataFn(func(param parameter.Parameters) ([]map[string]interface{}, int) {
			filters := make(map[string][]string)
			listFilters := map[string][]string{
				"offset": {strconv.Itoa((param.PageInt - 1) * param.PageSizeInt)},
				"limit":  {param.PageSize},
			}
			for k, v := range param.Fields {
				if strings.HasPrefix(k, "__") {
					continue
				}
				key := strings.TrimSuffix(k, "__goadmin")
				filters[key] = v
				listFilters[key] = v
			}
			if param.SortField != "" {
				if param.SortType == "asc" {
					listFilters["sort_asc"] = []string{param.SortField}
				} else {
					listFilters["sort_desc"] = []string{param.SortField}
				}
			}
			items, err := manager.GetConnectionLimiters(listFilters)
			if err != nil {
				logger.Error(err)
				return nil, 0
			}
			count, err := manager.GetConnectionLimitersCount(filters)
			if err != nil {
				logger.Error(err)
				return nil, 0
			}
			result := make([]map[string]interface{}, 0, len(items))
			for _, item := range items {
				var data map[string]interface{}
				raw, _ := json.Marshal(item)
				json.Unmarshal(raw, &data)
				result = append(result, data)
			}
			return result, count
		})

		info.SetDeleteFn(func(ids []string) error {
			for _, id := range ids {
				i, err := strconv.Atoi(id)
				if err != nil {
					return err
				}
				if _, err := manager.DeleteConnectionLimiter(i); err != nil {
					return err
				}
			}
			return nil
		})

		info.SetTable("connection_limiters").SetTitle("Connection Limiters").SetDescription("Connection Limiters")

		formList := connectionLimiterTable.GetForm()
		formList.AddField("ID", "id", db.Int, form.Default).
			FieldNotAllowAdd().
			FieldNotAllowEdit()
		formList.AddField("Squads", "squad_ids", db.Varchar, form.Select).
			FieldMust().
			FieldOptions(squadOptions).
			FieldDisableWhenUpdate()
		formList.AddField("Username", "username", db.Varchar, form.Text).
			FieldMust().
			FieldDisplayButCanNotEditWhenUpdate()
		formList.AddField("Outbound", "outbound", db.Varchar, form.Text).
			FieldMust().
			FieldDisplayButCanNotEditWhenUpdate()
		formList.AddField("Strategy", "strategy", db.Varchar, form.SelectSingle).
			FieldMust().
			FieldOptions(types.FieldOptions{
				{Text: "Connection", Value: "connection"},
			}).
			FieldDefault("connection")
		formList.AddField("Connection type", "connection_type", db.Varchar, form.SelectSingle).
			FieldOptions(types.FieldOptions{
				{Text: "Mux", Value: "mux"},
				{Text: "HWID", Value: "hwid"},
				{Text: "IP", Value: "ip"},
			})
		formList.AddField("Lock type", "lock_type", db.Varchar, form.SelectSingle).
			FieldOptions(types.FieldOptions{
				{Text: "Manager", Value: "manager"},
			})
		formList.AddField("Count", "count", db.Int, form.Number).
			FieldMust().
			FieldDefault("0")

		formList.SetInsertFn(func(values mForm.Values) error {
			squadIDs := make([]int, len(values["squad_ids[]"]))
			for i, rawSquadID := range values["squad_ids[]"] {
				squadID, err := strconv.Atoi(rawSquadID)
				if err != nil {
					return err
				}
				squadIDs[i] = squadID
			}
			count, err := strconv.ParseUint(values.Get("count"), 10, 32)
			if err != nil {
				return err
			}
			_, err = manager.CreateConnectionLimiter(CM.ConnectionLimiterCreate{
				SquadIDs:       squadIDs,
				Username:       values.Get("username"),
				Outbound:       values.Get("outbound"),
				Strategy:       values.Get("strategy"),
				ConnectionType: values.Get("connection_type"),
				LockType:       values.Get("lock_type"),
				Count:          uint32(count),
			})
			return err
		})

		formList.SetUpdateFn(func(values mForm.Values) error {
			id, err := strconv.Atoi(values.Get("id"))
			if err != nil {
				return err
			}
			count, err := strconv.ParseUint(values.Get("count"), 10, 32)
			if err != nil {
				return err
			}
			_, err = manager.UpdateConnectionLimiter(id, CM.ConnectionLimiterUpdate{
				Username:       values.Get("username"),
				Outbound:       values.Get("outbound"),
				Strategy:       values.Get("strategy"),
				ConnectionType: values.Get("connection_type"),
				LockType:       values.Get("lock_type"),
				Count:          uint32(count),
			})
			return err
		})

		formList.SetTable("connection_limiters").SetTitle("Connection Limiters").SetDescription("Connection Limiters")
		return connectionLimiterTable
	}
}
