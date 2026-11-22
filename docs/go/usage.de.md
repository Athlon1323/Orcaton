# Arbeiten im Terminal (`orcaton`)

[English](usage.md) · **Deutsch** · [Русский](usage.ru.md) · [ქართული](usage.ka.md)

Der Befehl heißt **`orcaton`** (oder `app-go/scripts/run.*` ohne Installation). Das Fenster kann dasselbe
([docs/gui/user.de.md](../gui/user.de.md)) — die Zuordnung von Schaltflächen und Optionen steht dort.

## Synchronisation
1. Ihr Schnittprogramm: eine Kamera oder ein Recorder — eine Spur; Timeline exportieren
   (*File → Export → Timeline…*) → **FCP 7 XML V5** (`.xml`) oder **DRT** (`.drt`).
2. Start (nach der Installation in einem **neuen** Terminalfenster; liegen die Medien woanders — `--map`, siehe unten):
   ```
   orcaton sync "Timeline 1.xml" "Timeline 1-out.xml" --report "Timeline 1-out.csv"
   orcaton sync "Timeline 1.drt" "Timeline 1-out.drt"          # DRT → DRT (der Bericht entsteht von selbst)
   orcaton sync "Timeline 1.drt" "out.drt" --map "D:\media\project"   # die Medien liegen jetzt hier (Suche nach Namen)
   ```
3. Ihr Schnittprogramm: das Ergebnis importieren (*File → Import → Timeline…*).

| Clip-Markierung | Bedeutung |
|---|---|
| keine Markierung | **nach Ton** synchronisiert (bildgenau) |
| **Mango** | **nach der Kamerauhr** gesetzt — der Ton hat es nicht bestätigt (≈1 s) |
| **Rose** | kein Partner gefunden — steht in der Reihenfolge der Aufnahme |

Die Reihenfolge der Dateien einer Kamera richtet sich nach der Nummer im Namen (MVI_0001, C0001, 00001…), danach nach der Aufnahmezeit.
Dateien ohne Partner stehen in dieser Reihenfolge ihrer eigenen Kamera neben ihren Nachbarn (auch innerhalb synchroner
Abschnitte); Dateien, die nicht auf der Festplatte liegen, kommen ganz ans Ende; Lücken gibt es nicht. In einer `.drt` ändern sich nur die Clip-Positionen, die Dauer und
der Name der Timeline; die Markierungen Rose/Mango werden noch nicht in die `.drt` geschrieben — die Status stehen in `*_report.csv`.

## Optionen von `sync`
| Option | Standard | Wofür |
|---|---|---|
| `--report FILE.csv` | — (bei `.drt` — `OUT_report.csv`) | Bericht zu jeder Datei |
| `--map ORDNER` / `--map "D:/=G:/"` | `$ORCATON_PATH_MAP` | wo die Medien auf diesem Rechner liegen: ein Ordner mit den Quellen (Suche nach Namen) und/oder Ersetzung des Pfadanfangs; `;` — mehrere |
| `--search ORDNER` | — | dasselbe wie `--map ORDNER`; mehrfach möglich |
| `--search-depth N` | `6` | wie viele Unterordner-Ebenen durchsucht werden |
| `-y`, `--yes` | aus | nicht nach nicht gefundenen Dateien fragen — sofort ans Ende der Timeline |
| `--keep-paths` | aus | mit `--map`/`--search`: von den neuen Pfaden lesen, die alten ins Ergebnis schreiben |
| `--originals ORDNER` | — | die Timeline enthält Proxys: wo die Originale der Kameras liegen (Aufnahmezeit); mehrfach möglich |
| `--unsynced` | `order_time` | `order_time` (in der Reihenfolge der Kamera, auch innerhalb von Abschnitten) · `edges` (nie in einen Abschnitt hinein; nach dem letzten Abschnitt — ans Ende) · `end` · `remove` |
| `--max-gap` / `--keep-gap` | `0` / `0` | welche Lücken entfernt und auf welche Länge sie verkürzt werden, Sek. |
| `--order-by` | `name` | Reihenfolge der Dateien einer Kamera: `name` — Nummer im Namen (MVI_0001, C0001, 00001…), danach Aufnahmezeit · `time` — Aufnahmezeit / TC, danach der Name |
| `--singles` | `sequential` | Dateien ohne Partner: `sequential` — laufen nicht gleichzeitig mit anderen Dateien ohne Partner nach Ton (sie stehen neben den Nachbarn ihrer Kamera, bei Bedarf wird Zeit eingefügt; der Synchronstand ändert sich nicht) · `stacked` — jede in der Reihenfolge ihrer Kamera, verschiedene Kameras dürfen gleichzeitig laufen |
| `--label` / `--time-label` | `Rose` / `Mango` | Farben der Markierungen |
| `--min-conf` / `--min-ratio` | `12` / `1.5` | Strenge des Tonvergleichs |
| `--no-auto-threshold` | — | die Schwelle nicht bis zum Rauschen des Projekts anheben |
| `--min-overlap` / `--max-offset` | `2` / — | minimale Überlappung und Suchfenster, Sek. |
| `--audio-mode` | `mix` | `mix` · `first_stream` · `first_channel` |
| `--no-file-order` / `--intra-lane` / `--no-record-time` | — | Kamerareihenfolge ausschalten / innerhalb einer Spur vergleichen / Uhren nicht verwenden |
| `-j N` | `0` (auto) | wie viele Dateien parallel dekodiert werden; auto — die Hälfte der Kerne (2…16), auf einer HDD — bis zu 4 |
| `--threads N` | `0` (alle Kerne) | Korrelations-Threads — weniger, wenn nebenbei Ihr Schnittprogramm läuft |
| `--mem-limit GB` | `0` (auto) | Speicherobergrenze: auto — ¾ des RAM, von 1 bis 7 GB; Minimum 0,25 GB |
| `--cache ORDNER` | System | Ordner des Merkmal-Caches |
| `--to xmeml\|drt` | nach Endung | Ausgabeformat |
| `--config DATEI` / `--no-config` | `orcaton.toml` | eigene Einstellungsdatei statt der `orcaton.toml` neben der Timeline / Einstellungsdateien nicht lesen ([unten](#einstellungsdatei-orcatontoml)) |
| `--brief` / `--json` | aus | Ergebnis in einer Zeile / als JSON |
| `--progress auto\|text\|json\|none` | `auto` | Fortschritt auf stderr: `auto` — als Zeile, nur wenn stderr ein Terminal ist und weder `--brief` noch `--json` gesetzt ist; `json` — ein Ereignis pro Zeile (`stage`, `title`, `done`, `total`, `file` — derselbe Strom wie beim Fenster, [api.de.md](api.de.md)) |

Die Korrelation der Paare nutzt alle Prozessorkerne (das Ergebnis hängt nicht von der Kernzahl ab).
Speicher: parallele Korrelationen sind durch die Obergrenze (`--mem-limit`) begrenzt, das Tonmerkmal wird
beim Dekodieren berechnet — ein Projekt mit rund 400 Dateien und 100 GB Medien kommt mit ~0,5 GB aus. Was gewählt wurde und
wie viel Speicher belegt ist — die Zeile `Resources:` im Ergebnis von `sync` (RAM → cap, threads, ffmpeg, Datenträgertyp,
FFmpeg-Version; Spitzenspeicher des Prozesses). `orcaton version` — Version des Programms und von FFmpeg (Pfad; älter als
4.4 — Warnung).

## Einstellungsdatei `orcaton.toml`
Die Standardwerte der `sync`-Optionen lassen sich in einer Datei ablegen — dieselbe Datei liest auch das Fenster
([docs/gui/user.de.md](../gui/user.de.md)). Schichten, jede folgende ist wichtiger:
1. die Standardwerte aus der Tabelle oben;
2. die **Benutzerdatei** — für alle Projekte: Windows `%AppData%\orcaton\orcaton.toml` · macOS
   `~/Library/Application Support/orcaton/orcaton.toml` · Linux `~/.config/orcaton/orcaton.toml`
   (`ORCATON_CONFIG` — eigener Pfad, `none` — nicht lesen);
3. **`orcaton.toml` neben der Timeline** (im Ordner der Eingabe-`.xml`/`.drt`) oder die Datei aus `--config`;
4. ein Kommandozeilen-Flag; bei `map` hat die Variable `ORCATON_PATH_MAP` Vorrang vor der Datei.
```
orcaton config "Timeline 1.drt"          # welche Dateien gelesen werden und was gilt
orcaton config --init                    # Vorlage der Benutzerdatei (alle Schlüssel auskommentiert)
orcaton config "Timeline 1.drt" --init   # Vorlage neben der Timeline
```
```toml
[sync]
map = 'D:\media\project'        # Windows-Pfade — in einfachen Anführungszeichen
min-conf = 10.0
jobs = 4                        # HDD
originals = ['Sony', 'D:/Originals']   # relative Pfade — ab dem Ordner dieser Datei
```
* Die Schlüssel sind die Namen der `sync`-Flags ohne `--` (`min_conf` geht auch): `map`, `search`, `search-depth`,
  `keep-paths`, `originals`, `unsynced`, `order-by`, `singles`, `label`, `time-label`, `min-conf`, `min-ratio`,
  `auto-threshold`, `min-overlap`, `max-offset` (0 — unbegrenzt), `audio-mode`, `file-order`,
  `intra-lane`, `record-time`, `max-gap`, `keep-gap`, `jobs`, `threads`, `mem-limit` (GB), `cache`,
  `progress`. Die Flags `--no-…` sind in der Datei `false` beim Schlüssel ohne `no-` (`record-time = false`). Eingabe, Ausgabe und
  Bericht sind bei jedem Lauf anders und gehören nicht in die Datei.
* Ein Tippfehler im Schlüssel, ein falscher Typ oder ein Wert außerhalb des Bereichs ist ein Fehler mit Datei- und Schlüsselname, Code 2
  (es wird nie stillschweigend der Standardwert genommen). Andere Abschnitte der Datei liest und ändert Orcaton nicht.
* `sync` gibt die Zeile `Settings: <Dateien>` aus, wenn eine Datei gelesen wurde.

## Weitere Befehle
| Befehl | Was er tut |
|---|---|
| `inspect FILE` | Übersicht: Sequenzen, Spuren, FPS, Markierungen, Ordner |
| `clips FILE [-t V1] [--csv OUT.csv]` | Liste der Clips (Sekunden und TC) |
| `convert IN OUT [--map …]` | XML ⇄ Modell ⇄ XML, DRT → XML, Pfade übertragen |
| `cache [info\|clear\|prune] [--older-than 30d] [--dry-run] [--json]` | der Merkmal-Cache (ohne Unterbefehl — `info`) |
| `probe MEDIA`, `offset A B` | Dateidaten; Verschiebung des Tons B gegenüber A |
| `config [PROJEKT] [--config DATEI] [--init]` | Einstellungsdateien `orcaton.toml` und die geltenden Werte; `--init` — eine Vorlage (überschreibt nie) |

Exit-Codes: `0` — Erfolg; `1` — Laufzeitfehler (z. B. das Ergebnis ließ sich nicht schreiben); `2` — Fehler bei Eingabe oder
Optionen, auch in `orcaton.toml` (Zahlen außerhalb des Bereichs — der Bereich steht in `--help`, z. B. `--jobs INTEGER [1<=x<=256]`);
`3` — FFmpeg nicht gefunden (bei der Synchronisation nach Medien — sofort, vor der Analyse); `130` — `sync` mit
Strg+C abgebrochen (ffmpeg-Prozesse werden beendet, keine Ausgabe geschrieben); `70` — interner Programmfehler (Meldung und Stack an den Entwickler schicken).

## Medien nicht am alten Ort
Eine Timeline speichert vollständige Pfade zu den Dateien. Wurde das Projekt auf einem anderen Rechner geschnitten oder die Medien
sind umgezogen, geben Sie den Ordner an, in dem sie jetzt liegen:
```
orcaton sync "Timeline 1.drt" "out.drt" --map "D:\media\project"
orcaton sync "Timeline 1.xml" "out.xml" --search "E:\Camera A" --search "F:\Sound"
```
* Dateien werden **nach Namen** gesucht (ohne Groß-/Kleinschreibung, NFC) in allen Unterordnern bis `--search-depth` (6 —
  genügt für `PRIVATE/M4ROOT/CLIP` bei Sony). Dateien, die am alten Ort liegen, bleiben unberührt.
* Gleiche Namen in verschiedenen Ordnern (`1/C0992.MP4` und `3/C0992.MP4`) werden anhand der Ordner am Ende des
  ursprünglichen Pfads unterschieden (`…/project/3/C0992.MP4` → `…/3/`). Lassen sie sich nicht unterscheiden, wird die Datei nicht ersetzt und
  eine Liste der Varianten ausgegeben — den Ordner genauer angeben.
* Die alte Form `--map "D:/media/project/=D:/media/project/"` (Ersetzen des Pfadanfangs) funktioniert wie bisher;
  beide Formen lassen sich mischen: `--map "D:/=G:/;E:\Sound"`.
* **Was nicht gefunden wurde** — im Terminal zeigt das Programm eine Liste und fragt: einen Ordner für die weitere Suche
  (Sie können ihn ins Fenster ziehen), **Enter** — überspringen, `q` — abbrechen. Übersprungene Dateien stehen ganz
  am Ende der Timeline mit ihrem alten Pfad — im Schnittprogramm sind sie offline und werden später neu verknüpft
  (*Relink Clips*). Ohne Terminal (Skript, `--yes`, `--brief`, `--json`) gibt es keine Frage — sie werden sofort übersprungen, die
  Liste erscheint als Warnung.
* Ins XML-Ergebnis kommen die gefundenen Pfade (Ihr Schnittprogramm sieht die Medien sofort; `--keep-paths` — die alten). In einer
  `.drt` ändern sich die Pfade nicht (nur Positionen werden gepatcht) — verknüpfen Sie die Medien nach dem Import im Schnittprogramm neu.
* Die Mediensuche gibt es bei `sync`.

## Cache
Das Tonmerkmal jeder Datei wird einmal berechnet und im Cache abgelegt: Windows
`%LOCALAPPDATA%\orcaton\cache` · macOS `~/Library/Caches/orcaton` · Linux `~/.cache/orcaton` (oder
`ORCATON_CACHE_DIR`, `--cache`). Wurde eine Synchronisation unterbrochen, nimmt ein Neustart Fertiges aus dem Cache.
Die Befehle funktionieren aus jedem Ordner; gelöscht werden nur Cache-Dateien von Orcaton:
```
orcaton cache                         # wo der Cache liegt und wie viel er belegt (= cache info)
orcaton cache clear                   # alles leeren (Merkmale werden beim nächsten sync neu berechnet)
orcaton cache prune --older-than 30d  # nur lange Ungenutztes und *.tmp-Reste
```

## Umgebungsvariablen
| Variable | Wofür |
|---|---|
| `ORCATON_PATH_MAP` | Standardwert von `--map` (Ordner und/oder Regeln) |
| `ORCATON_CACHE_DIR` | Cache-Ordner |
| `ORCATON_CONFIG` | Benutzer-Einstellungsdatei statt des Systemordners; `none` — nicht lesen |
| `ORCATON_FFMPEG_DIR` | Ordner mit ffmpeg/ffprobe — hat Vorrang vor allem; danach: Programmordner, PATH, übliche Ordner |
| `ORCATON_CPUPROFILE` | (für Entwickler) CPU-Profil für `go tool pprof` schreiben |

## Bericht
`--report` schreibt eine CSV (`;`, UTF-8 mit BOM — öffnet sich in Excel): zu jeder Datei — Status, mit wem sie
übereinstimmt, Sicherheit (z), Aufnahmezeit und der Grund, falls sie nicht synchronisiert ist. Für die Ausgabe `.drt` wird der Bericht
`OUT_report.csv` immer erstellt — Markierungen werden noch nicht in die `.drt` geschrieben.

## Fragen und Lösungen
**DRT: wo sind die Markierungen Rose/Mango?** In der `.drt` werden sie noch nicht geschrieben — die Status stehen in `*_report.csv`. Brauchen Sie
Markierungen auf der Timeline, wählen Sie die Ausgabe `.xml`.

**DRT aus XML?** Geht nicht: eine `.drt` entsteht durch Patchen der Quell-`.drt`. Exportieren Sie die Timeline als `.drt`.
DRT → XML funktioniert.

**Die Medien liegen auf einem anderen Laufwerk, das Ergebnis wird auf dem ursprünglichen Rechner gebraucht.** `--map "D:/Project/=G:/Project/"`
(oder `--map ORDNER`) + `--keep-paths` — von den neuen Pfaden lesen, die alten ins Ergebnis schreiben.

**Die Timeline enthält Proxys (ProRes aus Ihrem Schnittprogramm).** Proxys haben keine Aufnahmezeit (ihre ist die Renderzeit) —
geben Sie die Originale an: `--originals "G:\Sony_card"`. Ohne Zeit werden Dateien ohne Partner schlechter platziert. Die Aufnahmezeit
wird auch aus dem Dateinamen gelesen (`DJI_20261003231144_0085_D.MP4`; Recorder `031026_191644_TrMic.WAV` — die Reihenfolge des
Datums wird anhand der anderen Dateien gewählt) und aus der WAV (BWF) des Recorders.

**Eine Datei „fand keinen Partner“ (Rose).** Ursachen: die Kamera hörte keinen gemeinsamen Ton, die Überlappung mit den anderen ist
kleiner als 2 s, Stille. Sind Originale vorhanden, platziert `--originals` sie nach der Kamerauhr (Mango).
Der Grund jeder Datei steht im Bericht.

**Wie viele Spuren sind möglich?** Keine Begrenzung; die Zeit wächst mit der Dateizahl (Hunderte — Minuten).

**Beim ersten Mal langsam.** FFmpeg liest die Dateien vollständig; die Zahl der parallelen Dekoder richtet sich nach
Kernen und Datenträgertyp (HDD — bis zu 4). Läuft nebenbei Ihr Schnittprogramm — `--threads` kleiner wählen.
