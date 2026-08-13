# Инструкции Для Агентов: Request Generator

Этот файл обязателен для любой работы в `request-generator`. Это независимая
Go-библиотека. Она не знает конкретный продукт, его таблицы, роли, ресурсы,
маршруты, переводные ключи, палитру или Vue-компоненты. Исполняемая
спецификация frontend metadata находится в
[docs/universal-renderer-contract.md](docs/universal-renderer-contract.md).

## Цель И Граница Библиотеки

`request-generator` владеет generic CRUD-механикой и стабильным typed
контрактом `UniversalRenderer`. Приложение описывает собственные модули через
`BaseModule`, `ModuleField`, actions и typed `renderer` structures; generator
валидирует, локализует и сериализует этот контракт.

Библиотека должна предоставлять:

- typed Go structures и constants/enums для renderer metadata;
- validation инвариантов contract;
- единое включение renderer identity/version в response;
- generic binding, localization и serialization behavior;
- tests, которые проверяют contract без зависимости от продукта.

Библиотека не должна содержать:

- название бизнес-модуля, таблицы, роли, route, field key или продуктовый
translation key;
- SQL, project DB queries, seed data, CSS/цвета/px или Vue-код;
- producer-specific `RenderFunc`, hooks или special response composer;
- project fallback, compatibility alias или magic string для одного клиента.

## Нормативные Правила

### 1. Typed Contract Вместо `extra`

Новая UI-возможность добавляется typed полем в `renderer`/`fields`/`module`
только когда текущий contract действительно не может выразить потребность.
Нельзя предлагать `Extra`, `ExtraFunc`, arbitrary UI map или untyped JSON как
универсальное решение.

`map[string]interface{}` допустим только в уже обозначенных runtime/transport
контейнерах, где содержимое является данными: context, action payload/query,
route query. Он не может описывать renderer shape, component, layout, visual
variant или field schema.

Closed перечни значений оформляются typed enum/constants и валидируются.
Расширяемые application tokens не становятся enum библиотеки: сохраняйте их
как string, документируйте семантику и не вшивайте продуктовый допустимый
список.

### 2. `ModuleField` И `BaseModule.Render` Не Взаимозаменяемы

| Ответственность | Место |
|---|---|
| Тип значения, conversion, checks, options, single-field presentation/media | `ModuleField` |
| CRUD/action mechanics, permissions, filters, sort, pagination | `ModuleAction`/`BaseModule` |
| Композиция страницы, layout, section, matrix, card, action placement | `BaseModule.Render` |
| Runtime variation готовой typed схемы | `RenderFunc` |

Не дублируйте field type, form type, checks, options, labels или value format
в matrix/card/page structures. Matrix ссылается на уже описанное field; action
ссылается на generic bindings; page metadata компонуется из typed blocks.

### 3. Расширение Контракта

До изменения public renderer структуры агент обязан:

1. Прочитать спецификацию и существующие typed structures/tests.
2. Сформулировать конкретный missing case без названия продукта.
3. Показать current shape, почему она не выражает case, и minimal proposed
   typed shape.
4. Получить согласование мейнтейнера перед реализацией нового contract.
5. Обновить specification, validation, localization/clone/serialization и
   unit/integration tests в одном PR.
6. Описать version/compatibility impact. Не вводить silent compatibility
   fallback или aliases.

При сомнении сначала задайте вопрос или создайте issue. Не расширяйте public
contract на основании одного UI-экрана.

### 4. RenderFunc И Generic Hooks

`RenderFunc` остаётся producer callback-ом, работающим с deep clone typed
`renderer.Universal`. Generator вызывает его и валидирует итог. Он не получает
project behavior внутри библиотеки и не является местом для SQL/DB queries,
manual response composition или cache глобального состояния.

Generator не должен допускать путь, по которому producer обходит `Validate()`
или сам задаёт `renderer.name/version`: identity/version добавляет generator.

### 5. Localization И Serialization

- Renderer metadata содержит localization keys в producer declaration;
  generator локализует их для текущего request language там, где это допускает
  contract.
- `renderer.name/version` добавляются автоматически к response с typed page
  metadata.
- JSON shape должен быть стабилен и описан в specification. Новое поле без
  спецификации и test нельзя считать частью contract.

## Запрещено

- Добавлять в library конкретные модули, роли, сущности, таблицы, endpoint-ы,
  field names, цвета, CSS, размеры или иконки приложения.
- Добавлять generic type, который на самом деле является ad-hoc map.
- Делать special-case по строке, чтобы пройти один проектный тест.
- Ослаблять validation ради обратной совместимости с legacy `extra`.
- Делать изменения public renderer API без согласования.
- Считать webapp renderer источником истины для contract: источник истины —
  typed API и `docs/universal-renderer-contract.md`.

## Чек-Лист PR

- [ ] Изменение применимо минимум к одному абстрактному domain case без
      проектных терминов.
- [ ] Нет проектных строк/SQL/ролей/routes в library.
- [ ] Новая shape typed, документирована и валидируется.
- [ ] Обновлены clone, localization и serialization, если field через них идёт.
- [ ] Покрыты positive и invalid contract cases.
- [ ] Нет legacy fallback или `extra` escape hatch.
- [ ] Выполнены `go test ./...` и необходимые integration tests.
