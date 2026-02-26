package tables

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	mForm "github.com/GoAdminGroup/go-admin/plugins/admin/modules/form"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/parameter"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"
	"github.com/go-playground/validator/v10"

	"github.com/sagernet/sing-box/log"
	CM "github.com/sagernet/sing-box/service/manager/constant"
)

func SquadTableFactory(manager CM.Manager, logger log.Logger) func(ctx *context.Context) (squadTable table.Table) {
	return func(ctx *context.Context) (squadTable table.Table) {
		squadTable = table.NewDefaultTable(ctx, table.Config{
			CanAdd:     true,
			Editable:   true,
			Deletable:  true,
			Exportable: true,
			PrimaryKey: table.PrimaryKey{
				Type: db.Int,
				Name: table.DefaultPrimaryKeyName,
			},
		})

		info := squadTable.GetInfo().SetFilterFormLayout(form.LayoutFilter)
		info.AddField("ID", "id", db.Int).
			FieldSortable()
		info.AddField("Name", "name", db.Varchar).
			FieldFilterable().
			FieldSortable()
		info.AddField("Created At", "created_at", db.Datetime).
			FieldDisplay(func(model types.FieldModel) interface{} {
				t, err := time.Parse(time.RFC3339, model.Value)
				if err != nil {
					return model.Value
				}
				return t.Format("2006-01-02 15:04:05")
			}).
			FieldSortable().
			FieldFilterable(types.FilterType{FormType: form.DatetimeRange})
		info.AddField("Updated At", "updated_at", db.Datetime).
			FieldDisplay(func(model types.FieldModel) interface{} {
				t, err := time.Parse(time.RFC3339, model.Value)
				if err != nil {
					return model.Value
				}
				return t.Format("2006-01-02 15:04:05")
			}).
			FieldSortable().
			FieldFilterable(types.FilterType{FormType: form.DatetimeRange})

		info.SetGetDataFn(func(param parameter.Parameters) ([]map[string]interface{}, int) {
			filters := make(map[string][]string, len(param.Fields))
			listFilters := make(map[string][]string, len(param.Fields)+2)
			listFilters["offset"] = []string{strconv.Itoa((param.PageInt - 1) * param.PageSizeInt)}
			listFilters["limit"] = []string{param.PageSize}
			for key, values := range param.Fields {
				if key == "__pk" {
					key = "pk"
				} else if strings.HasPrefix(key, "__") {
					continue
				} else {
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
			squads, err := manager.GetSquads(listFilters)
			if err != nil {
				logger.Error(err)
				return nil, 0
			}
			count, err := manager.GetSquadsCount(filters)
			if err != nil {
				logger.Error(err)
				return nil, 0
			}
			result := make([]map[string]interface{}, 0, len(squads))
			for _, squad := range squads {
				var data map[string]interface{}
				rawData, _ := json.Marshal(squad)
				json.Unmarshal(rawData, &data)
				result = append(result, data)
			}
			return result, count
		})

		info.SetDeleteFn(func(ids []string) error {
			for _, id := range ids {
				intID, err := strconv.Atoi(id)
				if err != nil {
					return err
				}
				if _, err := manager.DeleteSquad(intID); err != nil {
					return err
				}
			}
			return nil
		})

		info.SetTable("squads").SetTitle("Squads").SetDescription("Squads")

		formList := squadTable.GetForm()
		formList.AddField("ID", "id", db.Int, form.Default).
			FieldNotAllowAdd().
			FieldNotAllowEdit()
		formList.AddField("Name", "name", db.Varchar, form.Text).
			FieldMust()

		formList.SetInsertFn(func(values mForm.Values) (err error) {
			_, err = manager.CreateSquad(CM.SquadCreate{
				Name: values.Get("name"),
			})
			if err != nil {
				if ve, ok := err.(validator.ValidationErrors); ok {
					var errors []string
					for _, e := range ve {
						switch e.Tag() {
						case "required":
							errors = append(errors, e.StructField()+": required field missing")
						default:
							errors = append(errors, e.StructField()+": invalid request")
						}
					}
					err = fmt.Errorf("%s", strings.Join(errors, "<br>"))
				}
			}
			return
		})

		formList.SetUpdateFn(func(values mForm.Values) (err error) {
			id, err := strconv.Atoi(values.Get("id"))
			if err != nil {
				return err
			}
			_, err = manager.UpdateSquad(id, CM.SquadUpdate{
				Name: values.Get("name"),
			})
			return
		})

		formList.SetTable("squads").SetTitle("Squads").SetDescription("Squads")

		return
	}
}
