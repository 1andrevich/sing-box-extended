package pages

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/template/types"
)

func DashboardPage(ctx *context.Context) (types.Panel, error) {

	return types.Panel{
		Title: "Dashboard",
	}, nil
}
