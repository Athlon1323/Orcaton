# API ядра для GUI (`orcaton/api`)

[English](api.md) · [Deutsch](api.de.md) · **Русский** · [ქართული](api.ka.md)

Публичный Go-пакет `app-go/api` — то, через что графический интерфейс (`app-gui/`, Wails) вызывает ядро. CLI `orcaton sync` работает на тех же
сценариях (`internal/app`), поэтому окно и командная строка дают **одинаковый результат**: тест
`api/gui_test.go` сверяет выход и отчёт API побайтно со снимками CLI. Пакеты `internal/…` из
другого модуля недоступны (правило Go) — окно видит только `api`.

## Подключение
```
app-gui/go.mod:   require orcaton v0.0.0
                  replace orcaton => ../app-go
код:              import "orcaton/api"
```
Ядро без внешних зависимостей; зависимости Wails — только в `app-gui/`. Версия контракта —
`api.Version` (меняется при несовместимых правках), версия программы — `api.ProductVersion`.

## Сценарий окна
| Шаг | Вызов | Что показать |
|---|---|---|
| старт | `api.FFmpegAvailable()` | нет FFmpeg — `api.Describe(err)`: текст + подсказка установки |
| About | `api.GetFFmpegInfo()` → `{ffmpeg, ffprobe, version, old, min}` (`old` — старше `min` = 4.4: предупредить); `api.GetCacheInfo(dir)` → `{path, exists, entries, bytes, partial}`, `api.ClearCache(dir)` (`""` — системная папка) | версия и путь FFmpeg, размер кэша, «очистить» |
| открыть XML/DRT | `p, err := api.Open(path, api.OpenOptions{})` → `p.Info()` | камеры, файлы (`ok` / `missing` / `offline` / `nomedia`), сколько не найдено, форматы сохранения (`outputFormats` — только формат входа: XML → XML, DRT → DRT) |
| найти медиа | `p.FindMedia(ctx, api.FindRequest{Roots, Ask, OnSearch})` | итог каждого поиска (`SearchReport`), вопрос — диалог (`Asker`) |
| настройки | `api.LoadSettings(project)` → `{options, media, files}`; `api.DefaultSettings()` / `api.DefaultOptions()`, `o.Validate()`; списки — `api.UnsyncedModes`, `api.OrderModes`, `api.SinglesModes`, `api.AudioModes`; сохранить — `api.SaveSettings(path, s)` | ошибка вида `bad_options` с именем поля (в файле — с путём и ключом) |
| синхронизация | `res, err := api.Sync(ctx, p, api.SyncRequest{Options: o}, onEvent)` | полоса прогресса по `Event`; «Стоп» — `cancel()` у ctx |
| результат | `res.Summary`, `res.Files` | таблица: звук / часы / без пары / нет на диске (`FileResult.Result`), причина кодом (`reasonCode`), партнёр; «до» — `startSec`, «после» — `positionSec` / `positionTC` |
| ресурсы | `Options.Jobs` / `Threads` / `MemLimitMB` (0 — авто: ffmpeg по ядрам и диску, все ядра, ¾ ОЗУ в пределах 1…7 ГБ); итог — `res.Summary.Resources` (`ramMB`, `memLimitMB`, `threads`, `jobs`, `disk`, `peakMB`, `ffmpeg` — версия FFmpeg, которым читался звук) | настройки «Производительность»; пик памяти — в окне итога |
| сохранить | `res.Save(path, api.SaveOptions{Format, KeepPaths}, onEvent)`, `res.SaveReport(csv)` | можно несколько раз; окно предлагает только формат входа; `.drt` — только для входа DRT; DRT → XML — явным `Format: "xmeml"` (так делает терминал) |

## Таймлайн «до / после» по камерам
* Строка таймлайна — **камера** (`ProjectInfo.cameras`: `id`, `title` «6D · V1+A1», `tracks`, `files`);
  файл — по `camera`, дорожки файла — `tracks`. `id` — «дорожка|папка»; если в папке файлы с разными
  расширениями (две камеры в одной папке), — «дорожка|папка|.ext», `title` — «6D2 MOV · V2+A2».
* «До» — `MediaFile.startSec` / `FileResult.startSec`: начало файла на входном таймлайне (клип =
  начало файла + точка входа), сек от начала секвенции; «после» — `FileResult.positionSec`
  (`nil` — файл убран). Длина — `durationSec` файла.
* Таймкод: `SequenceInfo{fpsNum, fpsDen, startSec, startTC}` — окно рисует шкалу само;
  готовые строки — `MediaFile.startTC`, `FileResult.positionTC`; `api.FormatTC(sec, seq)` — то же
  в Go (стартовый TC секвенции, без drop-frame — как в выходе XML).
* Волна на клипах в v1 — декоративная (признак звука в API не отдаётся).

## Тексты и перевод
Окно переведено (английский, немецкий, русский, грузинский); тексты ядра — английские. Переводить **по кодам**, текст ядра — запасной и
для «Details»: этап — `Event.stage`; ошибка — `ErrorInfo.kind`; причина файла —
`FileResult.reasonCode`: `no_match` (нет надёжного совпадения по звуку), `short_audio`, `no_audio`,
`offline` (нет на диске, офлайн в монтажной программе), `no_path`, `not_found` (не найден по пути),
`decode_error`, `conflict` (наложение на своей дорожке), `by_clock` (по часам, звук не подтвердил),
`duplicate` (копия другого файла проекта на общей с ним дорожке; в DRT дорожки не добавляются — копия без пары);
константы — `api.Reason*`. Тексты CLI и CSV-отчёта не меняются.

`Summary.duplicates` — копии файлов (тот же звук): в XML стоят вместе с оригиналом, их клипы при
наложении — на дополнительных дорожках (V/A с номером после последней); `Summary.timeShift` — из
`timeAudio` подтверждены проверкой сдвигом (музыка).

Все типы — структуры с json-тегами (Wails строит по ним модели TypeScript). `Result` держит
результат в памяти: окно хранит указатель в своей структуре и отдаёт в JS только `Summary`/`Files`.

## Прогресс — `Event`
`{stage, title, done, total, file}`; `done == total` — этап закончен, `total == 0` — без счётчика.
Этапы по порядку: `search` (поиск медиа) → `record_time` (часы камер) → `features` (звук, по файлу:
`file`) → `pairs` (сопоставление пар, ~1 % — до 100 событий) → `refine` → `solve` (3 шага) → `layout` →
`write`. Тот же поток печатает `orcaton sync --progress json` (строка JSON на событие, stderr).
Обработчик вызывается последовательно, но из рабочих горутин: только `runtime.EventsEmit` и выход.

## Отмена
`ctx` у `Sync` и `FindMedia`. Проверяется между этапами и внутри долгих: время записи — по файлу,
признаки — процессы ffmpeg завершаются (`exec.CommandContext`), пары — по сериям. Ошибка вида
`cancelled`, результата нет, проект не меняется. Индексация папок при поиске медиа (обход диска)
пока не прерывается — только между поисками.

## Вопрос про ненайденные медиа — `Asker`
`AskMissing(ctx, missing []MissingFile) (Answer, error)` вызывается, пока список не пуст.
Ответ: `{action: "folder", folder}` — искать там (указали файл — ищется в его папке), `"skip"` —
оставшиеся встанут в самый конец как «нет на диске» (в монтажной программе — офлайн), `"cancel"` — ошибка
`cancelled`. Вызов блокирующий: в Wails — событие в окно и ожидание ответа из канала:
```go
func (a *App) AskMissing(ctx context.Context, ms []api.MissingFile) (api.Answer, error) {
	runtime.EventsEmit(a.ctx, "media:missing", ms)
	select {
	case ans := <-a.answers: // метод App.Answer(ans) вызывается из JS
		return ans, nil
	case <-ctx.Done():
		return api.Answer{}, ctx.Err()
	}
}
```
В CLI на его месте терминал (`internal/cli/mediafind.go`, `termAsker`).

## Настройки — `orcaton.toml`
Окно и `orcaton sync` читают одни и те же файлы ([usage.ru.md](usage.ru.md#файл-настроек-orcatontoml)):
* `api.LoadSettings(project)` — умолчания ← файл пользователя (`api.UserSettingsPath()`) ← файл
  рядом с проектом (`api.ProjectSettingsPath(project)`; `""` — только пользователь). Результат —
  `Settings{Options, Media{Map, Search, SearchDepth, KeepPaths, Originals}, Files}`;
  `Media.SearchRoots()` — папки для `FindRequest.Roots` (как `--map ПАПКА` + `--search`).
  `Map` → `OpenOptions.PathMap`, `KeepPaths` → `SaveOptions`, `Originals` → `SyncRequest`.
* Ошибка в файле — `bad_options` (в тексте путь и ключ) и `DefaultSettings()`: окно показывает
  ошибку и работает с умолчаниями.
* `api.SaveSettings(path, s)` — пишет в раздел `[sync]` значения, отличные от умолчаний (ключ,
  уже записанный в файле, обновляется и при умолчании); комментарии, другие разделы и ключи только
  терминала (`progress`) не трогаются; запись атомарная; неверные настройки — `bad_options`, файл
  не меняется. Куда сохранять (пользователь / проект) — решает окно.
* `MemLimitMB` в файле хранится в ГБ (`mem-limit`), как у флага.

## Ошибки
`api.KindOf(err)` → `cancelled` · `no_ffmpeg` · `not_found` · `bad_format` · `bad_options` ·
`bad_output` · `write` · `failed`; `api.Describe(err)` → `{kind, message, hint}` для окна. Текст
ошибки — тот же, что у CLI. Нет FFmpeg при синхронизации по медиа — сразу `no_ffmpeg` (раньше
каждый файл получал «ошибку декодирования»).

## Потоки
Один `Project` — одна операция за раз (`FindMedia` подставляет пути в проект). `Sync` проект не
меняет (работает с копией), `Result.Save` можно вызывать повторно.
