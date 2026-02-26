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

func BandwidthLimiterTableFactory(manager CM.Manager, logger log.Logger) func(ctx *context.Context) table.Table {
	return func(ctx *context.Context) table.Table {
		t := table.NewDefaultTable(ctx, table.Config{
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
		info := t.GetInfo().SetFilterFormLayout(form.LayoutFilter)
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
					{Text: "Global", Value: "global"},
				},
			}).
			FieldSortable()
		info.AddField("Mode", "mode", db.Varchar).
			FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,
				Options: types.FieldOptions{
					{Text: "Download", Value: "download"},
					{Text: "Upload", Value: "upload"},
					{Text: "Duplex", Value: "duplex"},
				},
			}).
			FieldSortable()
		info.AddField("Connection type", "connection_type", db.Varchar).
			FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,
				Options: types.FieldOptions{
					{Text: "HWID", Value: "hwid"},
					{Text: "Mux", Value: "mux"},
					{Text: "IP", Value: "ip"},
				},
			}).
			FieldSortable()
		info.AddField("Speed", "speed", db.Varchar).
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
			items, err := manager.GetBandwidthLimiters(listFilters)
			if err != nil {
				logger.Error(err)
				return nil, 0
			}
			count, err := manager.GetBandwidthLimitersCount(filters)
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
				if _, err := manager.DeleteBandwidthLimiter(i); err != nil {
					return err
				}
			}
			return nil
		})

		info.SetTable("bandwidth_limiters").SetTitle("Bandwidth Limiters").SetDescription("Bandwidth Limiters")

		formList := t.GetForm()
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
				{Text: "Global", Value: "global"},
			}).
			FieldOnChooseOptionsHide([]string{"", "global"}, "connection_type")
		formList.AddField("Mode", "mode", db.Varchar, form.SelectSingle).
			FieldMust().
			FieldOptions(types.FieldOptions{
				{Text: "Download", Value: "download"},
				{Text: "Upload", Value: "upload"},
				{Text: "Duplex", Value: "duplex"},
			})
		formList.AddField("Connection type", "connection_type", db.Varchar, form.SelectSingle).
			FieldOptions(types.FieldOptions{
				{Text: "HWID", Value: "hwid"},
				{Text: "Mux", Value: "mux"},
				{Text: "IP", Value: "ip"},
			})
		formList.AddField("Speed", "speed", db.Varchar, form.Text).
			FieldMust()

		formList.SetInsertFn(func(values mForm.Values) error {
			squadIDs := make([]int, len(values["squad_ids[]"]))
			for i, rawSquadID := range values["squad_ids[]"] {
				squadID, err := strconv.Atoi(rawSquadID)
				if err != nil {
					return err
				}
				squadIDs[i] = squadID
			}
			_, err := manager.CreateBandwidthLimiter(CM.BandwidthLimiterCreate{
				SquadIDs:       squadIDs,
				Username:       values.Get("username"),
				Outbound:       values.Get("outbound"),
				Strategy:       values.Get("strategy"),
				Mode:           values.Get("mode"),
				ConnectionType: values.Get("connection_type"),
				Speed:          values.Get("speed"),
			})
			return err
		})

		formList.SetUpdateFn(func(values mForm.Values) error {
			id, err := strconv.Atoi(values.Get("id"))
			if err != nil {
				return err
			}
			_, err = manager.UpdateBandwidthLimiter(id, CM.BandwidthLimiterUpdate{
				Username:       values.Get("username"),
				Outbound:       values.Get("outbound"),
				Strategy:       values.Get("strategy"),
				Mode:           values.Get("mode"),
				ConnectionType: values.Get("connection_type"),
				Speed:          values.Get("speed"),
			})
			return err
		})

		formList.SetTable("bandwidth_limiters").SetTitle("Bandwidth Limiters").SetDescription("Bandwidth Limiters")
		return t
	}
}
