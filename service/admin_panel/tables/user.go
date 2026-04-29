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
	"github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"
	"github.com/go-playground/validator/v10"

	"github.com/sagernet/sing-box/log"
	CM "github.com/sagernet/sing-box/service/manager/constant"
)

func UserTableFactory(manager CM.Manager, logger log.Logger) func(ctx *context.Context) (userTable table.Table) {
	return func(ctx *context.Context) (userTable table.Table) {
		userTable = table.NewDefaultTable(ctx, table.Config{
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
		info := userTable.GetInfo().SetFilterFormLayout(form.LayoutFilter)
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
		info.AddField("Type", "type", db.Varchar).
			FieldFilterable(
				types.FilterType{
					FormType: form.SelectSingle,
					Options: types.FieldOptions{
						{Text: "Hysteria", Value: "hysteria"},
						{Text: "Hysteria2", Value: "hysteria2"},
						{Text: "MTProxy", Value: "mtproxy"},
						{Text: "Trojan", Value: "trojan"},
						{Text: "TUIC", Value: "tuic"},
						{Text: "VLESS", Value: "vless"},
						{Text: "VMess", Value: "vmess"},
					},
				},
			).
			FieldSortable()
		info.AddField("Inbound", "inbound", db.Varchar).FieldFilterable().
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
			filters := make(map[string][]string, len(param.Fields))
			listFilters := make(map[string][]string, len(param.Fields)+2)
			listFilters["offset"] = []string{strconv.Itoa((param.PageInt - 1) * param.PageSizeInt)}
			listFilters["limit"] = []string{param.PageSize}
			for key, values := range param.Fields {
				if key == "__pk" {
					key = "pk"
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
			users, err := manager.GetUsers(listFilters)
			if err != nil {
				logger.Error(err)
				return nil, 0
			}
			count, err := manager.GetUsersCount(filters)
			if err != nil {
				logger.Error(err)
				return nil, 0
			}
			result := make([]map[string]interface{}, 0, len(users))
			for _, user := range users {
				var data map[string]interface{}
				rawData, _ := json.Marshal(user)
				json.Unmarshal(rawData, &data)
				result = append(result, data)
			}
			return result, count
		})
		info.SetDeleteFn(func(ids []string) error {
			for _, id := range ids {
				value, err := strconv.Atoi(id)
				if err != nil {
					return err
				}
				if _, err := manager.DeleteUser(value); err != nil {
					return err
				}
			}
			return nil
		})

		info.SetTable("users").SetTitle("Users").SetDescription("Users")

		formList := userTable.GetForm()
		formList.AddField("ID", "id", db.Int, form.Default).
			FieldNotAllowEdit().
			FieldNotAllowAdd()
		formList.AddField("Squads", "squad_ids", db.Varchar, form.Select).
			FieldMust().
			FieldOptions(squadOptions).
			FieldDisableWhenUpdate()
		formList.AddField("Username", "username", db.Varchar, form.Text).
			FieldMust().
			FieldDisplayButCanNotEditWhenUpdate()
		formList.AddField("Type", "type", db.Varchar, form.SelectSingle).
			FieldMust().
			FieldDisplayButCanNotEditWhenUpdate().
			FieldOptions(types.FieldOptions{
				{Text: "Hysteria", Value: "hysteria"},
				{Text: "Hysteria2", Value: "hysteria2"},
				{Text: "MTProxy", Value: "mtproxy"},
				{Text: "Trojan", Value: "trojan"},
				{Text: "TUIC", Value: "tuic"},
				{Text: "VLESS", Value: "vless"},
				{Text: "VMess", Value: "vmess"},
			}).
			FieldOnChooseOptionsHide([]string{""}, "inbound").
			FieldOnChooseOptionsHide([]string{"", "hysteria", "hysteria2", "mtproxy", "shadowsocks", "trojan", "tuic"}, "uuid").
			FieldOnChooseOptionsHide([]string{"", "mtproxy", "vless", "vmess"}, "password").
			FieldOnChooseOptionsHide([]string{"", "hysteria", "hysteria2", "shadowsocks", "trojan", "tuic", "vless", "vmess"}, "secret").
			FieldOnChooseOptionsHide([]string{"", "hysteria", "hysteria2", "mtproxy", "shadowsocks", "trojan", "tuic", "vmess"}, "flow").
			FieldOnChooseOptionsHide([]string{"", "hysteria", "hysteria2", "mtproxy", "shadowsocks", "trojan", "tuic", "vless"}, "alter_id")
		formList.AddField("Inbound", "inbound", db.Varchar, form.Text).
			FieldMust().
			FieldDisplayButCanNotEditWhenUpdate().
			FieldOptionInitFn(func(val types.FieldModel) types.FieldOptions {
				return types.FieldOptions{
					{Value: val.Value, Text: val.Value, Selected: true},
				}
			})
		formList.AddField("UUID", "uuid", db.Varchar, form.Text)
		formList.AddField("Password", "password", db.Varchar, form.Text)
		formList.AddField("Secret", "secret", db.Varchar, form.Text)
		formList.AddField("Flow", "flow", db.Varchar, form.SelectSingle).
			FieldOptions(types.FieldOptions{
				{Text: "xtls-rprx-vision", Value: "xtls-rprx-vision"},
			})
		formList.AddField("Alter ID", "alter_id", db.Varchar, form.Number).
			FieldDefault("0")

		formList.SetInsertFn(func(values mForm.Values) (err error) {
			squadIDs := make([]int, len(values["squad_ids[]"]))
			for i, rawSquadID := range values["squad_ids[]"] {
				squadID, err := strconv.Atoi(rawSquadID)
				if err != nil {
					return err
				}
				squadIDs[i] = squadID
			}
			var alterId int
			if value := values.Get("alter_id"); value != "" {
				alterId, err = strconv.Atoi(value)
				if err != nil {
					return err
				}
			}
			_, err = manager.CreateUser(CM.UserCreate{
				SquadIDs: squadIDs,
				Username: values.Get("username"),
				Type:     values.Get("type"),
				Inbound:  values.Get("inbound"),
				UUID:     values.Get("uuid"),
				Password: values.Get("password"),
				Secret:   values.Get("secret"),
				Flow:     values.Get("flow"),
				AlterID:  alterId,
			})
			if err != nil {
				if ve, ok := err.(validator.ValidationErrors); ok {
					var errors []string
					for _, e := range ve {
						switch e.Tag() {
						case "required":
							errors = append(errors, e.StructField()+": required field missing")
						case "uuid4":
							errors = append(errors, e.StructField()+": invalid UUID")
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
			var alterId int
			if value := values.Get("alter_id"); value != "" {
				alterId, err = strconv.Atoi(value)
				if err != nil {
					return err
				}
			}
			_, err = manager.UpdateUser(id, CM.UserUpdate{
				UUID:     values.Get("uuid"),
				Password: values.Get("password"),
				Secret:   values.Get("secret"),
				Flow:     values.Get("flow"),
				AlterID:  alterId,
			})
			return
		})

		formList.SetTable("users").SetTitle("Users").SetDescription("Users")

		return
	}
}
