# Kern-API für das Fenster (`orcaton/api`)

[English](api.md) · **Deutsch** · [Русский](api.ru.md) · [ქართული](api.ka.md)

Das öffentliche Go-Paket `app-go/api` ist das, worüber die grafische Oberfläche (`app-gui/`, Wails) den Kern aufruft. Die CLI `orcaton sync` arbeitet mit denselben
Szenarien (`internal/app`), daher liefern Fenster und Kommandozeile **dasselbe Ergebnis**: der Test
`api/gui_test.go` vergleicht Ausgabe und Bericht der API Byte für Byte mit den CLI-Snapshots. Die Pakete `internal/…` sind von einem
anderen Modul aus nicht erreichbar (Go-Regel) — das Fenster sieht nur `api`.

## Einbindung
```
app-gui/go.mod:   require orcaton v0.0.0
                  replace orcaton => ../app-go
Code:             import "orcaton/api"
```
Der Kern hat keine externen Abhängigkeiten; die Wails-Abhängigkeiten gibt es nur in `app-gui/`. Die Version des Vertrags ist
`api.Version` (ändert sich bei inkompatiblen Änderungen), die Programmversion — `api.ProductVersion`.

## Ablauf des Fensters
| Schritt | Aufruf | Was anzeigen |
|---|---|---|
| Start | `api.FFmpegAvailable()` | kein FFmpeg — `api.Describe(err)`: Text + Installationshinweis |
| About | `api.GetFFmpegInfo()` → `{ffmpeg, ffprobe, version, old, min}` (`old` — älter als `min` = 4.4: warnen); `api.GetCacheInfo(dir)` → `{path, exists, entries, bytes, partial}`, `api.ClearCache(dir)` (`""` — Systemordner) | FFmpeg-Version und -Pfad, Cache-Größe, „leeren“ |
| XML/DRT öffnen | `p, err := api.Open(path, api.OpenOptions{})` → `p.Info()` | Kameras, Dateien (`ok` / `missing` / `offline` / `nomedia`), wie viele nicht gefunden wurden, Speicherformate (`outputFormats` — nur das Eingabeformat: XML → XML, DRT → DRT) |
| Medien suchen | `p.FindMedia(ctx, api.FindRequest{Roots, Ask, OnSearch})` | Ergebnis jeder Suche (`SearchReport`), die Frage — ein Dialog (`Asker`) |
| Einstellungen | `api.LoadSettings(project)` → `{options, media, files}`; `api.DefaultSettings()` / `api.DefaultOptions()`, `o.Validate()`; Listen — `api.UnsyncedModes`, `api.OrderModes`, `api.SinglesModes`, `api.AudioModes`; speichern — `api.SaveSettings(path, s)` | Fehler `bad_options` mit Feldname (in einer Datei — mit Pfad und Schlüssel) |
| Synchronisation | `res, err := api.Sync(ctx, p, api.SyncRequest{Options: o}, onEvent)` | Fortschrittsbalken nach `Event`; „Stopp“ — `cancel()` des ctx |
| Ergebnis | `res.Summary`, `res.Files` | Tabelle: Ton / Uhr / ohne Partner / nicht auf der Festplatte (`FileResult.Result`), Grund als Code (`reasonCode`), Partner; „vorher“ — `startSec`, „nachher“ — `positionSec` / `positionTC` |
| Ressourcen | `Options.Jobs` / `Threads` / `MemLimitMB` (0 — auto: ffmpeg nach Kernen und Datenträger, alle Kerne, ¾ des RAM im Bereich 1…7 GB); Ergebnis — `res.Summary.Resources` (`ramMB`, `memLimitMB`, `threads`, `jobs`, `disk`, `peakMB`, `ffmpeg` — Version des FFmpeg, der den Ton gelesen hat) | Einstellungen „Leistung“; Speicherspitze — im Ergebnisfenster |
| Speichern | `res.Save(path, api.SaveOptions{Format, KeepPaths}, onEvent)`, `res.SaveReport(csv)` | mehrfach aufrufbar; das Fenster bietet nur das Eingabeformat an; `.drt` — nur bei DRT-Eingabe; DRT → XML — mit explizitem `Format: "xmeml"` (so macht es das Terminal) |

## Timeline „vorher / nachher“ nach Kameras
* Eine Zeile der Timeline ist eine **Kamera** (`ProjectInfo.cameras`: `id`, `title` „6D · V1+A1“, `tracks`, `files`);
  eine Datei gehört über `camera` dazu, die Spuren der Datei sind `tracks`. `id` ist „Spur|Ordner“; liegen im Ordner Dateien mit verschiedenen
  Endungen (zwei Kameras in einem Ordner) — „Spur|Ordner|.ext“, `title` — „6D2 MOV · V2+A2“.
* „Vorher“ — `MediaFile.startSec` / `FileResult.startSec`: der Anfang der Datei auf der Eingabe-Timeline (Clip =
  Dateianfang + Einstiegspunkt), Sekunden ab Sequenzanfang; „nachher“ — `FileResult.positionSec`
  (`nil` — die Datei wurde entfernt). Die Länge ist `durationSec` der Datei.
* Timecode: `SequenceInfo{fpsNum, fpsDen, startSec, startTC}` — das Fenster zeichnet die Skala selbst;
  fertige Zeichenketten — `MediaFile.startTC`, `FileResult.positionTC`; `api.FormatTC(sec, seq)` — dasselbe
  in Go (Start-TC der Sequenz, ohne Drop-Frame — wie in der XML-Ausgabe).
* Die Wellenform auf den Clips ist in v1 dekorativ (das Tonmerkmal gibt die API nicht heraus).

## Texte und Übersetzung
Das Fenster ist lokalisiert (Englisch, Deutsch, Russisch, Georgisch); die Texte des Kerns sind englisch. **Nach Codes** übersetzen; der Text des Kerns ist ein Ersatz und
für „Details“: die Phase — `Event.stage`; der Fehler — `ErrorInfo.kind`; der Grund einer Datei —
`FileResult.reasonCode`: `no_match` (keine verlässliche Übereinstimmung nach Ton), `short_audio`, `no_audio`,
`offline` (nicht auf der Festplatte, im Schnittprogramm offline), `no_path`, `not_found` (unter dem Pfad nicht gefunden),
`decode_error`, `conflict` (Überlagerung auf der eigenen Spur), `by_clock` (nach der Uhr, der Ton hat es nicht bestätigt),
`duplicate` (Kopie einer anderen Projektdatei auf einer gemeinsamen Spur mit ihr; in DRT werden keine Spuren hinzugefügt — die Kopie bleibt ohne Partner);
Konstanten — `api.Reason*`. Die Texte der CLI und des CSV-Berichts ändern sich nicht.

`Summary.duplicates` — Kopien von Dateien (gleicher Ton): in XML stehen sie zusammen mit dem Original, ihre Clips liegen bei
Überlagerung auf zusätzlichen Spuren (V/A mit einer Nummer nach der letzten); `Summary.timeShift` — von
`timeAudio` durch die Verschiebungsprüfung bestätigt (Musik).

Alle Typen sind Strukturen mit json-Tags (Wails baut daraus die TypeScript-Modelle). `Result` hält das
Ergebnis im Speicher: das Fenster bewahrt den Zeiger in seiner eigenen Struktur auf und gibt nur `Summary`/`Files` an JS weiter.

## Fortschritt — `Event`
`{stage, title, done, total, file}`; `done == total` — die Phase ist beendet, `total == 0` — ohne Zähler.
Phasen der Reihe nach: `search` (Mediensuche) → `record_time` (Kamerauhren) → `features` (Ton, je Datei:
`file`) → `pairs` (Paarvergleich, ~1 % — bis zu 100 Ereignisse) → `refine` → `solve` (3 Schritte) → `layout` →
`write`. Denselben Strom gibt `orcaton sync --progress json` aus (eine JSON-Zeile pro Ereignis, stderr).
Der Handler wird nacheinander, aber aus Worker-Goroutinen aufgerufen: nur `runtime.EventsEmit` und Rückkehr.

## Abbruch
Der `ctx` von `Sync` und `FindMedia`. Er wird zwischen den Phasen und innerhalb langer geprüft: Aufnahmezeit — je Datei,
Merkmale — ffmpeg-Prozesse werden beendet (`exec.CommandContext`), Paare — je Serie. Fehler der Art
`cancelled`, kein Ergebnis, das Projekt ändert sich nicht. Das Indizieren von Ordnern bei der Mediensuche (Durchlauf der Festplatte)
lässt sich noch nicht unterbrechen — nur zwischen den Suchen.

## Die Frage nach fehlenden Medien — `Asker`
`AskMissing(ctx, missing []MissingFile) (Answer, error)` wird aufgerufen, solange die Liste nicht leer ist.
Die Antwort: `{action: "folder", folder}` — dort suchen (wurde eine Datei angegeben, wird in ihrem Ordner gesucht), `"skip"` —
die übrigen kommen ganz ans Ende als „nicht auf der Festplatte“ (im Schnittprogramm offline), `"cancel"` — Fehler
`cancelled`. Der Aufruf blockiert: in Wails — ein Ereignis ans Fenster und Warten auf die Antwort aus einem Kanal:
```go
func (a *App) AskMissing(ctx context.Context, ms []api.MissingFile) (api.Answer, error) {
	runtime.EventsEmit(a.ctx, "media:missing", ms)
	select {
	case ans := <-a.answers: // die Methode App.Answer(ans) wird aus JS aufgerufen
		return ans, nil
	case <-ctx.Done():
		return api.Answer{}, ctx.Err()
	}
}
```
In der CLI steht an seiner Stelle das Terminal (`internal/cli/mediafind.go`, `termAsker`).

## Einstellungen — `orcaton.toml`
Fenster und `orcaton sync` lesen dieselben Dateien ([usage.de.md](usage.de.md#einstellungsdatei-orcatontoml)):
* `api.LoadSettings(project)` — Standardwerte ← Benutzerdatei (`api.UserSettingsPath()`) ← Datei
  neben dem Projekt (`api.ProjectSettingsPath(project)`; `""` — nur der Benutzer). Das Ergebnis ist
  `Settings{Options, Media{Map, Search, SearchDepth, KeepPaths, Originals}, Files}`;
  `Media.SearchRoots()` — Ordner für `FindRequest.Roots` (wie `--map ORDNER` + `--search`).
  `Map` → `OpenOptions.PathMap`, `KeepPaths` → `SaveOptions`, `Originals` → `SyncRequest`.
* Ein Fehler in einer Datei — `bad_options` (im Text Pfad und Schlüssel) und `DefaultSettings()`: das Fenster zeigt
  den Fehler und arbeitet mit den Standardwerten.
* `api.SaveSettings(path, s)` — schreibt in den Abschnitt `[sync]` die Werte, die von den Standardwerten abweichen (ein Schlüssel,
  der schon in der Datei steht, wird auch beim Standardwert aktualisiert); Kommentare, andere Abschnitte und nur für das Terminal
  gedachte Schlüssel (`progress`) bleiben unberührt; das Schreiben ist atomar; falsche Einstellungen — `bad_options`, die Datei
  ändert sich nicht. Wohin gespeichert wird (Benutzer / Projekt), entscheidet das Fenster.
* `MemLimitMB` steht in der Datei in GB (`mem-limit`), wie das Flag.

## Fehler
`api.KindOf(err)` → `cancelled` · `no_ffmpeg` · `not_found` · `bad_format` · `bad_options` ·
`bad_output` · `write` · `failed`; `api.Describe(err)` → `{kind, message, hint}` für das Fenster. Der Text
des Fehlers ist derselbe wie in der CLI. Fehlt FFmpeg bei der Synchronisation nach Medien — sofort `no_ffmpeg` (früher
bekam jede Datei einen „Dekodierfehler“).

## Threads
Ein `Project` — eine Operation zur Zeit (`FindMedia` setzt Pfade im Projekt ein). `Sync` ändert das Projekt nicht
(arbeitet mit einer Kopie), `Result.Save` kann wiederholt aufgerufen werden.
