# Changelog

## [Unreleased]

### Added

- **`AtomicExecutor.SelectMany`** — ограниченное и детерминированное typed
  чтение нескольких строк внутри generator-owned transaction. Контракт требует
  `ORDER BY` и положительный `LIMIT`; пустой результат не является ошибкой.

- **`AtomicExecutor.Update`** — typed обновление строк с обязательным Jet
  `Where`; закрытые операции `set` и numeric `increment` не допускают
  произвольных SQL expressions. Добавлен `AtomicTime` для timestamp values.

- **`AtomicExecutor.Upsert` и atomic realtime publish** — конфликтобезопасное
  идемпотентное создание без доступа модуля к transaction/raw SQL, а также
  typed публикация actor-scoped realtime-события только после commit.
  Recipient topics строятся generator-ом из server-produced result fields.

- **`NavigationEntry` на `BaseModule`** — декларативное описание навигации и frontend routes для config endpoint.
  Поля: `ActionName`, `ID`, `Path`, `Title`, `Icon`, `Show`, `Order`, `Group`, `Target`, `Roles`, `Query`, `Data`.

- **Config endpoint** — возвращает единый role-based список `navigation`, глобальные `widgets` и роль пользователя.
  Навигация формируется из `Navigation` каждого модуля, фильтруется по правам действия и `Roles` пункта.
  Для `target.type=page` renderer/query/children встраиваются прямо в `navigation[].target`, без отдельного `routes`.

- **`WidgetConfig` на действиях модулей** — глобальные виджеты описываются на конкретном действии (`list`, `view`, `add`, `defrec`, ...).
  Такие виджеты автоматически попадают в `/api/config.widgets` с query соответствующего действия.

- **`ExtraFunc` на `ListModuleAction`** — динамические extra-данные per-request.
  Функция вызывается при каждом List-запросе; результат добавляется в ответ как `extra`.

  ```go
  ExtraFunc: func(c *gin.Context) interface{} {
      return map[string]interface{}{"pills": buildPills(c)}
  }
  ```

- **`FilterFunc` на `ListModuleAction`** — динамический список фильтруемых колонок per-request.
  Заменяет/дополняет статический `Filter []pg.Column` когда набор доступных фильтров зависит от контекста запроса.

  ```go
  FilterFunc: func(c *gin.Context) []pg.Column {
      // вернуть нужные колонки на основе c
  }
  ```

- **`FilterCondition` на `ModuleField`** — функция условия видимости поля в фильтрах.
  Если задана — поле включается в фильтры только когда функция возвращает `true`.

  ```go
  {
      Column:          table.Courses.Price,
      FilterCondition: func(c *gin.Context) bool { ... },
  }
  ```

- **`Where` на `UpdateModuleAction`** — дополнительное WHERE-условие для UPDATE.
  Если ни одна строка не обновлена — возвращается 404. Возврат `nil` снимает ограничение.

  ```go
  actions.UpdateModuleAction{
      Where: func(c *gin.Context) pg.BoolExpression {
          // вернуть условие или nil
      },
  }
  ```

- **`Where` на `DeleteModuleAction`** — дополнительное WHERE-условие для DELETE.
  Если ни одна строка не удалена — возвращается 404.

  ```go
  actions.DeleteModuleAction{
      Where: func(c *gin.Context) pg.BoolExpression {
          // вернуть условие или nil
      },
  }
  ```

- **`Group` и `Order` на `ModuleField`** — организация полей в группы фильтров.
  `Group` — строковый ключ группы. `Order` — порядок внутри группы.
  Оба поля экспортируются в JSON при `addFilters=true`.

  ```go
  {
      Column: table.Courses.Status,
      Group:  "options",
      Order:  1,
  }
  ```

- **`DataCheckRule` interface** — валидация с доступом к БД и контексту запроса.
  Встраивает `CheckRules` (обратная совместимость) и добавляет `ValidateData`. Генератор автоматически определяет реализацию и вызывает `ValidateData` вместо `Validate`.

  ```go
  type DataCheckRule interface {
      CheckRules
      ValidateData(c *gin.Context, db *sql.DB, data map[string]interface{}, lang string) error
  }
  ```

  Реализация через конструктор `DataRule()` или собственный тип:

  ```go
  // Inline:
  fields.DataRule(func(c *gin.Context, db *sql.DB, data map[string]interface{}, lang string) error {
      // доступ к data["field"], выполнение DB-запросов
      return nil
  }, []fields.Scenario{fields.ScenarioAdd})

  // Собственный тип:
  type myRule struct{ scenarios []fields.Scenario }
  func (r myRule) Validate(_ interface{}, _ string) error { return nil }
  func (r myRule) GetScenarios() []fields.Scenario       { return r.scenarios }
  func (r myRule) ValidateData(c *gin.Context, db *sql.DB, data map[string]interface{}, lang string) error {
      // логика
      return nil
  }
  ```

- **`DefaultFunc` защита от spoofing** — поля, не входящие в `action.Columns`, теперь всегда получают значение через `DefaultFunc(c)`, даже если клиент передал то же поле в теле запроса. Это предотвращает подмену системных полей (`who_add`, `user_id` и аналогичных).

- **`RawDB()` на `DBExecutor`** — новый метод интерфейса возвращает `*sql.DB` для использования в `DataCheckRule.ValidateData`.

  ```go
  type DBExecutor interface {
      // ...
      RawDB() *sql.DB
  }
  ```

- **`SetDB` / `GetDB` в `icontext`** — утилиты для передачи `*sql.DB` через `context.Context`.

### Changed

- **`Convert` на `ModuleField`** принимает `*gin.Context` первым аргументом.
  Это позволяет использовать роль, пользователя или другие данные контекста при преобразовании значения.

  ```go
  // Было:
  Convert: func(value interface{}) (interface{}, error) { ... }

  // Стало:
  Convert: func(c *gin.Context, value interface{}) (interface{}, error) { ... }
  ```

- `NavigationEntry` поддерживает поля `Query` (`map[string]interface{}`) и `Data` (`map[string]interface{}`) для передачи дополнительных параметров клиенту.
- `NavigationEntry.Path` позволяет явно задать frontend route для `target.type=page`.
