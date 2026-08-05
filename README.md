# Request Generator

CRUD REST API framework for Go + Gin + PostgreSQL (go-jet). Автоматически генерирует эндпоинты на основе декларативных описаний модулей с поддержкой ролей, валидации, мультиязычности и OpenAPI.

## Содержание

- [Архитектура](#архитектура)
- [Этапы создания модуля](#этапы-создания-модуля)
- [Системные таблицы](#системные-таблицы)
- [Обязательные поля таблиц](#обязательные-поля-таблиц)
- [Middleware и порядок регистрации](#middleware-и-порядок-регистрации)
- [Описание компонентов](#описание-компонентов)
- [API эндпоинты](#api-эндпоинты)

---

## Архитектура

```
packages/request-generator/
  generator.go           -- точка входа, регистрация роутов
  module.go              -- BaseModule (описание CRUD-модуля)
  actions/
    module_actions.go    -- ModuleAction interface, JoinType, ModuleActionJoin
    list_module_action.go
    add_module_action.go
    view_module_action.go
    update_module_action.go
    delete_module_action.go
    defrec_module_action.go
    role.go              -- Role type
    role_context.go      -- RoleContext, RoleWhere, RoleJoin, RoleHook, resolve-функции
    sort.go              -- SortOption
  fields/
    module_field.go      -- ModuleField, типы, валидация, CheckRules
  db/
    executor.go          -- DBExecutor interface, TranslationContext
    postgres_db.go       -- PostgreSQL реализация
  icontext/
    context.go           -- UserInfo, logger, request ID в context
  locale/                -- Lang type, мультиязычность
  response/              -- обёртки HTTP-ответов
  utils/                 -- ParseJson и утилиты
```

## Архитектурный стандарт: request-generator как источник UI-контракта

Request-generator должен позволять приложению отдавать не только CRUD-данные, но и расширяемый контракт для frontend renderer. Если клиентское приложение вынуждено знать конкретные ключи полей, роли, порядок блоков или бизнес-условия конкретного модуля, значит metadata в модуле описана неполно.

Базовая схема:

```
BaseModule
  ├── Fields
  │     ├── Type/FormType
  │     ├── Options/OptionsFunc
  │     ├── Check/DefaultFunc/Convert
  │     ├── Presentation  -> typed metadata визуального представления поля
  │     ├── Media         -> typed metadata одиночного media-поля
  │     └── Extra         -> legacy escape hatch
  │
  ├── ListModuleAction
  │     ├── Filter/Search/Sort/SortDefault
  │     ├── Where/Permission/Auth
  │     └── ExtraFunc -> legacy/dynamic app-specific metadata
  │
  ├── ViewModuleAction
  │     └── Extra -> legacy custom view/detail metadata
  │
  ├── DefrecModuleAction
  │     └── schema for add/edit forms
  │
  ├── Render
  │     └── typed UniversalRenderer page metadata
  │
  └── Navigation
        └── navigation/config metadata and page routes
```

### Ответственность модуля

Модуль должен описывать:

- права ролей и server-side ограничения;
- видимые/редактируемые поля;
- порядок полей, групп и секций;
- типы контролов и их варианты;
- server-side options и справочники;
- фильтры, сортировки и pagination;
- действия карточек/строк/detail view;
- условия видимости действий;
- ключи переводов;
- ключи иконок;
- typed metadata для UniversalRenderer через `BaseModule.Render`;
- typed metadata одиночных полей через `ModuleField.Presentation` и специализированные typed blocks вроде `ModuleField.Media`;
- legacy/application-specific metadata через `extra` только для старого кода. Новые возможности UniversalRenderer нужно добавлять в typed contract.

Frontend должен получать готовую схему и рендерить ее своими универсальными компонентами. Frontend не должен угадывать, что поле `status` нужно показать badge, что `category_ids` является multiselect, а `owner_id` нужно скрыть от определенной роли.

### Где описывать UI metadata

| Что нужно описать | Где описывать |
|---|---|
| Поле формы add/edit | Базово `ModuleField`; расширения через typed поля `Presentation`, `Media` и т.п. |
| Поле detail/list/card view | Базово `ModuleField`; расширения через typed поля `Presentation`, `Media` и т.п. |
| Полный список с фильтрами и карточками | `BaseModule.Render.List` |
| Универсальная form/edit page | `BaseModule.Render.Form` |
| Страница просмотра записи | `BaseModule.Render.Record` |
| Рабочий grid с create/update/delete/status | `BaseModule.Render.ResourceGrid` |
| Legacy/detail groups | `ViewModuleAction.Extra` |
| Навигация и page routes | `BaseModule.Navigation` |
| Динамический список колонок по роли | `ColumnsFunc` или `Fields`/`RoleContext` |
| Динамические фильтры по роли | `FilterFunc` |
| Права на строки | `Where`, `RoleWhere`, `BeforeAction`, `DataCheckRule` |
| Options из базы | `OptionsFunc` или `Extra.Defrec.options_url` |

Подробная спецификация UniversalRenderer: [docs/universal-renderer-contract.md](docs/universal-renderer-contract.md).

### Typed field metadata

Для новых UI-возможностей нельзя заставлять frontend угадывать поведение по имени поля и нельзя добавлять ad-hoc `extra`. Если поле требует явного renderer contract, используйте typed поля `ModuleField`.

Пример одиночного media-поля:

```go
{
    Column:   table.Profiles.Avatar,
    Title:    "profiles.fields.avatar",
    Type:     fields.ModuleFieldTypeString,
    FormType: fields.ModuleFieldFormTypeText,
    Presentation: &renderer.FieldPresentation{
        Renderer: renderer.RendererAvatar,
        Variant:  "avatar",
        Size:     renderer.MediaSizeThumb,
        Ratio:    renderer.MediaRatioSquare,
    },
    Media: &renderer.FieldMediaConfig{
        Item: &renderer.MediaGalleryItem{
            Kind:       renderer.MediaKindPhoto,
            Visibility: renderer.MediaVisibilityPublic,
            Usage:      renderer.MediaUsageAvatar,
        },
        Upload: &renderer.MediaUploadConfig{
            Title:        "settings.profile.avatar_title",
            LoadingTitle: "settings.profile.uploading",
            Accept:       "image/jpeg,image/png,image/webp",
            Multiple:     false,
        },
        Actions: &renderer.MediaGalleryActions{
            Upload: &renderer.Action{ID: "upload", Label: "settings.profile.upload_new", Type: renderer.ActionEmit},
            Remove: &renderer.Action{ID: "remove", Label: "settings.profile.remove_avatar", Type: renderer.ActionEmit},
        },
    },
}
```

В `defrec.fields[field]` и `view.item[field]` эти blocks приходят как `presentation` и `media`. Labels в media/action metadata переводятся request-generator по текущему языку. В `view` `media.item.src` может быть автоматически заполнен из `item[field].value`, если producer не указал `src`.

Для typed controls выбора в `defrec` и `view` используйте renderer key вместе
с обычными `Options`. `ModuleFieldOptions.Icon` передается в list filters,
`defrec` и `view`, а `Label` локализуется генератором. List filter не получает
`Presentation`: он использует базовый control по `form_type`. Специализированный
filter control потребует отдельного typed contract.

```go
{
    Column:   table.Items.Categories,
    Title:    "items.fields.categories",
    Type:     fields.ModuleFieldTypeArray,
    FormType: fields.ModuleFieldFormTypeMultiselect,
    Presentation: &renderer.FieldPresentation{
        Renderer: renderer.RendererChipSelect,
    },
    Options: []fields.ModuleFieldOptions{
        {Value: "example", Label: "items.options.example", Icon: "tag"},
    },
},
{
    Column:   table.Items.Status,
    Title:    "items.fields.status",
    Type:     fields.ModuleFieldTypeString,
    FormType: fields.ModuleFieldFormTypeSelect,
    Presentation: &renderer.FieldPresentation{
        Renderer: renderer.RendererPrimaryRadio,
    },
    Options: []fields.ModuleFieldOptions{
        {Value: "active", Label: "items.options.active", Icon: "check"},
    },
},
```

### FieldMatrix list

`field.matrix` раскладывает уже описанные typed поля формы. Для `list` не
нужны rows или cells: producer задает только порядок полей и число колонок.
Само поле сохраняет стандартные `type`, `form_type`, value, options и checks.

```go
renderer.FormSection{
    ID:       "preferences",
    Renderer: renderer.RendererFieldMatrix,
    Matrix: &renderer.FieldMatrix{
        Type:      renderer.FieldMatrixTypeList,
        Underline: "settings",
        List: &renderer.FieldMatrixList{
            Fields:  []string{"email_enabled", "push_enabled", "quiet_hours"},
            Columns: renderer.FieldMatrixColumnsTwo,
        },
    },
}
```

`Columns` - closed typed enum от `FieldMatrixColumnsOne` до
`FieldMatrixColumnsFour`. Для table используется отдельная typed структура с
heads, rows и cells; полный contract и правила локализации приведены в
[docs/universal-renderer-contract.md](docs/universal-renderer-contract.md).

### Extra.Defrec

`Extra.Defrec` является legacy-механизмом. Его можно использовать только для существующих модулей или временной совместимости, пока нужный UI contract еще не типизирован.

```go
{
    Column:   table.CatalogItems.CategoryIDs,
    Title:    "catalog_items.fields.categories",
    Type:     fields.ModuleFieldTypeArray,
    FormType: fields.ModuleFieldFormTypeMultiselect,
    OptionsFunc: categoryOptions,
    Extra: &fields.FieldExtra{
        Defrec: map[string]interface{}{
            "visual_kind": "select",
            "multiple":    true,
            "searchable":  true,
            "icon":        "tag",
            "section":     "general",
            "group":       "main",
            "order":       30,
            "layout":      "full",
            "placeholder": "ui.search",
        },
    },
}
```

Рекомендуемые ключи:

| Ключ | Назначение |
|---|---|
| `visual_kind` | Универсальный тип: `input`, `textarea`, `select`, `location`, `radio`, `switch`, `matrix`, `media`, `collection`. |
| `section` | ID секции страницы. |
| `group` | ID группы внутри секции. |
| `order` | Порядок поля. |
| `layout` | `full`, `grid`, `inline`, `compact`, `matrix`. |
| `icon` | Ключ иконки из реестра. |
| `hint` | Ключ перевода подсказки. |
| `placeholder` | Ключ перевода placeholder. |
| `multiple` | Множественный выбор. |
| `searchable` | Поиск внутри select. |
| `options_url` | Endpoint для server-driven options. |
| `options_params` | Параметры endpoint options. |
| `prefix` / `suffix` | Внутренний prefix/suffix поля. |
| `min` / `max` / `step` | Числовые ограничения. |

Нельзя добавлять поле так, чтобы frontend узнавал его по имени и вручную выбирал компонент.

### Extra.View

`Extra.View` описывает отображение значения в list/detail/card view.

```go
Extra: &fields.FieldExtra{
    View: map[string]interface{}{
        "display": map[string]interface{}{
            "type":   "badge",
            "tone":   "glass-cyan",
            "marker": false,
            "option": "status",
        },
    },
}
```

Типовые `display`:

- `text`;
- `badge`;
- `chips`;
- `boolean`;
- `code`;
- `json`;
- `masked`;
- `media`;
- `money`;
- `date`;
- `location`.

### Typed Render и страницы списков

Canonical UniversalRenderer metadata описывается через typed `BaseModule.Render`. Request-generator сам добавляет в response `renderer.name/version` и top-level page metadata: `list_page`, `form_page`, `record_page`, `resource_grid_page`.

`/api/config` также получает route discovery metadata в `navigation[].target.renderer` и `navigation[].target.page_type`, поэтому frontend может выбрать renderer до загрузки данных страницы.

Если page metadata зависит от роли, query или текущего пользователя, используйте typed `RenderFunc`. Он получает deep clone базового `renderer.Universal`, возвращает итоговый `renderer.Universal`, а request-generator валидирует результат перед ответом. Clone покрывает pointer structs, slices, maps и стандартные JSON-like значения внутри `interface{}` (`map[string]interface{}`, `[]interface{}`, `map[string]string`, `[]string` и т.п.); произвольные custom objects внутри `interface{}` остаются ответственностью producer module.

```go
Render: renderer.Universal{
    List: &renderer.ListPage{
        Title:      "catalog_items.list.title",
        Subtitle:   "catalog_items.list.subtitle",
        ShowHeader: ptrBool(false),
        Filters: &renderer.Filters{
            PrimaryPlacement: "topbar",
            Primary:          []string{"status", "category_id", "price"},
            Secondary:        []string{"created_at", "owner_id"},
            More:             []string{"rating", "tags"},
        },
        Grid: &renderer.Grid{Enabled: true, Mode: renderer.GridModeCards},
        Pagination: &renderer.Pagination{
            Renderer: "universal.pagination",
            Mode:     renderer.PaginationServer,
        },
        CardSchema: cardSchema,
        Context: map[string]interface{}{
            "can_create_request": canCreateRequest(c),
        },
    },
}
```

Фильтры в UI должны соответствовать server-side `Filter`/`FilterFunc`. Если фильтр есть в metadata, он обязан применяться на сервере.

### Действия и условия видимости

Действия карточек, строк и detail view описываются декларативно.

```go
renderer.Action{
    ID:         "message",
    Type:       renderer.ActionEmit,
    Icon:       "message",
    Variant:    renderer.ActionVariantSuccess,
    Appearance: renderer.ActionAppearanceOutline,
    VisibleIf: &renderer.Condition{
        All: []renderer.Condition{
            {Path: "relationship.allowed", Equals: true},
            {Path: "record.status", Equals: "active"},
            {Path: "record.owner_id", NotEmpty: ptrBool(true)},
        },
    },
}
```

Поддерживаемая форма условий:

- `path`: путь в `record`, `relationship` или `context`;
- `equals`: строгое равенство;
- `not_equals`: строгое неравенство;
- `in` / `not_in`: значение входит или не входит в список;
- `empty` / `not_empty`: проверка пустого или непустого значения;
- `truthy` / `falsy`: проверка truthy/falsy значения;
- `all`: все условия истинны;
- `any`: хотя бы одно условие истинно;
- `not`: только отрицание вложенного condition object.

Важно: `visible_if` и `hidden_if` управляют только отображением. Серверное действие обязательно должно проверять доступ через `Permission`, `Where`, `BeforeAction` или `DataCheckRule`.

### Collection manager metadata

`renderer.CollectionConfig` описывает универсальную коллекцию записей самостоятельного модуля. Контракт не содержит owner foreign key, target module/id, business-specific полей или endpoint-ов.

```go
Render: renderer.Universal{
    Form: &renderer.FormPage{
        Sections: []renderer.FormSection{
            {
                ID:       "services",
                Renderer: renderer.RendererCollectionManager,
                Collection: &renderer.CollectionConfig{
                    Module:     "related_records",
                    Relation:   "owner",
                    EditFields: []string{"amount", "note", "enabled"},
                    Item: &renderer.CollectionItem{
                        LabelField: "kind",
                        MetaFields: []string{"amount", "note"},
                    },
                    Buckets: []renderer.CollectionBucket{
                        {
                            ID:      "included",
                            Title:   "ui.included",
                            BlockID: "collection.included",
                            Predicate: &renderer.CollectionPredicate{
                                Field:    "amount",
                                Operator: renderer.CollectionPredicateEquals,
                                Value:    &renderer.TypedValue{Type: renderer.TypedValueNumber, Number: 0},
                            },
                            Defaults: []renderer.CollectionFieldDefaultValue{
                                {Field: "amount", Value: renderer.TypedValue{Type: renderer.TypedValueNumber, Number: 0}},
                            },
                        },
                    },
                },
            },
        },
    },
}
```

Если collection нужно ограничить текущей открытой записью, связь описывается в backend module, а renderer указывает только technical relation name:

```go
Relations: []module.ModuleRelation{
    {
        Name:         "owner",
        TargetModule: "records",
        SourceField:  table.RelatedRecords.RecordID,
        TargetField:  table.Records.ID,
        ScopeCheck: func(c *gin.Context, scope module.RelationScope) error {
            return checkActorCanUseRecord(c, scope.ID)
        },
    },
}
```

Frontend строит стандартные list/defrec/action endpoints из `collection.module`. Если `collection.relation` задан, frontend передаёт `scope[relation]` и `scope[id]` в query. Body add/update содержит только обычные поля модуля; relation source field подставляет generator.

### Resource grid metadata

Для страниц типа “управление сущностями” модуль описывает typed `BaseModule.Render.ResourceGrid`. `ExtraFunc` и `extra.resource_grid_page` остаются legacy compatibility, но не являются canonical способом для новых модулей.

Для одного list route `Render.List` и `Render.ResourceGrid` взаимоисключающие. Если нужны обе страницы, создавайте отдельный route/module; request-generator валидирует это при старте.

```go
Render: renderer.Universal{
    ResourceGrid: &renderer.ResourceGridPage{
        Endpoint: "/catalog_items",
        List: &renderer.ResourceGridListConfig{
            Size:    100,
            Filters: map[string]interface{}{"scope": "owned"},
        },
        Create: &renderer.Action{
            Type:         renderer.ActionAPI,
            API:          &renderer.APIAction{Method: "put", Endpoint: "/catalog_items"},
            AfterSuccess: &renderer.ActionResult{Reload: "list"},
        },
        Delete: &renderer.Action{
            Type: renderer.ActionAPI,
            API: &renderer.APIAction{
                Method:   "delete",
                Endpoint: "/catalog_items/delete/id/:id",
                Params:   map[string]string{"id": "record.id"},
            },
            Confirm:      &renderer.Confirm{Title: "ui.confirm_delete"},
            AfterSuccess: &renderer.ActionResult{Reload: "list"},
        },
        Update: &renderer.Action{
            Type: renderer.ActionAPI,
            API: &renderer.APIAction{
                Method:   "post",
                Endpoint: "/catalog_items/id/:id",
                Params:   map[string]string{"id": "record.id"},
            },
            AfterSuccess: &renderer.ActionResult{Reload: "record"},
        },
        Card: &renderer.CardSchema{
            Type:           "catalog_item",
            SurfaceVariant: renderer.SurfaceDefault,
            BadgeSize:      renderer.SizeSM,
            ActionSize:     renderer.SizeMD,
        },
        Status: &renderer.ResourceGridStatusConfig{
            VerifyField:         "review_status",
            ActiveField:         "status",
            VerifiedValue:       "approved",
            PendingValue:        "pending",
            InactiveActionValue: "inactive",
            ActiveActionValue:   "active",
        },
    }
}
```

Для такого режима обязательно добавляйте `Where`, чтобы пользователь не мог запросить чужие сущности через тот же endpoint.

### Контракт универсального рендера

Документация по универсальному рендеру разделена на три документа:

- [docs/universal-renderer-contract.md](docs/universal-renderer-contract.md) - сама исполняемая спецификация `UniversalRenderer`.
- [docs/specification-process.md](docs/specification-process.md) - процесс предложения, принятия, статусов и версионирования спецификаций.
- [docs/specification-goals.md](docs/specification-goals.md) - общие цели, одинаковые для всех спецификаций.

README содержит только обзор request-generator. Полная схема typed `BaseModule.Render`, `renderer`, `field.extra`, `list_page`, `resource_grid_page`, `form_page`, `record_page`, legacy `extra`, actions, conditions, filters и renderer registry описывается в спецификации.

### Правила совместимости с универсальным frontend

PR в модуле считается неготовым, если:

- новое поле требует `if field.key == ...` на frontend;
- options захардкожены во frontend, хотя зависят от базы или роли;
- UniversalRenderer metadata добавляется через legacy `ExtraFunc`, хотя уже есть typed `BaseModule.Render`;
- действие скрывается только на frontend, но API все равно разрешает его выполнить;
- фильтр есть в UI, но не применяется в `Filter`/`Where`;
- переводимый текст отдается literal-строкой вместо ключа;
- иконка вшита в клиент вместо передачи стабильного ключа/metadata;
- техническая таблица превращена в menu entry без пользовательского сценария.

---

## Этапы создания модуля

### Этап 1. Создание таблицы в базе данных

Создайте SQL-миграцию с таблицей. Обязательные требования:
- Первичный ключ `id BIGSERIAL PRIMARY KEY`
- Все внешние ключи — `BIGINT ... REFERENCES <table>(id)`
- CHECK-ограничения для enum-полей
- Индексы на часто запрашиваемые поля и FK

```sql
CREATE TABLE courses (
    id              BIGSERIAL PRIMARY KEY,
    category_id     BIGINT NOT NULL REFERENCES categories(id),
    price           DECIMAL(10,2) NOT NULL DEFAULT 0,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_courses_category_id ON courses (category_id);
```

Если у таблицы есть **переводимые поля** (title, description, name и т.д.) — НЕ создавайте колонки в основной таблице. Переводы хранятся в системной таблице `translations` (см. [Системные таблицы](#системные-таблицы)).

### Этап 2. Генерация Jet-моделей

После применения миграции перегенерируйте Jet-код:

```bash
jet -dsn=<DATABASE_URL> -schema=public -path=./generated
```

Это создаст:
- `generated/muta_alim/public/table/<table_name>.go` — Jet table + columns
- `generated/muta_alim/public/model/<table_name>.go` — Go struct модели

### Этап 3. Создание sentinel-колонок для переводимых полей

Для каждого переводимого поля создайте виртуальную колонку (не существует в БД, нужна для маппинга):

```go
var coursesTitle = pg.StringColumn("title")
var coursesDescription = pg.StringColumn("description")
```

### Этап 4. Описание модуля (BaseModule)

Создайте файл `admin/modules/<name>.go` с фабричной функцией:

```go
func NewCoursesModule() *module.BaseModule {
    return &module.BaseModule{
        Name:       "courses",         // имя роута (GET /admin/courses)
        Label:      "courses.label",   // ключ перевода для UI
        Table:      table.Courses,     // Jet-таблица
        PrimaryKey: table.Courses.ID,  // первичный ключ
        Path:       "",                // префикс пути (обычно пустой)
        Fields:     []fields.ModuleField{...},
        Defrec:     actions.DefrecModuleAction{Label: "courses.label"},
        Actions:    []actions.ModuleAction{...},
    }
}
```

**Обязательные поля BaseModule:**

| Поле         | Тип                          | Описание                          |
|--------------|------------------------------|-----------------------------------|
| `Name`       | `string`                     | Имя модуля, используется в URL    |
| `Label`      | `string`                     | Ключ i18n для отображения         |
| `Table`      | `pg.Table`                   | Jet-таблица                       |
| `PrimaryKey` | `pg.Column`                  | Колонка первичного ключа          |
| `Fields`     | `[]fields.ModuleField`       | Описание полей модуля             |
| `Defrec`     | `actions.DefrecModuleAction` | Описание дефолтной записи         |
| `Actions`    | `[]actions.ModuleAction`     | Набор CRUD-действий               |

**Опциональные поля BaseModule:**

| Поле             | Тип                        | Описание                                   |
|------------------|----------------------------|---------------------------------------------|
| `Path`           | `string`                   | Доп. префикс URL                           |
| `Labels`         | `map[string]string`        | Мультиязычные метки                         |
| `EntityName`     | `string`                   | Имя сущности для translations (по умолч. = имя таблицы) |
| `RoleWhere`      | `[]actions.RoleWhere`      | WHERE-условия по ролям                      |
| `RoleJoin`       | `[]actions.RoleJoin`       | JOIN-ы по ролям                             |
| `RoleBeforeHook` | `[]actions.RoleHook`       | Хуки до обработки по ролям                  |
| `RoleAfterHook`  | `[]actions.RoleAfterHook`  | Хуки после обработки по ролям               |
| `Navigation`     | `[]module.NavigationEntry` | Пункты навигации и frontend page routes для config endpoint |

#### NavigationEntry

`NavigationEntry` описывает пункт навигации и связанный с ним frontend route. Навигация строится из `Navigation` всех модулей, фильтруется по `Roles` пункта и по permission указанного `ActionName`.

```go
Navigation: []module.NavigationEntry{
    {
        ActionName: "list",         // имя действия модуля (list/view/…)
        ID:         "catalog.list", // стабильный id пункта, опционально
        Path:       "/catalog",     // frontend route для page target
        Title:      "menu.catalog", // ключ i18n для заголовка пункта
        Icon:       "catalog",       // иконка (передаётся клиенту as-is)
        Show:       true,
        Order:      1,
        Group:      "main",        // группа блока в левом меню
        Roles:      []actions.Role{"admin", "editor"},
        Target:     module.NavigationTarget{Type: "page"},
    },
},
```

| Поле          | Тип                        | Описание                                              |
|---------------|----------------------------|-------------------------------------------------------|
| `ActionName`  | `string`                   | Имя действия (`list`, `view`, …)                      |
| `Title`       | `string`                   | Ключ i18n для отображаемого заголовка                 |
| `Icon`        | `string`                   | Имя иконки (передаётся клиенту)                       |
| `Show`        | `bool`                     | Показывать пункт (`false` — скрыт, но в features есть) |
| `Order`       | `int`                      | Порядок внутри группы                                 |
| `Group`       | `string`                   | Ключ группы для группировки на клиенте                |
| `Target`      | `module.NavigationTarget`  | Явное поведение пункта навигации                      |
| `Roles`       | `[]actions.Role`           | Роли, которым доступен пункт (пустой = все)           |
| `Query`       | `map[string]interface{}`   | Доп. query-параметры для клиента                      |
| `Data`        | `map[string]interface{}`   | Произвольные данные для клиента                       |

`Target.Type` определяет поведение пункта:

| Type | Обязательные поля | Назначение |
|---|---|---|
| `page` | `Path` | Переход на frontend route. Renderer/query/children встраиваются в `navigation[].target`. |
| `modal` | `Name` | Открытие клиентского popup/modal. |
| `client_action` | `Name` | Выполнение клиентского действия, например `chat.open`. |
| `external` | `Name` или `Data.url` | Переход на внешний URL по договоренности клиента. |

#### WidgetConfig

Глобальные виджеты описываются на конкретном действии модуля через `Widget`. Config endpoint автоматически возвращает такие действия в `widgets[]`; query строится из действия, на котором висит виджет.

```go
Actions: []actions.ModuleAction{
    actions.ListModuleAction{
        Label:      "profile_menu",
        Permission: []actions.Role{actions.RoleAll},
        Auth:       true,
        Widget: &actions.WidgetConfig{
            ID:        "profile-menu",
            Type:      "module",
            Placement: "topbar",
            Order:     10,
        },
    },
},
```

`DefrecModuleAction` тоже может быть widget source, если нужно автоматически получить форму добавления в `/api/config.widgets`.

### Этап 5. Описание полей (ModuleField)

Каждое поле описывает одну колонку/виртуальное поле модуля:

```go
{
    Column:       table.Courses.Price,
    Title:        "price",
    Type:         fields.ModuleFieldTypeFloat,
    FormType:     fields.ModuleFieldFormTypeNumber,
    Check: []fields.CheckRules{
        fields.RequiredRule(table.Courses.Price, []fields.Scenario{fields.ScenarioAdd}),
    },
}
```

**Для переводимых полей:**

```go
{
    Column:       coursesTitle,          // sentinel-колонка
    FieldName:    "title",              // логическое имя поля
    Translatable: true,                 // флаг переводимости
    Title:        "title",
    Type:         fields.ModuleFieldTypeObject,
    FormType:     fields.ModuleFieldFormTypeMap,
}
```

#### Типы полей (ModuleFieldType)

| Константа                  | Значение   | Описание                   |
|----------------------------|------------|----------------------------|
| `ModuleFieldTypeString`    | `"string"` | Строка                     |
| `ModuleFieldTypeInt`       | `"int"`    | Целое число                |
| `ModuleFieldTypeFloat`     | `"float"`  | Дробное число              |
| `ModuleFieldTypeArray`     | `"array"`  | Массив                     |

For `ModuleFieldTypeArray` fields, add/update accepts JSON arrays and converts them to PostgreSQL arrays before executing SQL. List filters for array columns use PostgreSQL overlap syntax with array literals, for example `filter[tags]={global,featured}`.
| `ModuleFieldTypeObject`    | `"object"` | Объект (используется для translatable) |

#### Типы форм (ModuleFieldFormType)

| Константа                        | Значение        | Описание                     |
|----------------------------------|-----------------|------------------------------|
| `ModuleFieldFormTypeText`        | `"text"`        | Текстовое поле               |
| `ModuleFieldFormTypeNumber`      | `"number"`      | Числовое поле                |
| `ModuleFieldFormTypeTextArea`    | `"textarea"`    | Многострочное поле           |
| `ModuleFieldFormTypeSelect`      | `"select"`      | Выпадающий список            |
| `ModuleFieldFormTypeCheckBox`    | `"checkbox"`    | Чекбокс                     |
| `ModuleFieldFormTypeMultiselect` | `"multiselect"` | Множественный выбор          |
| `ModuleFieldFormTypeMap`         | `"map"`         | Карта ключ-значение (i18n)   |
| `ModuleFieldFormTypeHidden`      | `"hidden"`      | Скрытое поле                 |
| `ModuleFieldFormTypeOnlyView`    | `"onlyview"`    | Только чтение                |

#### Правила валидации (CheckRules)

| Функция                      | Описание                                         |
|------------------------------|--------------------------------------------------|
| `fields.RequiredRule(col, scenarios)` | Поле обязательно для заданных сценариев    |
| `fields.InRule(col, values, scenarios)` | Значение из допустимого списка            |
| `fields.LenRule(col, min, max, scenarios)` | Ограничение длины строки              |
| `fields.UrlRule(col, scenarios)`    | Валидация URL                                |

Сценарии: `fields.ScenarioAdd`, `fields.ScenarioUpdate`.

#### Кросс-полевая и DB-валидация (DataCheckRule)

Когда правилу нужен доступ к другим полям запроса или к базе данных, используйте `DataCheckRule`.
Генератор автоматически обнаруживает реализацию интерфейса и вызывает `ValidateData` вместо `Validate`,
передавая контекст запроса, `*sql.DB` и полный набор входящих данных.

```go
type DataCheckRule interface {
    CheckRules                      // обратная совместимость
    ValidateData(c *gin.Context, db *sql.DB, data map[string]interface{}, lang string) error
}
```

**Быстрый вариант через `DataRule()`:**

```go
{
    Column: table.Lists.WhomAdd,
    Check: []fields.CheckRules{
        fields.RequiredRule(table.Lists.WhomAdd, []fields.Scenario{fields.ScenarioAdd}),
        fields.DataRule(func(c *gin.Context, db *sql.DB, data map[string]interface{}, lang string) error {
            whoID, _ := data["who_add"].(float64)
            whomID, _ := data["whom_add"].(float64)
            tag, _ := data["tag"].(string)
            var exists int
            _ = db.QueryRowContext(c.Request.Context(),
                `SELECT 1 FROM lists WHERE who_add=$1 AND whom_add=$2 AND tag=$3 LIMIT 1`,
                int64(whoID), int64(whomID), tag,
            ).Scan(&exists)
            if exists == 1 {
                return fmt.Errorf("already_exists")
            }
            return nil
        }, []fields.Scenario{fields.ScenarioAdd}),
    },
}
```

**Собственный тип** (когда нужна структура с полями или переиспользование):

```go
type uniqueListRule struct {
    scenarios []fields.Scenario
}

func (r uniqueListRule) Validate(_ interface{}, _ string) error { return nil }
func (r uniqueListRule) GetScenarios() []fields.Scenario       { return r.scenarios }
func (r uniqueListRule) ValidateData(c *gin.Context, db *sql.DB, data map[string]interface{}, lang string) error {
    // ... логика с доступом к c, db, data
    return nil
}
```

> **Важно:** В типе, реализующем `DataCheckRule`, метод `Validate` должен возвращать `nil` —
> генератор вызывает только `ValidateData`. Метод нужен лишь для удовлетворения интерфейса `CheckRules`.

### Этап 6. Описание действий (Actions)

Каждый модуль должен содержать массив `Actions` с нужными CRUD-операциями.

#### ListModuleAction

```go
actions.ListModuleAction{
    Label:       "courses.list",
    Auth:        true,
    Permission:  []actions.Role{"admin", "moderator"},
    Columns:     []pg.Column{table.Courses.ID, ...},
    Filter:      []pg.Column{table.Courses.Status},
    FilterFunc:  func(c *gin.Context) []pg.Column { ... }, // динамические фильтры по ролям
    Search:      []pg.Column{coursesTitle},
    Sort:        []pg.Column{table.Courses.ID, table.Courses.Price},
    SortDefault: table.Courses.ID,
    Size:        50,
    Maxsize:     1000,
    Join:        []actions.ModuleActionJoin{...},
    Where:       func(c *gin.Context) pg.BoolExpression { ... },
    ExtraFunc:   func(c *gin.Context) interface{} { ... }, // legacy/dynamic app-specific extra
}
```

**Обязательные поля:** `Label`, `Columns`.

**`ExtraFunc`** вызывается при каждом запросе; результат включается в ответ как поле `extra`. Используется для legacy/dynamic app-specific данных, зависящих от роли или параметров запроса. UniversalRenderer page metadata для новых модулей описывайте через `BaseModule.Render`.

```go
ExtraFunc: func(c *gin.Context) interface{} {
    return map[string]interface{}{
        "pills": []map[string]interface{}{
            {"label": "All"},
            {"label": "Verified", "key": "verify_status", "val": "verified"},
        },
    }
},
```

**`FilterFunc`** заменяет/дополняет статический `Filter` — используется когда набор доступных фильтров зависит от роли:

```go
FilterFunc: func(c *gin.Context) []pg.Column {
    user, _ := icontext.GetUser(c.Request.Context())
    if user.Role == "admin" {
        return []pg.Column{table.Courses.Status, table.Courses.UserID}
    }
    return []pg.Column{table.Courses.Status}
},
```

#### AddModuleAction

```go
actions.AddModuleAction{
    Label:   "courses.add",
    Auth:    true,
    Columns: []pg.Column{...},  // колонки, которые принимаются при создании
}
```

**Обязательные поля:** `Label`, `Columns`.

#### ViewModuleAction

```go
actions.ViewModuleAction{
    Label:   "courses.view",
    Auth:    true,
    Columns: []pg.Column{...},          // отображаемые колонки
    By:      []pg.Column{table.Courses.ID},  // по каким ключам можно получить запись
}
```

**Обязательные поля:** `Label`, `Columns`, `By`.

#### UpdateModuleAction

```go
actions.UpdateModuleAction{
    Label:   "courses.update",
    Auth:    true,
    Columns: []pg.Column{...},
    By:      []pg.Column{table.Courses.ID},
    Where:   func(c *gin.Context) pg.BoolExpression { ... }, // дополнительное WHERE-условие
}
```

**Обязательные поля:** `Label`, `Columns`, `By`.

Опция `ViewAfterUpdate *bool` (по умолч. `true`) — после обновления возвращает полный View-ответ, если у модуля есть ViewAction.

**`Where`** — дополнительное WHERE-условие для UPDATE. Если ни одна строка не совпала — возвращается 404. Возврат `nil` снимает ограничение:

```go
Where: func(c *gin.Context) pg.BoolExpression {
    // вернуть условие или nil
},
```

#### DeleteModuleAction

```go
actions.DeleteModuleAction{
    Label: "courses.delete",
    Auth:  true,
    By:    []pg.Column{table.Courses.ID},
    Where: func(c *gin.Context) pg.BoolExpression { ... }, // дополнительное WHERE-условие
}
```

**Обязательные поля:** `Label`, `By`.

**`Where`** — дополнительное WHERE-условие для DELETE. Если ни одна строка не совпала — возвращается 404.

#### DefrecModuleAction

Описание дефолтной записи для формы создания. Всегда указывается в `BaseModule.Defrec`:

```go
Defrec: actions.DefrecModuleAction{Label: "courses.label"}
```

### Этап 7. Общие свойства Actions

Все действия поддерживают:

| Поле           | Тип                                       | Описание                                   |
|----------------|-------------------------------------------|---------------------------------------------|
| `Label`        | `string`                                  | Ключ перевода                              |
| `Labels`       | `map[string]string`                       | Мультиязычные метки                        |
| `Auth`         | `bool`                                    | Требовать авторизацию                      |
| `Permission`   | `[]actions.Role`                          | Допустимые роли (пустой = все авторизованные) |
| `BeforeAction` | `func(c *gin.Context) error`              | Хук перед обработкой                       |
| `AfterAction`  | `func(c *gin.Context)`                    | Хук после обработки                        |

#### BeforeAction — собственный ответ

Если `BeforeAction` записывает ответ самостоятельно (например, возвращает 400 с деталями ошибки), генератор проверяет `c.Writer.Written()` и **не пишет второй ответ**. Это предотвращает `superfluous response.WriteHeader call` панику:

```go
BeforeAction: func(c *gin.Context) error {
    if !isValid(c) {
        c.JSON(http.StatusBadRequest, gin.H{"error": "custom validation failed"})
        return fmt.Errorf("invalid")  // возвращаем ошибку — генератор остановится,
                                       // но не перезапишет уже отправленный ответ
    }
    return nil
},
```
| `Columns`      | `[]pg.Column`                             | Статический список колонок                 |
| `ColumnsFunc`  | `func(c *gin.Context) []pg.Column`        | Динамический список колонок                |
| `Fields`       | `[]actions.RoleContext`                    | Колонки по ролям (приоритет над Columns)   |

### Этап 8. Регистрация модуля в приложении

Добавьте модуль в массив `allModules` в `main.go`:

```go
allModules := []*module.BaseModule{
    modules.NewUsersModule(),
    modules.NewCategoriesModule(),
    modules.NewCoursesModule(),  // <-- новый модуль
    // ...
}
```

### Этап 9. Добавление переводов (опционально)

Если модуль имеет метки для UI, добавьте ключи в файлы переводов:

`translations/en.json`:
```json
{
    "courses.label": "Courses",
    "courses.list": "Course List",
    "courses.add": "Add Course",
    "courses.view": "View Course",
    "courses.update": "Edit Course",
    "courses.delete": "Delete Course"
}
```

`translations/ar.json`:
```json
{
    "courses.label": "الدورات",
    "courses.list": "قائمة الدورات"
}
```

### Этап 10. Запуск и проверка

После запуска приложения генератор автоматически:
1. Создаёт все CRUD-эндпоинты для модуля
2. Регистрирует модуль в `/admin/api/features`
3. Добавляет в OpenAPI-спеку (если `EnableOpenAPI = true`)

### Этап 11. Чеклист API-driven готовности

Перед PR проверьте:

- `ListModuleAction.Where` ограничивает строки по роли, владельцу и контексту.
- `UpdateModuleAction.Where` и `DeleteModuleAction.Where` не позволяют менять чужие записи.
- `AddModuleAction.Columns` содержит только поля, которые клиент имеет право прислать.
- Системные поля задаются через `DefaultFunc`, а не принимаются от клиента.
- Все options приходят из `Options`, `OptionsFunc` или `options_url`.
- Все поля формы имеют `Title` как ключ перевода.
- Новые UI-возможности описаны typed metadata (`Presentation`, `Media` и т.п.), а не ad-hoc `Extra`.
- `Extra.Defrec` и `Extra.View` используются только для legacy-совместимости.
- UniversalRenderer page metadata описана через typed `BaseModule.Render`, а не через legacy `ExtraFunc`.
- Действия, зависящие от состояния записи, описаны через `visible_if`/`hidden_if`.
- Все ограничения из `visible_if` продублированы серверной проверкой.
- Новые фильтры реально работают на сервере.
- Новые metadata покрыты targeted tests.

---

## Системные таблицы

Эти таблицы **обязательны** для работы фреймворка и НЕ являются CRUD-модулями:

### 1. `translations` — мультиязычный контент

Централизованное хранилище переводов для всех сущностей.

```sql
CREATE TABLE translations (
    id         BIGSERIAL PRIMARY KEY,
    entity     VARCHAR(100) NOT NULL,   -- имя таблицы/сущности ("courses", "categories")
    entity_id  BIGINT       NOT NULL,   -- ID записи в основной таблице
    field      VARCHAR(100) NOT NULL,   -- имя поля ("title", "description", "name")
    lang       VARCHAR(10)  NOT NULL,   -- код языка ("en", "ar")
    value      TEXT         NOT NULL DEFAULT '',
    UNIQUE (entity, entity_id, field, lang)
);
CREATE INDEX idx_translations_entity_lookup ON translations (entity, entity_id);
```

**Как работает:** При `Add`/`Update` записи с переводимыми полями, генератор автоматически выполняет INSERT/UPSERT в `translations`. При `List`/`View` — формирует подзапрос с `json_object_agg()` для сборки JSON-объекта переводов.

**Формат запроса:**
```json
{
    "title": {"en": "Course Title", "ar": "عنوان الدورة"},
    "slug": "course-slug"
}
```

**Формат ответа:**
```json
{
    "id": 1,
    "title": {"en": "Course Title", "ar": "عنوان الدورة"},
    "slug": "course-slug"
}
```

### 2. `sessions` — авторизация

Хранит токены аутентификации для AuthMiddleware.

```sql
CREATE TABLE sessions (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL
);
CREATE INDEX idx_sessions_token ON sessions (token);
CREATE INDEX idx_sessions_user_id ON sessions (user_id);
```

**Как работает:** AuthMiddleware выполняет запрос:
```sql
SELECT s.user_id, u.role
FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.token = $1 AND s.expires_at > NOW()
```

### 3. `users` — пользователи (системная часть)

Таблица `users` обязательна для работы авторизации. Минимально необходимые колонки:

| Колонка         | Тип           | Обязательна | Описание                       |
|-----------------|---------------|-------------|--------------------------------|
| `id`            | `BIGSERIAL`   | да          | PK, связь с sessions           |
| `role`          | `VARCHAR(20)` | да          | Роль для PermissionMiddleware  |
| `email`         | `VARCHAR(255)`| да*         | Для Login handler              |
| `password_hash` | `VARCHAR(255)`| да*         | Для Login handler              |

*\* Если используется встроенный Login handler.*

---

## Обязательные поля таблиц

### При создании новой таблицы (миграция)

| Требование                | Описание                                              |
|---------------------------|-------------------------------------------------------|
| `id BIGSERIAL PRIMARY KEY`| Обязательный первичный ключ                           |
| `NOT NULL` на FK          | Внешние ключи должны быть NOT NULL (если связь обязательна) |
| `REFERENCES` на FK        | Все FK с ссылкой на родительскую таблицу               |
| `ON DELETE CASCADE`       | На дочерних таблицах (videos, test_questions и т.д.)   |
| `DEFAULT` значения        | Для timestamps (`NOW()`), статусов, счётчиков          |
| `CHECK` для enum          | Ограничения допустимых значений                        |
| Индексы                   | На FK-колонки и часто фильтруемые поля                 |

### При создании ModuleField

| Поле              | Обязательное | Описание                                          |
|-------------------|-------------|---------------------------------------------------|
| `Column`          | да          | Jet-колонка (реальная или sentinel)               |
| `Title`           | да          | Ключ перевода заголовка                           |
| `Type`            | да          | Тип данных (string/int/float/array/object)        |
| `FormType`        | да          | Тип формы (text/number/select/map/...)            |
| `Group`           | нет         | Ключ группы фильтров (напр. `"breast"`, `"options"`) |
| `Order`           | нет         | Порядок внутри группы фильтров                    |
| `FilterCondition` | нет         | `func(c *gin.Context) bool` — показывать ли поле в фильтрах |
| `Convert`         | нет         | `func(c *gin.Context, value interface{}) (interface{}, error)` — преобразование входящего значения |
| `DefaultFunc`     | нет         | `func(c *gin.Context) interface{}` — значение по умолчанию; для полей вне `action.Columns` вызывается всегда (защита от spoofing) |

`Group` и `Order` экспортируются в JSON при `addFilters=true`. `FilterCondition` позволяет скрывать поля из фильтров в зависимости от роли или контекста запроса.

#### Convert и DefaultFunc

**`Convert`** — преобразование входящего (и/или исходящего) значения. Принимает `*gin.Context`, поэтому может учитывать роль пользователя:

```go
{
    Column:  table.Profiles.Status,
    Convert: func(c *gin.Context, value interface{}) (interface{}, error) {
        // например, нормализация строки перед сохранением
        if s, ok := value.(string); ok {
            return strings.ToLower(s), nil
        }
        return value, nil
    },
}
```

**`DefaultFunc`** — серверное значение по умолчанию. Критичное отличие от просто дефолта в БД: если поле не входит в `action.Columns`, генератор **всегда** вызывает `DefaultFunc(c)` и подставляет его значение, даже если клиент передал это поле в теле запроса. Это предотвращает подмену системных полей:

```go
{
    Column:      table.Lists.WhoAdd,
    DefaultFunc: func(c *gin.Context) interface{} {
        user, _ := icontext.GetUser(c.Request.Context())
        return user.ID  // клиент не может переопределить
    },
}
```

**Дополнительно для переводимых полей:**

| Поле           | Обязательное | Описание                                  |
|----------------|-------------|-------------------------------------------|
| `FieldName`    | да          | Логическое имя поля ("title", "name")     |
| `Translatable` | да          | `true`                                    |
| `Type`         | да          | `ModuleFieldTypeObject`                   |
| `FormType`     | да          | `ModuleFieldFormTypeMap`                  |

### При создании Actions

**ListModuleAction:**

| Поле      | Обязательное | Описание                       |
|-----------|-------------|--------------------------------|
| `Label`   | да          | Ключ перевода                  |
| `Columns` | да          | Список отображаемых колонок    |

**AddModuleAction:**

| Поле      | Обязательное | Описание                       |
|-----------|-------------|--------------------------------|
| `Label`   | да          | Ключ перевода                  |
| `Columns` | да          | Список принимаемых колонок     |

**ViewModuleAction:**

| Поле      | Обязательное | Описание                       |
|-----------|-------------|--------------------------------|
| `Label`   | да          | Ключ перевода                  |
| `Columns` | да          | Список отображаемых колонок    |
| `By`      | да          | Колонки для поиска записи      |

**UpdateModuleAction:**

| Поле      | Обязательное | Описание                       |
|-----------|-------------|--------------------------------|
| `Label`   | да          | Ключ перевода                  |
| `Columns` | да          | Список обновляемых колонок     |
| `By`      | да          | Колонки для идентификации      |

**DeleteModuleAction:**

| Поле      | Обязательное | Описание                       |
|-----------|-------------|--------------------------------|
| `Label`   | да          | Ключ перевода                  |
| `By`      | да          | Колонки для идентификации      |

---

## Middleware и порядок регистрации

### Инициализация Generator

```go
generator := module.NewGenerator(
    dbExecutor,                        // func(*BaseModule) db.DBExecutor
    routerGroup,                       // gin.RouterGroup
    allModules,                        // []*BaseModule
    middleware.PermissionMiddleware,    // проверка ролей
    middleware.NewAuthMiddleware(db),   // аутентификация
)
```

**Порядок аргументов критичен.** Сигнатура `NewGenerator`:

```go
func NewGenerator(
    db                   func(module *BaseModule) db.DBExecutor,
    group                gin.RouterGroup,
    modules              []*BaseModule,
    permissionMiddleware func(action ModuleAction, permissions []Role) gin.HandlerFunc,
    authMiddleware       func(action ModuleAction) gin.HandlerFunc,
) *Generator
```

### Порядок выполнения Middleware для каждого запроса

```
Запрос -> Gin Router
  |
  +--> [1] AuthMiddleware (если action.Auth == true)
  |     |-- Извлекает Bearer token из заголовка Authorization
  |     |-- Ищет активную сессию в таблице sessions
  |     |-- Устанавливает UserInfo (ID, Role) в context
  |     +-- Abort 401 если токен невалиден/истёк
  |
  +--> [2] PermissionMiddleware (если len(action.Permission) > 0)
  |     |-- Получает роль из context
  |     |-- Проверяет входит ли роль в action.Permission
  |     |-- Роль "admin" всегда имеет доступ
  |     +-- Abort 403 если нет доступа
  |
  +--> [3] RoleBeforeHook (если задан в module.RoleBeforeHook)
  |     +-- Произвольная проверка по роли перед обработкой
  |
  +--> [4] Action.BeforeRequest (BeforeAction хук конкретного действия)
  |
  +--> [5] Обработка запроса (List/Add/View/Update/Delete)
  |
  +--> [6] Action.AfterRequest (AfterAction хук конкретного действия)
  |
  +--> [7] RoleAfterHook (если задан в module.RoleAfterHook)
```

### Регистрация middleware по компонентам

Middleware применяется **на уровне действия**, а не модуля. Для каждого action генератор создаёт отдельную группу роутов:

```go
// Пример из generator.go для ListAction:
listGroup := generator.group.Group(module.Path)
if listAction.Auth {
    listGroup.Use(generator.AuthMiddleware(listAction))    // [1]
}
if len(listAction.Permission) > 0 {
    listGroup.Use(generator.PermissionMiddleware(listAction, listAction.Permission))  // [2]
}
listGroup.GET(module.Name, generator.actionList(module, listAction))
```

Это значит что каждое действие может иметь **свою комбинацию middleware**:

| Сценарий                          | Auth | Permission | Результат                       |
|-----------------------------------|------|------------|---------------------------------|
| `Auth: false`                     | нет  | нет        | Публичный эндпоинт              |
| `Auth: true`                      | да   | нет        | Любой авторизованный            |
| `Auth: true, Permission: ["admin"]`| да  | да         | Только admin                    |
| `Auth: true, Permission: ["admin", "moderator"]` | да | да | admin или moderator   |

**Если `Auth: false` и `Permission` задан** — произойдёт паника при инициализации (нет `AuthMiddleware` для получения роли).

### Ролевая система (Role-Based Access)

Роли задаются как `actions.Role` (alias `string`). Специальное значение `actions.RoleAll` ("all") — применяется ко всем ролям.

#### RoleContext (колонки по ролям)

```go
actions.ListModuleAction{
    Fields: []actions.RoleContext{
        {Role: "admin", Columns: []pg.Column{col1, col2, col3}},
        {Role: "user",  Columns: []pg.Column{col1, col2}},
        {Role: actions.RoleAll, Columns: []pg.Column{col1}},  // fallback
    },
}
```

Приоритет: точное совпадение роли > `RoleAll` > `Columns`/`ColumnsFunc`.

#### RoleWhere (фильтрация по ролям)

```go
RoleWhere: []actions.RoleWhere{
    {
        Role: "user",
        Where: func(c *gin.Context) pg.BoolExpression {
            user, _ := icontext.GetUser(c.Request.Context())
            return table.Courses.UserID.EQ(pg.Int(user.ID))
        },
    },
},
```

#### RoleJoin (JOIN по ролям)

```go
RoleJoin: []actions.RoleJoin{
    {
        Role: "user",
        Join: []actions.ModuleActionJoin{
            actions.NewJoin(table.Enrollments, actions.JoinTypeInner, onCondition, columns, "enrollments"),
        },
    },
},
```

#### RoleHook / RoleAfterHook (хуки по ролям)

```go
RoleBeforeHook: []actions.RoleHook{
    {
        Role: "user",
        Hook: func(c *gin.Context) error {
            // проверка перед обработкой
            return nil
        },
    },
},
RoleAfterHook: []actions.RoleAfterHook{
    {
        Role: "admin",
        Hook: func(c *gin.Context) {
            // действие после обработки
        },
    },
},
```

#### RoleCheck / RoleOptions (валидация и опции по ролям)

На уровне `ModuleField`:

```go
{
    Column: table.Courses.Status,
    RoleCheck: []fields.RoleCheck{
        {
            Role: "moderator",
            Rules: []fields.CheckRules{
                fields.InRule(table.Courses.Status, []interface{}{"draft"}, []fields.Scenario{fields.ScenarioAdd}),
            },
        },
    },
    RoleOptions: []fields.RoleOptions{
        {
            Role: "moderator",
            Options: []fields.ModuleFieldOptions{
                {Value: "draft", Label: "Draft"},
            },
        },
    },
}
```

---

## Описание компонентов

### Generator

`Generator` — центральный объект, который принимает описания модулей и регистрирует HTTP-маршруты.

Дополнительные настройки после создания:

```go
generator.Locales = []locale.Lang{locale.EN, locale.AR}
generator.DefaultLocale = locale.EN
generator.EnableOpenAPI = true
generator.LoadTranslationsFile(locale.EN, "translations/en.json")
generator.LoadTranslationsFile(locale.AR, "translations/ar.json")
```

Метод `generator.Run()`:
1. Создаёт `GET /admin/api/features` — список всех модулей и действий
2. Для каждого модуля регистрирует CRUD-маршруты с middleware
3. Создаёт `GET /admin/api/lang` и `GET /admin/api/lang/:key` — i18n
4. Создаёт `GET /admin/api/openapi.json` (если `EnableOpenAPI`)

### DBExecutor

Интерфейс для работы с БД. Реализация `db.NewDB(sqlDB)` для PostgreSQL.

```go
type DBExecutor interface {
    List(...)   ([]interface{}, int64, error)
    View(...)   (interface{}, error)
    Add(...)    (interface{}, error)
    Update(...) (interface{}, error)
    Delete(...) error
    RawRequest(log, query, params...) (*sql.Rows, error)
    RawDB() *sql.DB  // доступ к raw *sql.DB — используется в DataCheckRule.ValidateData
}
```

### TranslationContext

Автоматически формируется из `BaseModule` для модулей с переводимыми полями:

```go
type TranslationContext struct {
    EntityName string                  // имя сущности (module.EntityName или table name)
    Fields     []TranslatableFieldInfo // список переводимых полей
    Langs      []string               // поддерживаемые языки
    EntityID   interface{}            // ID записи (для Update/Delete)
}
```

---

## API эндпоинты

Для модуля с `Name: "courses"` и `Path: ""` генерируются:

| Метод    | URL                                    | Действие | Описание                              |
|----------|----------------------------------------|----------|---------------------------------------|
| `GET`    | `/admin/courses`                       | List     | Список с пагинацией и фильтрами      |
| `PUT`    | `/admin/courses`                       | Add      | Создание записи                       |
| `GET`    | `/admin/courses/defrec/`               | Defrec   | Структура формы с полями и опциями    |
| `GET`    | `/admin/courses/view/:bykey/:value`    | View     | Просмотр записи                       |
| `POST`   | `/admin/courses/:bykey/:value`         | Update   | Обновление записи                     |
| `DELETE` | `/admin/courses/delete/:bykey/:value`  | Delete   | Удаление записи                       |

### Query-параметры List

| Параметр        | Описание                                |
|-----------------|-----------------------------------------|
| `page`          | Номер страницы (по умолч. 0)            |
| `size`          | Размер страницы (по умолч. 3000)        |
| `filter[field]` | Фильтр по полю                          |
| `search`        | Полнотекстовый поиск                    |
| `sort`          | Сортировка (`field:asc` или `field:desc`) |
| `addFilters`    | `true` — включить метаданные фильтров   |
| `addHeads`      | `true` — включить заголовки колонок      |
| `csv`           | `1` — вернуть TSV вместо JSON           |
| `lang`          | Код языка (или `Accept-Language` заголовок) |

### Служебные эндпоинты

| Метод | URL                       | Описание                              |
|-------|---------------------------|---------------------------------------|
| `GET` | `/admin/api/features`     | Список модулей и действий с ролями    |
| `GET` | `/admin/api/lang`         | Список поддерживаемых языков          |
| `GET` | `/admin/api/lang/:key`    | Все переводы для языка                |
| `GET` | `/admin/api/openapi.json` | OpenAPI 3.0 спецификация              |
| `GET` | `<path>/config`           | Клиентский конфиг: navigation, widgets и роль |

### Config endpoint

Возвращает конфигурацию для клиентского приложения. Навигация и виджеты фильтруются по роли текущего пользователя и permission действия.

**Query-параметры:** `lang` — код языка для перевода заголовков навигации.

**Пример ответа:**
```json
{
  "navigation": [
    {
      "id": "catalog.list",
      "path": "/catalog",
      "target": {
        "type": "page",
        "renderer": {
          "name": "UniversalRenderer",
          "version": "1.0.0"
        },
        "page_type": "list",
        "query": {
          "url": "/api/api/catalog",
          "method": "GET"
        },
        "data": {
          "view_adapter": "list_table"
        },
        "children": {}
      },
      "title": "Catalog",
      "icon": "catalog",
      "order": 10,
      "group": "main"
    }
  ],
  "widgets": [
    {
      "id": "profile-menu",
      "type": "module",
      "placement": "topbar",
      "order": 10,
      "query": {
        "url": "/api/api/profile-menu",
        "method": "GET"
      }
    }
  ],
  "role": "admin"
}
```

Навигация строится из `Navigation` каждого `BaseModule`. Пункты сортируются по `Group`, затем по `Order`. Для `target.type=page` frontend route хранится в `path`, а renderer/query/children находятся прямо в `target`. Для popup/client_action route не требуется. Глобальные виджеты строятся из `WidgetConfig` на действиях модулей и возвращаются в `widgets`.
