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

### Этап 6. Описание действий (Actions)

Каждый модуль должен содержать массив `Actions` с нужными CRUD-операциями.

#### ListModuleAction

```go
actions.ListModuleAction{
    Label:   "courses.list",
    Auth:    true,                                        // требуется авторизация
    Permission: []actions.Role{"admin", "moderator"},     // допустимые роли (опционально)
    Columns: []pg.Column{table.Courses.ID, ...},          // отображаемые колонки
    Filter:  []pg.Column{table.Courses.Status},           // фильтруемые колонки
    Search:  []pg.Column{coursesTitle},                   // поиск по колонкам
    Sort:    []pg.Column{table.Courses.ID, table.Courses.Price}, // сортируемые колонки
    SortDefault: table.Courses.ID,                        // сортировка по умолчанию
    Size:    50,                                          // размер страницы по умолчанию
    Maxsize: 1000,                                        // макс. размер страницы
    Join:    []actions.ModuleActionJoin{...},              // JOIN-ы
    Where:   func(c *gin.Context) pg.BoolExpression {...}, // доп. WHERE
}
```

**Обязательные поля:** `Label`, `Columns`.

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
    Columns: []pg.Column{...},               // редактируемые колонки
    By:      []pg.Column{table.Courses.ID},  // по каким ключам обновлять
}
```

**Обязательные поля:** `Label`, `Columns`, `By`.

Опция `ViewAfterUpdate *bool` (по умолч. `true`) — после обновления возвращает полный View-ответ, если у модуля есть ViewAction.

#### DeleteModuleAction

```go
actions.DeleteModuleAction{
    Label: "courses.delete",
    Auth:  true,
    By:    []pg.Column{table.Courses.ID},  // по каким ключам удалять
}
```

**Обязательные поля:** `Label`, `By`.

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

| Поле       | Обязательное | Описание                                      |
|------------|-------------|------------------------------------------------|
| `Column`   | да          | Jet-колонка (реальная или sentinel)            |
| `Title`    | да          | Ключ перевода заголовка                        |
| `Type`     | да          | Тип данных (string/int/float/array/object)     |
| `FormType` | да          | Тип формы (text/number/select/map/...)         |

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
