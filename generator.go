package module

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/portalenergy/pe-request-generator/actions"
	"github.com/portalenergy/pe-request-generator/db"
	"github.com/portalenergy/pe-request-generator/fields"
	"github.com/portalenergy/pe-request-generator/icontext"
	"github.com/portalenergy/pe-request-generator/locale"
	"github.com/portalenergy/pe-request-generator/response"
	"github.com/portalenergy/pe-request-generator/utils"
)

const (
	GeneratorErrorAdd    string = "Cannot create record"
	GeneratorErrorUpdate string = "Cannot update record"
	GeneratorErrorDelete string = "Cannot delete record"
)

type Generator struct {
	db                   func(module *BaseModule) db.DBExecutor
	group                gin.RouterGroup
	Modules              []*BaseModule
	Features             []Features
	AuthMiddleware       func(module actions.ModuleAction) gin.HandlerFunc
	PermissionMiddleware func(action actions.ModuleAction, permissions []actions.Role) gin.HandlerFunc
	Locales              []locale.Lang
	DefaultLocale        locale.Lang
}

func NewGenerator(
	db func(module *BaseModule) db.DBExecutor,
	group gin.RouterGroup,
	modules []*BaseModule,
	permissionMiddleware func(action actions.ModuleAction, permissions []actions.Role) gin.HandlerFunc,
	authMiddleware func(action actions.ModuleAction) gin.HandlerFunc,
) *Generator {
	return &Generator{
		db:                   db,
		group:                group,
		Modules:              modules,
		Features:             []Features{},
		PermissionMiddleware: permissionMiddleware,
		AuthMiddleware:       authMiddleware,
		Locales:              []locale.Lang{locale.EN},
		DefaultLocale:        locale.EN,
	}
}

func (generator *Generator) getLang(c *gin.Context) locale.Lang {
	if lang := c.Query("lang"); lang != "" {
		l := locale.Lang(lang)
		for _, s := range generator.Locales {
			if s == l {
				return l
			}
		}
	}
	return locale.ParseAcceptLanguage(c.GetHeader("Accept-Language"), generator.Locales, generator.DefaultLocale)
}

func (generator *Generator) localeStrings() []string {
	result := make([]string, len(generator.Locales))
	for i, l := range generator.Locales {
		result[i] = string(l)
	}
	return result
}

func (generator *Generator) FeaturesMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		l, _ := icontext.GetLogger(ctx)
		lang := generator.getLang(c)

		localized := make([]Features, len(generator.Features))
		for i, f := range generator.Features {
			lf := Features{
				ModuleName: locale.Resolve(f.ModuleNameLabels, lang, f.ModuleName),
				Actions:    make(map[string]FeaturesActions, len(f.Actions)),
			}
			for k, a := range f.Actions {
				lf.Actions[k] = FeaturesActions{
					Label: locale.Resolve(a.Labels, lang, a.Label),
					Url:   a.Url,
					Type:  a.Type,
					Roles: a.Roles,
				}
			}
			localized[i] = lf
		}

		i18n := make(map[string]map[string]FeaturesI18n, len(generator.Locales))
		for _, loc := range generator.Locales {
			locStr := string(loc)
			moduleI18n := make(map[string]FeaturesI18n, len(generator.Features))
			for _, f := range generator.Features {
				fi := FeaturesI18n{
					ModuleName: locale.Resolve(f.ModuleNameLabels, loc, f.ModuleName),
					Actions:    make(map[string]string, len(f.Actions)),
				}
				for k, a := range f.Actions {
					fi.Actions[k] = locale.Resolve(a.Labels, loc, a.Label)
				}
				moduleI18n[f.ModuleName] = fi
			}
			i18n[locStr] = moduleI18n
		}

		resp := FeaturesResponse{
			Locale:  string(lang),
			Locales: generator.localeStrings(),
			Modules: localized,
			I18n:    i18n,
		}

		response.Response(l, c, resp)
	}
}

func (generator *Generator) Run() {

	featuresGroup := generator.group.Group("/api")
	featuresGroup.GET("/features", generator.FeaturesMiddleware())

	for _, module := range generator.Modules {
		featuresModule := Features{
			ModuleName:       module.Label,
			ModuleNameLabels: module.Labels,
			Actions:          make(map[string]FeaturesActions),
		}

		for _, action := range module.Actions {
			switch action.Action() {
			case actions.ModuleActionNameList:

				listAction, _ := action.(actions.ListModuleAction)
				featuresModule.Actions["list"] = FeaturesActions{
					Label:  listAction.Label,
					Labels: listAction.Labels,
					Url:    module.Path + "/" + module.Name,
					Type:   "GET",
					Roles:  listAction.Permission,
				}
				listGrpup := generator.group.Group(module.Path)
				if listAction.Auth {
					if generator.AuthMiddleware == nil {
						panic(fmt.Sprintf("auth middleware not implemented in module: %s", module.Name))
					}
					listGrpup.Use(generator.AuthMiddleware(listAction))
				}
				if len(listAction.Permission) > 0 {
					if generator.PermissionMiddleware == nil {
						panic(fmt.Sprintf("permission middleware not implemented in module: %s", module.Name))
					}
					listGrpup.Use(generator.PermissionMiddleware(listAction, listAction.Permission))
				}

				listGrpup.GET(module.Name, generator.actionList(module, listAction))
			case actions.ModuleActionNameAdd:
				addAction, _ := action.(actions.AddModuleAction)
				featuresModule.Actions["add"] = FeaturesActions{
					Label:  addAction.Label,
					Labels: addAction.Labels,
					Url:    module.Path + "/" + module.Name,
					Type:   "PUT",
					Roles:  addAction.Permission,
				}
				featuresModule.Actions["defrec"] = FeaturesActions{
					Label:  addAction.Label,
					Labels: addAction.Labels,
					Url:    fmt.Sprintf("%s/%s/defrec/", module.Path, module.Name),
					Type:   "GET",
					Roles:  addAction.Permission,
				}
				addGrpup := generator.group.Group(module.Path)
				if addAction.Auth {
					if generator.AuthMiddleware == nil {
						panic(fmt.Sprintf("auth middleware not implemented in module: %s", module.Name))
					}
					addGrpup.Use(generator.AuthMiddleware(addAction))
				}
				if len(addAction.Permission) > 0 {
					if generator.PermissionMiddleware == nil {
						panic(fmt.Sprintf("permission middleware not implemented in module: %s", module.Name))
					}
					addGrpup.Use(generator.PermissionMiddleware(addAction, addAction.Permission))
				}
				addGrpup.PUT(module.Name, generator.actionAdd(module, addAction))

				defrecGroup := generator.group.Group(fmt.Sprintf("%s/%s/defrec", module.Path, module.Name))
				defrecGroup.GET("/", generator.actionDefrec(module))

			case actions.ModuleActionNameView:
				viewAction, _ := action.(actions.ViewModuleAction)
				featuresModule.Actions["view"] = FeaturesActions{
					Label:  viewAction.Label,
					Labels: viewAction.Labels,
					Url:    module.Path + "/" + module.Name,
					Type:   "GET",
					Roles:  viewAction.Permission,
				}
				viewGrout := generator.group.Group(module.Path)
				if viewAction.Auth {
					if generator.AuthMiddleware == nil {
						panic(fmt.Sprintf("auth middleware not implemented in module: %s", module.Name))
					}
					viewGrout.Use(generator.AuthMiddleware(viewAction))
				}
				if len(viewAction.Permission) > 0 {
					if generator.PermissionMiddleware == nil {
						panic(fmt.Sprintf("permission middleware not implemented in module: %s", module.Name))
					}
					viewGrout.Use(generator.PermissionMiddleware(viewAction, viewAction.Permission))
				}

				viewGrout.GET(fmt.Sprintf("%s/view/:bykey/:value", module.Name), generator.actionView(module, viewAction))
			case actions.ModuleActionNameUpdate:
				updateAction, _ := action.(actions.UpdateModuleAction)
				featuresModule.Actions["update"] = FeaturesActions{
					Label:  updateAction.Label,
					Labels: updateAction.Labels,
					Url:    module.Path + "/" + module.Name,
					Type:   "POST",
					Roles:  updateAction.Permission,
				}
				updateGroup := generator.group.Group(module.Path)
				if updateAction.Auth {
					if generator.AuthMiddleware == nil {
						panic(fmt.Sprintf("auth middleware not implemented in module: %s", module.Name))
					}
					updateGroup.Use(generator.AuthMiddleware(updateAction))
				}
				if len(updateAction.Permission) > 0 {
					if generator.PermissionMiddleware == nil {
						panic(fmt.Sprintf("permission middleware not implemented in module: %s", module.Name))
					}
					updateGroup.Use(generator.PermissionMiddleware(updateAction, updateAction.Permission))
				}

				updateGroup.POST(fmt.Sprintf("%s/:bykey/:value", module.Name), generator.actionUpdate(module, updateAction))
			case actions.ModuleActionNameDelete:
				deleteAction, _ := action.(actions.DeleteModuleAction)
				featuresModule.Actions["delete"] = FeaturesActions{
					Label:  deleteAction.Label,
					Labels: deleteAction.Labels,
					Url:    module.Path + "/" + module.Name,
					Type:   "DELETE",
					Roles:  deleteAction.Permission,
				}
				deleteGroup := generator.group.Group(module.Path)
				if deleteAction.Auth {
					if generator.AuthMiddleware == nil {
						panic(fmt.Sprintf("auth middleware not implemented in module: %s", module.Name))
					}
					deleteGroup.Use(generator.AuthMiddleware(deleteAction))
				}
				if len(deleteAction.Permission) > 0 {
					if generator.PermissionMiddleware == nil {
						panic(fmt.Sprintf("permission middleware not implemented in module: %s", module.Name))
					}
					deleteGroup.Use(generator.PermissionMiddleware(deleteAction, deleteAction.Permission))
				}
				deleteGroup.DELETE(fmt.Sprintf("%s/delete/:bykey/:value", module.Name), generator.actionDelete(module, deleteAction))
			}
		}

		generator.Features = append(generator.Features, featuresModule)
	}
}

func (generator *Generator) actionList(module *BaseModule, action actions.ListModuleAction) func(c *gin.Context) {
	return func(c *gin.Context) {
		defer action.AfterRequest(c)

		ctx := c.Request.Context()
		l, _ := icontext.GetLogger(ctx)
		role := actions.GetRoleFromContext(c)
		lang := generator.getLang(c)

		if hook := actions.ResolveRoleHook(module.RoleBeforeHook, role); hook != nil {
			if err := hook(c); err != nil {
				response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
				return
			}
		}
		defer func() {
			if hook := actions.ResolveRoleAfterHook(module.RoleAfterHook, role); hook != nil {
				hook(c)
			}
		}()

		err := action.BeforeRequest(c)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
			return
		}

		page := int64QueryParam(c, "page", 0)
		size := int64QueryParam(c, "size", 3000)
		isCSV := int64QueryParam(c, "csv", 0)
		filters := generator.normalizeFilters(c.QueryMap("filter"), module, action, lang)
		searchText := c.Query("search")
		addFilters := c.Query("addFilters")
		addHeads := c.Query("addHeads")

		columns := action.GetColumns(c)

		realFields := make([]fields.ModuleField, 0, 10)
		for _, realField := range module.Fields {
			if containsColumn(columns, realField.Column) {
				realFields = append(realFields, realField)
			}
		}

		var where pg.BoolExpression
		if whereFn := actions.ResolveRoleWhere(module.RoleWhere, role); whereFn != nil {
			where = whereFn(c)
		}
		if action.Where != nil {
			actionWhere := action.Where(c)
			if where != nil {
				where = pg.AND(where, actionWhere)
			} else {
				where = actionWhere
			}
		}

		joins := action.Join
		if roleJoins := actions.ResolveRoleJoin(module.RoleJoin, role); roleJoins != nil {
			joins = append(roleJoins, joins...)
		}

		results, count, err := generator.db(module).List(
			l,
			module.Table,
			module.PrimaryKey,
			realFields,
			page,
			size,
			action.Search,
			searchText,
			filters,
			where,
			joins,
		)

		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
			return
		}

		var heads map[string]string
		if addHeads == "true" {
			heads = make(map[string]string)

			for _, realField := range module.Fields {
				if containsColumn(columns, realField.Column) {
					heads[realField.ColumnName()] = locale.Resolve(realField.Titles, lang, realField.Title)
				}
			}
		}

		var filter map[string]fields.ModuleFilterField
		if addFilters == "true" {
			filter = make(map[string]fields.ModuleFilterField)
			for _, realField := range module.Fields {
				if containsColumn(action.Filter, realField.Column) {
					options := make([]fields.ModuleFieldOptions, 0, 10)
					if realField.Options != nil {
						for _, item := range realField.Options {
							options = append(options, item)
						}
					}
					if realField.OptionsFunc != nil {
						for _, item := range realField.OptionsFunc(c) {
							options = append(options, item)
						}
					}
					roleStr := string(actions.GetRoleFromContext(c))
					for _, ro := range realField.RoleOptions {
						if ro.Role == roleStr || ro.Role == string(actions.RoleAll) {
							options = append(options, ro.Options...)
							break
						}
					}

					for i, opt := range options {
						options[i].Label = locale.Resolve(opt.Labels, lang, opt.Label)
					}

					filterField := fields.ModuleFilterField{
						Column:   realField.Column,
						Title:    locale.Resolve(realField.Titles, lang, realField.Title),
						Type:     realField.Type,
						FormType: realField.FormType,
						Example:  realField.Example,
						Options:  options,
						Check:    realField.Check,
						Convert:  realField.Convert,
					}
					filter[realField.ColumnName()] = filterField
				}
			}
		}

		if len(results) == 0 {
			results = make([]interface{}, 0, 10)
		}

		if len(heads) == 0 {
			heads = make(map[string]string)
		}

		var headsI18n map[string]map[string]string
		if addHeads == "true" {
			headsI18n = make(map[string]map[string]string, len(generator.Locales))
			for _, loc := range generator.Locales {
				locHeads := make(map[string]string)
				for _, realField := range module.Fields {
					if containsColumn(columns, realField.Column) {
						locHeads[realField.ColumnName()] = locale.Resolve(realField.Titles, loc, realField.Title)
					}
				}
				headsI18n[string(loc)] = locHeads
			}
		}

		output := struct {
			Count     int64                               `json:"count"`
			Size      int64                               `json:"size"`
			Page      int64                               `json:"page"`
			Locale    string                              `json:"locale"`
			Locales   []string                            `json:"locales"`
			Extra     interface{}                         `json:"extra"`
			Rows      []interface{}                       `json:"rows"`
			Heads     map[string]string                   `json:"heads"`
			HeadsI18n map[string]map[string]string        `json:"heads_i18n,omitempty"`
			Filters   map[string]fields.ModuleFilterField `json:"filters,omitempty"`
		}{
			Count:     count,
			Size:      size,
			Page:      page,
			Locale:    string(lang),
			Locales:   generator.localeStrings(),
			Extra:     action.Extra,
			Rows:      results,
			Heads:     heads,
			HeadsI18n: headsI18n,
			Filters:   filter,
		}

		if isCSV == 0 {
			response.Response(l, c, output)
		} else {
			resultJsonString, err := json.Marshal(results)
			if err != nil {
				response.ErrorResponse(l, c, http.StatusInternalServerError, err.Error(), nil)
				return
			}

			var d []map[string]interface{}
			err = json.Unmarshal(resultJsonString, &d)
			if err != nil {
				response.ErrorResponse(l, c, http.StatusInternalServerError, err.Error(), nil)
				return
			}

			csvResults := make([][]string, 0, 10)
			keys := make([]string, 0, 10)
			for _, v := range d {
				for key := range v {
					keys = append(keys, key)
				}
				break
			}
			sort.Strings(keys)
			csvResults = append(csvResults, keys)

			for _, v := range d {
				values := make([]string, 0, 10)
				for _, key := range keys {
					valueString, err := json.Marshal(v[key])
					if err != nil {
						continue
					}

					values = append(values, string(valueString))
				}
				csvResults = append(csvResults, values)
			}

			b := new(bytes.Buffer)
			w := csv.NewWriter(b)
			w.Comma = '\t'
			err = w.WriteAll(csvResults)

			response.ResponseCSV(l, c, b.Bytes())
		}
	}
}

func (generator *Generator) actionAdd(module *BaseModule, action actions.AddModuleAction) func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		l, _ := icontext.GetLogger(ctx)
		role := actions.GetRoleFromContext(c)
		lang := generator.getLang(c)

		if hook := actions.ResolveRoleHook(module.RoleBeforeHook, role); hook != nil {
			if err := hook(c); err != nil {
				response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
				return
			}
		}
		defer func() {
			if hook := actions.ResolveRoleAfterHook(module.RoleAfterHook, role); hook != nil {
				hook(c)
			}
		}()

		err := action.BeforeRequest(c)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorAdd, []string{
				err.Error(),
			})
			return
		}

		var input map[string]interface{}
		err = utils.ParseJson(c.Request, &input)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorAdd, []string{
				"Parse Input Error",
			})
			return
		}

		errs := generator.checkRequest(c, input, module, action, fields.ScenarioAdd, lang)
		if len(errs) > 0 {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorAdd, errs)
			return
		}

		columns := action.GetColumns(c)

		realFields := make([]fields.ModuleField, 0, 10)
		for _, realField := range module.Fields {
			if containsColumn(columns, realField.Column) {
				realFields = append(realFields, realField)
			}
		}

		mapInput := generator.mapRequestInput(input, module, columns)
		output, err := generator.db(module).Add(l, module.Table, module.PrimaryKey, realFields, mapInput)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorAdd, []string{
				err.Error(),
			})
			return
		}

		response.Response(l, c, output)

		action.AfterRequest(c)
	}
}

func (generator *Generator) actionDefrec(module *BaseModule) func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		l, _ := icontext.GetLogger(ctx)
		lang := generator.getLang(c)

		err := module.Defrec.BeforeRequest(c)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
			return
		}

		output := make([]fields.ModuleField, 0, 10)

		role := string(actions.GetRoleFromContext(c))

		for _, field := range module.Fields {
			checkItems := make([]fields.CheckRules, 0, 10)
			optionItems := make([]fields.ModuleFieldOptions, 0, 10)

			if field.Options != nil {
				for _, option := range field.Options {
					optionItems = append(optionItems, option)
				}
			}
			if field.OptionsFunc != nil {
				for _, option := range field.OptionsFunc(c) {
					optionItems = append(optionItems, option)
				}
			}
			for _, ro := range field.RoleOptions {
				if ro.Role == role || ro.Role == string(actions.RoleAll) {
					optionItems = append(optionItems, ro.Options...)
					break
				}
			}

			for i, opt := range optionItems {
				optionItems[i].Label = locale.Resolve(opt.Labels, lang, opt.Label)
			}

			if field.Check != nil {
				for _, check := range field.Check {
					checkItems = append(checkItems, check)
				}
			}
			if field.CheckFunc != nil {
				for _, check := range field.CheckFunc(c) {
					checkItems = append(checkItems, check)
				}
			}
			for _, rc := range field.RoleCheck {
				if rc.Role == role || rc.Role == string(actions.RoleAll) {
					checkItems = append(checkItems, rc.Rules...)
					break
				}
			}

			field.Title = locale.Resolve(field.Titles, lang, field.Title)
			field.Options = optionItems
			field.Check = checkItems

			output = append(output, field)
		}

		response.Response(l, c, response.NewDefrecResponse(nil, output, lang, generator.Locales))

		module.Defrec.AfterRequest(c)
	}
}

func (generator *Generator) actionView(module *BaseModule, action actions.ViewModuleAction) func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		l, _ := icontext.GetLogger(ctx)
		role := actions.GetRoleFromContext(c)

		if hook := actions.ResolveRoleHook(module.RoleBeforeHook, role); hook != nil {
			if err := hook(c); err != nil {
				response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
				return
			}
		}
		defer func() {
			if hook := actions.ResolveRoleAfterHook(module.RoleAfterHook, role); hook != nil {
				hook(c)
			}
		}()

		err := action.BeforeRequest(c)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
			return
		}

		whereKey := c.Param("bykey")

		allowedKeys := make([]interface{}, 0, len(action.By))
		for _, col := range action.By {
			allowedKeys = append(allowedKeys, col.Name())
		}
		err = validation.In(allowedKeys...).Error(fmt.Sprintf(`allowed keys %v`, allowedKeys)).Validate(whereKey)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorDelete, []string{
				err.Error(),
			})
			return
		}

		whereValue := c.Param("value")
		if len(whereValue) == 0 {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorDelete, []string{
				"value param not found",
			})
			return
		}

		columns := action.GetColumns(c)

		realFields := make([]fields.ModuleField, 0, 10)
		for _, realField := range module.Fields {
			if containsColumn(columns, realField.Column) {
				realFields = append(realFields, realField)
			}
		}

		where := pg.RawBool(
			fmt.Sprintf(`%s."%s" = #val`, module.Table.Alias(), whereKey),
			pg.RawArgs{"#val": whereValue},
		)
		if module.Table.Alias() == "" {
			where = pg.RawBool(
				fmt.Sprintf(`"%s"."%s" = #val`, module.Table.TableName(), whereKey),
				pg.RawArgs{"#val": whereValue},
			)
		}

		joins := action.Join
		if roleJoins := actions.ResolveRoleJoin(module.RoleJoin, role); roleJoins != nil {
			joins = append(roleJoins, joins...)
		}

		result, err := generator.db(module).View(l, module.Table, module.PrimaryKey, realFields, where, joins)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
			return
		}

		response.Response(l, c, result)

		action.AfterRequest(c)
	}
}

func (generator *Generator) actionUpdate(module *BaseModule, action actions.UpdateModuleAction) func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		l, _ := icontext.GetLogger(ctx)
		role := actions.GetRoleFromContext(c)
		lang := generator.getLang(c)

		if hook := actions.ResolveRoleHook(module.RoleBeforeHook, role); hook != nil {
			if err := hook(c); err != nil {
				response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
				return
			}
		}
		defer func() {
			if hook := actions.ResolveRoleAfterHook(module.RoleAfterHook, role); hook != nil {
				hook(c)
			}
		}()

		err := action.BeforeRequest(c)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorUpdate, nil)
			return
		}

		whereKey := c.Param("bykey")
		allowedKeys := make([]interface{}, 0, len(action.By))
		for _, col := range action.By {
			allowedKeys = append(allowedKeys, col.Name())
		}
		err = validation.In(allowedKeys...).Error(fmt.Sprintf(`allowed keys %v`, allowedKeys)).Validate(whereKey)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorUpdate, []string{
				err.Error(),
			})
			return
		}

		whereValue := c.Param("value")
		if len(whereValue) == 0 {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorUpdate, []string{
				"value param not found",
			})
			return
		}

		var input map[string]interface{}
		err = utils.ParseJson(c.Request, &input)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorUpdate, nil)
			return
		}

		errs := generator.checkRequest(c, input, module, action, fields.ScenarioUpdate, lang)
		if len(errs) > 0 {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorUpdate, errs)
			return
		}

		columns := action.GetColumns(c)

		realFields := make([]fields.ModuleField, 0, 10)
		for _, realField := range module.Fields {
			if containsColumn(columns, realField.Column) {
				realFields = append(realFields, realField)
			}
		}

		mapInput := generator.mapRequestInput(input, module, columns)

		// Build WHERE condition
		where := pg.RawBool(
			fmt.Sprintf(`"%s" = #val`, whereKey),
			pg.RawArgs{"#val": whereValue},
		)

		output, err := generator.db(module).Update(l, module.Table, module.PrimaryKey, realFields, mapInput, where)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorUpdate, nil)
			return
		}

		response.Response(l, c, output)

		action.AfterRequest(c)
	}
}

func (generator *Generator) actionDelete(module *BaseModule, action actions.DeleteModuleAction) func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		l, _ := icontext.GetLogger(ctx)
		role := actions.GetRoleFromContext(c)

		if hook := actions.ResolveRoleHook(module.RoleBeforeHook, role); hook != nil {
			if err := hook(c); err != nil {
				response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
				return
			}
		}
		defer func() {
			if hook := actions.ResolveRoleAfterHook(module.RoleAfterHook, role); hook != nil {
				hook(c)
			}
		}()

		err := action.BeforeRequest(c)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorDelete, nil)
			return
		}

		whereKey := c.Param("bykey")
		allowedKeys := make([]interface{}, 0, len(action.By))
		for _, col := range action.By {
			allowedKeys = append(allowedKeys, col.Name())
		}
		err = validation.In(allowedKeys...).Error(fmt.Sprintf(`allowed keys %v`, allowedKeys)).Validate(whereKey)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorDelete, []string{
				err.Error(),
			})
			return
		}

		whereValue := c.Param("value")
		if len(whereValue) == 0 {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorDelete, nil)
			return
		}

		// Build WHERE condition
		where := pg.RawBool(
			fmt.Sprintf(`"%s" = #val`, whereKey),
			pg.RawArgs{"#val": whereValue},
		)

		err = generator.db(module).Delete(l, module.Table, where)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, GeneratorErrorDelete, []string{
				err.Error(),
			})
			return
		}

		output := struct {
			Delete bool `json:"delete"`
		}{
			Delete: true,
		}
		response.Response(l, c, output)

		action.AfterRequest(c)
	}
}
