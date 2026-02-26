package tables

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/modules/db"
	mForm "github.com/GoAdminGroup/go-admin/plugins/admin/modules/form"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/parameter"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"
	"github.com/gofrs/uuid/v5"

	"github.com/sagernet/sing-box/log"
	CM "github.com/sagernet/sing-box/service/manager/constant"
)

func label(ctx *context.Context) types.LabelAttribute {
	return template.Get(ctx, config.GetTheme()).Label().SetType("success")
}

func NodeTableFactory(manager CM.Manager, logger log.Logger) func(ctx *context.Context) (nodeTable table.Table) {
	return func(ctx *context.Context) (nodeTable table.Table) {
		nodeTable = table.NewDefaultTable(ctx, table.Config{
			CanAdd:     true,
			Editable:   true,
			Deletable:  true,
			Exportable: true,
			PrimaryKey: table.PrimaryKey{
				Type: db.Varchar,
				Name: "uuid",
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
		info := nodeTable.GetInfo().SetFilterFormLayout(form.LayoutFilter)
		info.AddField("UUID", "uuid", db.Varchar).
			FieldFilterable().
			FieldSortable()
		info.AddField("Name", "name", db.Varchar).
			FieldFilterable().
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
		info.AddField("Status", "status", db.Varchar).
			FieldDisplay(func(value types.FieldModel) interface{} {
				uuid := value.Row["uuid"].(string)
				return manager.GetNodeStatus(uuid)
			})
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
			filters := make(map[string][]string, len(param.Fields))
			listFilters := make(map[string][]string, len(param.Fields)+2)
			listFilters["offset"] = []string{strconv.Itoa((param.PageInt - 1) * param.PageSizeInt)}
			listFilters["limit"] = []string{param.PageSize}
			for key, values := range param.Fields {
				if key == "__pk" {
					key = "uuid"
				} else {
					if strings.HasPrefix(key, "__") {
						continue
					}
					key = strings.TrimSuffix(key, "__goadmin")
				}
				filters[key] = values
				listFilters[key] = values
			}
			if param.SortField != "" {
				if param.SortType == "asc" {
					listFilters["sort_asc"] = []string{param.SortField}
				} else {
					listFilters["sort_desc"] = []string{param.SortField}
				}
			}
			nodes, err := manager.GetNodes(listFilters)
			if err != nil {
				logger.Error(err)
				return nil, 0
			}
			count, err := manager.GetNodesCount(filters)
			if err != nil {
				logger.Error(err)
				return nil, 0
			}
			result := make([]map[string]interface{}, 0, len(nodes))
			for _, node := range nodes {
				var data map[string]interface{}
				rawData, _ := json.Marshal(node)
				json.Unmarshal(rawData, &data)
				result = append(result, data)
			}
			return result, count
		})

		info.SetDeleteFn(func(ids []string) error {
			for _, uuid := range ids {
				if _, err := manager.DeleteNode(uuid); err != nil {
					return err
				}
			}
			return nil
		})

		info.SetTable("nodes").SetTitle("Nodes").SetDescription("Nodes")

		defaultUUID, _ := uuid.NewV4()
		formList := nodeTable.GetForm()
		formList.AddField("UUID", "uuid", db.Varchar, form.Text).
			FieldMust().
			FieldNotAllowEdit().
			FieldDefault(defaultUUID.String())
		formList.AddField("Name", "name", db.Varchar, form.Text).
			FieldMust()
		formList.AddField("Squads", "squad_ids", db.Varchar, form.Select).
			FieldMust().
			FieldOptions(squadOptions).
			FieldDisableWhenUpdate()

		formList.SetInsertFn(func(values mForm.Values) (err error) {
			squadIDs := make([]int, len(values["squad_ids[]"]))
			for i, rawSquadID := range values["squad_ids[]"] {
				squadID, err := strconv.Atoi(rawSquadID)
				if err != nil {
					return err
				}
				squadIDs[i] = squadID
			}
			_, err = manager.CreateNode(CM.NodeCreate{
				UUID:     values.Get("uuid"),
				Name:     values.Get("name"),
				SquadIDs: squadIDs,
			})
			return
		})

		formList.SetUpdateFn(func(values mForm.Values) (err error) {
			uuid := values.Get("uuid")
			_, err = manager.UpdateNode(uuid, CM.NodeUpdate{
				Name: values.Get("name"),
			})
			return
		})

		formList.SetTable("nodes").SetTitle("Nodes").SetDescription("Nodes")

		return
	}
}
