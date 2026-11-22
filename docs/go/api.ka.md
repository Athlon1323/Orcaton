# ბირთვის API ფანჯრისთვის (`orcaton/api`)

[English](api.md) · [Deutsch](api.de.md) · [Русский](api.ru.md) · **ქართული**

საჯარო Go-პაკეტი `app-go/api` არის ის, რითაც გრაფიკული ფანჯარა (`app-gui/`, Wails) ბირთვს იძახებს. CLI `orcaton sync` იმავე
სცენარებზე (`internal/app`) მუშაობს, ამიტომ ფანჯარა და ბრძანების სტრიქონი **ერთნაირ შედეგს** იძლევა: ტესტი
`api/gui_test.go` API-ის გამოსავალსა და ანგარიშს ბაიტ-ბაიტ ადარებს CLI-ის სნეპშოტებს. პაკეტები `internal/…` სხვა
მოდულიდან მიუწვდომელია (Go-ს წესი) — ფანჯარა ხედავს მხოლოდ `api`-ს.

## მიერთება
```
app-gui/go.mod:   require orcaton v0.0.0
                  replace orcaton => ../app-go
კოდი:             import "orcaton/api"
```
ბირთვს გარე დამოკიდებულებები არ აქვს; Wails-ის დამოკიდებულებები მხოლოდ `app-gui/`-შია. კონტრაქტის ვერსია —
`api.Version` (შეუთავსებელი ცვლილებისას იცვლება), პროგრამის ვერსია — `api.ProductVersion`.

## ფანჯრის სცენარი
| ნაბიჯი | გამოძახება | რა აჩვენოს |
|---|---|---|
| გაშვება | `api.FFmpegAvailable()` | FFmpeg არ არის — `api.Describe(err)`: ტექსტი + ინსტალაციის მინიშნება |
| About | `api.GetFFmpegInfo()` → `{ffmpeg, ffprobe, version, old, min}` (`old` — `min`-ზე ძველი = 4.4: გააფრთხილოს); `api.GetCacheInfo(dir)` → `{path, exists, entries, bytes, partial}`, `api.ClearCache(dir)` (`""` — სისტემური საქაღალდე) | FFmpeg-ის ვერსია და ბილიკი, ქეშის ზომა, „გასუფთავება“ |
| XML/DRT-ის გახსნა | `p, err := api.Open(path, api.OpenOptions{})` → `p.Info()` | კამერები, ფაილები (`ok` / `missing` / `offline` / `nomedia`), რამდენი ვერ მოიძებნა, შენახვის ფორმატები (`outputFormats` — მხოლოდ შესავლის ფორმატი: XML → XML, DRT → DRT) |
| მედიის ძებნა | `p.FindMedia(ctx, api.FindRequest{Roots, Ask, OnSearch})` | თითოეული ძებნის შედეგი (`SearchReport`), კითხვა — დიალოგი (`Asker`) |
| პარამეტრები | `api.LoadSettings(project)` → `{options, media, files}`; `api.DefaultSettings()` / `api.DefaultOptions()`, `o.Validate()`; სიები — `api.UnsyncedModes`, `api.OrderModes`, `api.SinglesModes`, `api.AudioModes`; შენახვა — `api.SaveSettings(path, s)` | შეცდომა `bad_options` ველის სახელით (ფაილში — ბილიკითა და გასაღებით) |
| სინქრონიზაცია | `res, err := api.Sync(ctx, p, api.SyncRequest{Options: o}, onEvent)` | პროგრესის ზოლი `Event`-ის მიხედვით; „გაჩერება“ — ctx-ის `cancel()` |
| შედეგი | `res.Summary`, `res.Files` | ცხრილი: ხმა / საათი / უწყვილო / დისკზე არ არის (`FileResult.Result`), მიზეზი კოდით (`reasonCode`), პარტნიორი; „მანამდე“ — `startSec`, „შემდეგ“ — `positionSec` / `positionTC` |
| რესურსები | `Options.Jobs` / `Threads` / `MemLimitMB` (0 — ავტო: ffmpeg ბირთვებისა და დისკის მიხედვით, ყველა ბირთვი, ოპერატიულის ¾ 1…7 GB-ის ფარგლებში); შედეგი — `res.Summary.Resources` (`ramMB`, `memLimitMB`, `threads`, `jobs`, `disk`, `peakMB`, `ffmpeg` — იმ FFmpeg-ის ვერსია, რომელმაც ხმა წაიკითხა) | პარამეტრები „წარმადობა“; მეხსიერების პიკი — შედეგის ფანჯარაში |
| შენახვა | `res.Save(path, api.SaveOptions{Format, KeepPaths}, onEvent)`, `res.SaveReport(csv)` | შეიძლება რამდენჯერმე; ფანჯარა სთავაზობს მხოლოდ შესავლის ფორმატს; `.drt` — მხოლოდ DRT-ის შესავლისთვის; DRT → XML — პირდაპირი `Format: "xmeml"`-ით (ტერმინალი ასე აკეთებს) |

## ტაიმლაინი „მანამდე / შემდეგ“ კამერების მიხედვით
* ტაიმლაინის სტრიქონი **კამერაა** (`ProjectInfo.cameras`: `id`, `title` „6D · V1+A1“, `tracks`, `files`);
  ფაილი მიეკუთვნება `camera`-ით, ფაილის ტრეკები — `tracks`. `id` არის „ტრეკი|საქაღალდე“; თუ საქაღალდეში სხვადასხვა
  გაფართოების ფაილებია (ორი კამერა ერთ საქაღალდეში) — „ტრეკი|საქაღალდე|.ext“, `title` — „6D2 MOV · V2+A2“.
* „მანამდე“ — `MediaFile.startSec` / `FileResult.startSec`: ფაილის დასაწყისი შემავალ ტაიმლაინზე (კლიპი =
  ფაილის დასაწყისი + შესვლის წერტილი), წამები სექვენციის დასაწყისიდან; „შემდეგ“ — `FileResult.positionSec`
  (`nil` — ფაილი მოხსნილია). სიგრძე — ფაილის `durationSec`.
* ტაიმკოდი: `SequenceInfo{fpsNum, fpsDen, startSec, startTC}` — ფანჯარა სკალას თავად ხატავს;
  მზა სტრიქონები — `MediaFile.startTC`, `FileResult.positionTC`; `api.FormatTC(sec, seq)` — იგივე
  Go-ში (სექვენციის საწყისი TC, drop-frame-ის გარეშე — როგორც XML-ის გამოსავალში).
* ტალღა კლიპებზე v1-ში დეკორატიულია (ხმის ნიშან-თვისებას API არ გასცემს).

## ტექსტები და თარგმანი
ფანჯარა ლოკალიზებულია (ინგლისური, გერმანული, რუსული, ქართული); ბირთვის ტექსტები ინგლისურია. თარგმნეთ **კოდების მიხედვით**; ბირთვის ტექსტი სარეზერვოა და
განკუთვნილია „Details“-ისთვის: ეტაპი — `Event.stage`; შეცდომა — `ErrorInfo.kind`; ფაილის მიზეზი —
`FileResult.reasonCode`: `no_match` (ხმით საიმედო დამთხვევა არ არის), `short_audio`, `no_audio`,
`offline` (დისკზე არ არის, მონტაჟის პროგრამაში ოფლაინია), `no_path`, `not_found` (ბილიკზე ვერ მოიძებნა),
`decode_error`, `conflict` (გადაფარვა საკუთარ ტრეკზე), `by_clock` (საათით, ხმამ ვერ დაადასტურა),
`duplicate` (პროექტის სხვა ფაილის ასლი მასთან საერთო ტრეკზე; DRT-ში ტრეკები არ ემატება — ასლი უწყვილო რჩება);
მუდმივები — `api.Reason*`. CLI-ისა და CSV-ანგარიშის ტექსტები არ იცვლება.

`Summary.duplicates` — ფაილების ასლები (იგივე ხმა): XML-ში დგანან ორიგინალთან ერთად, მათი კლიპები გადაფარვისას —
დამატებით ტრეკებზე (V/A ნომრით ბოლოს შემდეგ); `Summary.timeShift` — `timeAudio`-დან
წანაცვლების შემოწმებით დადასტურებული (მუსიკა).

ყველა ტიპი json-ტეგებიანი სტრუქტურაა (Wails მათგან აგებს TypeScript-ის მოდელებს). `Result` ინახავს
შედეგს მეხსიერებაში: ფანჯარა მაჩვენებელს თავის სტრუქტურაში ინახავს და JS-ს აძლევს მხოლოდ `Summary`/`Files`-ს.

## პროგრესი — `Event`
`{stage, title, done, total, file}`; `done == total` — ეტაპი დასრულდა, `total == 0` — მთვლელის გარეშე.
ეტაპები რიგით: `search` (მედიის ძებნა) → `record_time` (კამერების საათები) → `features` (ხმა, ფაილის მიხედვით:
`file`) → `pairs` (წყვილების შედარება, ~1 % — 100 მოვლენამდე) → `refine` → `solve` (3 ნაბიჯი) → `layout` →
`write`. იგივე ნაკადს ბეჭდავს `orcaton sync --progress json` (JSON-ის ხაზი მოვლენაზე, stderr).
დამმუშავებელი გამოიძახება თანმიმდევრულად, მაგრამ სამუშაო გორუტინებიდან: მხოლოდ `runtime.EventsEmit` და დაბრუნება.

## გაუქმება
`Sync`-ისა და `FindMedia`-ს `ctx`. მოწმდება ეტაპებს შორის და გრძელების შიგნით: ჩაწერის დრო — ფაილის მიხედვით,
ნიშან-თვისებები — ffmpeg-ის პროცესები სრულდება (`exec.CommandContext`), წყვილები — სერიების მიხედვით. შეცდომა სახეობის
`cancelled`, შედეგი არ არის, პროექტი არ იცვლება. საქაღალდეების ინდექსაცია მედიის ძებნისას (დისკის გავლა)
ჯერ არ წყდება — მხოლოდ ძებნებს შორის.

## კითხვა ვერნაპოვნი მედიის შესახებ — `Asker`
`AskMissing(ctx, missing []MissingFile) (Answer, error)` გამოიძახება, სანამ სია ცარიელი არ არის.
პასუხი: `{action: "folder", folder}` — იქ ეძებოს (თუ ფაილი მიუთითეს — მისი საქაღალდე ისხება), `"skip"` —
დარჩენილები ყველაზე ბოლოში დგებიან როგორც „დისკზე არ არის“ (მონტაჟის პროგრამაში — ოფლაინი), `"cancel"` — შეცდომა
`cancelled`. გამოძახება ბლოკავს: Wails-ში — მოვლენა ფანჯარაში და პასუხის ლოდინი არხიდან:
```go
func (a *App) AskMissing(ctx context.Context, ms []api.MissingFile) (api.Answer, error) {
	runtime.EventsEmit(a.ctx, "media:missing", ms)
	select {
	case ans := <-a.answers: // მეთოდი App.Answer(ans) გამოიძახება JS-იდან
		return ans, nil
	case <-ctx.Done():
		return api.Answer{}, ctx.Err()
	}
}
```
CLI-ში მის ადგილას ტერმინალია (`internal/cli/mediafind.go`, `termAsker`).

## პარამეტრები — `orcaton.toml`
ფანჯარა და `orcaton sync` ერთსა და იმავე ფაილებს კითხულობენ ([usage.ka.md](usage.ka.md#პარამეტრების-ფაილი-orcatontoml)):
* `api.LoadSettings(project)` — ნაგულისხმევი ← მომხმარებლის ფაილი (`api.UserSettingsPath()`) ← ფაილი
  პროექტის გვერდით (`api.ProjectSettingsPath(project)`; `""` — მხოლოდ მომხმარებელი). შედეგი —
  `Settings{Options, Media{Map, Search, SearchDepth, KeepPaths, Originals}, Files}`;
  `Media.SearchRoots()` — საქაღალდეები `FindRequest.Roots`-ისთვის (როგორც `--map საქაღალდე` + `--search`).
  `Map` → `OpenOptions.PathMap`, `KeepPaths` → `SaveOptions`, `Originals` → `SyncRequest`.
* შეცდომა ფაილში — `bad_options` (ტექსტში ბილიკი და გასაღები) და `DefaultSettings()`: ფანჯარა აჩვენებს
  შეცდომას და ნაგულისხმევით მუშაობს.
* `api.SaveSettings(path, s)` — `[sync]` სექციაში წერს ნაგულისხმევისგან განსხვავებულ მნიშვნელობებს (გასაღები,
  რომელიც ფაილში უკვე წერია, ნაგულისხმევის შემთხვევაშიც განახლდება); კომენტარები, სხვა სექციები და მხოლოდ ტერმინალის
  გასაღებები (`progress`) ხელუხლებელია; ჩაწერა ატომურია; არასწორი პარამეტრები — `bad_options`, ფაილი
  არ იცვლება. სად შეინახოს (მომხმარებელი / პროექტი) — ფანჯარა წყვეტს.
* `MemLimitMB` ფაილში ინახება GB-ში (`mem-limit`), როგორც ფლაგი.

## შეცდომები
`api.KindOf(err)` → `cancelled` · `no_ffmpeg` · `not_found` · `bad_format` · `bad_options` ·
`bad_output` · `write` · `failed`; `api.Describe(err)` → `{kind, message, hint}` ფანჯრისთვის. შეცდომის
ტექსტი იგივეა, რაც CLI-ში. FFmpeg-ის არქონა მედიით სინქრონიზაციისას — დაუყოვნებლივ `no_ffmpeg` (ადრე
ყოველი ფაილი იღებდა „დეკოდირების შეცდომას“).

## ნაკადები
ერთი `Project` — ერთი ოპერაცია ერთდროულად (`FindMedia` პროექტში ბილიკებს ანაცვლებს). `Sync` პროექტს არ ცვლის
(ასლზე მუშაობს), `Result.Save` შეიძლება განმეორებით გამოიძახოთ.
