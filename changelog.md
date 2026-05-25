# Changelog

## [Unreleased]

### Added

- **`MenuEntry` на `BaseModule`** — декларативное описание пунктов левого меню для клиентского конфига.
  Поля: `ActionName`, `Title`, `Icon`, `Show`, `Order`, `Group`, `Roles`, `CustomLink`, `CustomQuery`, `CustomData`.

- **`/api/config` эндпоинт** — возвращает клиентский конфиг с role-based левым меню (`left_menu`).
  Меню формируется из `MenuEntries` каждого модуля, группируется по `Group`, фильтруется по `Roles` текущего пользователя.
  Заголовки блоков переводятся через i18n (lang из query-параметра или `Accept-Language`).

  ```
  GET /api/config?lang=ru
  Response: { "left_menu": [{ "blockTitle": "...", "elements": [...] }] }
  ```

- **`ExtraFunc` на `ListModuleAction`** — динамические extra-данные per-request.
  Функция вызывается при каждом List-запросе; результат добавляется в ответ как `extra`.
  Используется для динамических pills, счётчиков и других данных, зависящих от роли/контекста запроса.

  ```go
  ExtraFunc: func(c *gin.Context) interface{} {
      return map[string]interface{}{"pills": buildPills(c)}
  }
  ```

- **`FilterFunc` на `ListModuleAction`** — динамический список фильтруемых колонок per-request.
  Заменяет/дополняет статический `Filter []pg.Column` когда набор доступных фильтров зависит от роли или других условий запроса.

  ```go
  FilterFunc: func(c *gin.Context) []pg.Column {
      user, _ := icontext.GetUser(c.Request.Context())
      if user.Role == "admin" {
          return []pg.Column{table.Profiles.Status, table.Profiles.AgencyID}
      }
      return []pg.Column{table.Profiles.Status}
  }
  ```

- **`FilterCondition` на `ModuleField`** — функция условия видимости поля в фильтрах.
  Если задана — поле включается в фильтры только когда функция возвращает `true`.

  ```go
  {
      Column:          table.Profiles.AgencyID,
      FilterCondition: func(c *gin.Context) bool {
          user, _ := icontext.GetUser(c.Request.Context())
          return user.Role != "model"
      },
  }
  ```

- **`Where` на `UpdateModuleAction`** — row-level авторизация при обновлении.
  Если задана — WHERE-условие добавляется к UPDATE-запросу. Если ни одна строка не обновлена (условие не выполнено) — возвращается 404.

  ```go
  actions.UpdateModuleAction{
      Where: func(c *gin.Context) pg.BoolExpression {
          user, _ := icontext.GetUser(c.Request.Context())
          if user.Role == "admin" {
              return nil
          }
          return table.Profiles.UserID.EQ(pg.Int(user.ID))
      },
  }
  ```

- **`Where` на `DeleteModuleAction`** — row-level авторизация при удалении.
  Аналогично UpdateModuleAction: если условие не выполнено — запись не удаляется, возвращается 404.

  ```go
  actions.DeleteModuleAction{
      Where: func(c *gin.Context) pg.BoolExpression {
          user, _ := icontext.GetUser(c.Request.Context())
          if user.Role == "admin" {
              return nil
          }
          return table.Profiles.UserID.EQ(pg.Int(user.ID))
      },
  }
  ```

- **`Group` и `Order` на `ModuleField`** — организация полей в группы фильтров.
  `Group` — строковый ключ группы (например `"breast"`, `"options"`).
  `Order` — порядок внутри группы.
  Оба поля экспортируются в JSON ответе `addFilters=true`.

  ```go
  {
      Column: table.Profiles.BreastSize,
      Group:  "breast",
      Order:  1,
  }
  ```

### Changed

- `LeftMenuItem` расширен полями `Query` (`map[string]interface{}`) и `Data` (`map[string]interface{}`) для передачи дополнительных параметров клиенту.
- `CustomLink` в `MenuEntry` позволяет переопределить URL пункта меню (вместо дефолтного API-пути модуля).
