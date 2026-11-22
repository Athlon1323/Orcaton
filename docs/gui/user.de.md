# Orcaton mit Fenster — Benutzerhandbuch

[English](user.md) · **Deutsch** · [Русский](user.ru.md) · [ქართული](user.ka.md)

> **Status: Alpha.** Das Fenster funktioniert bereits: Timeline öffnen → Medien
> finden → synchronisieren → speichern. Mit *(später)* markiert ist, was es im Fenster noch nicht gibt.
> Dasselbe kann das Terminal — [docs/go/usage.de.md](../go/usage.de.md). **Das Terminal bleibt**:
> Fenster und `orcaton sync` laufen auf demselben Kern und liefern dasselbe Ergebnis.

## Das Fenster öffnen
Orcaton mit Fenster ist ein **eigenständiges Desktop-Programm**: Es öffnet ein eigenes Programmfenster
(Symbol in der Taskleiste, Alt+Tab, die üblichen Schaltflächen Minimieren/Schließen). Es ist **keine
Website und keine Browserseite**: Alles, was das Programm zeigt, ist in seinem eigenen Fenster. Wenn Sie
Orcaton im Browser öffnen (zum Beispiel die Adresse `http://localhost:…` aus der Terminalausgabe während
der Entwicklung), zeigt die Seite den Hinweis „Orcaton als App öffnen“ und tut nichts: Aus dem Browser
lassen sich weder Dateien auf dem Computer wählen noch eine Synchronisierung starten.

**Installation** — ein Befehl im Terminal im Projektordner; Schritt für Schritt für Windows, macOS und
Ubuntu — [docs/quickstart.de.md](../quickstart.de.md):
```powershell
powershell -ExecutionPolicy Bypass -File scripts\install.ps1 -Run      # Windows 10/11
```
```bash
bash scripts/install.sh --run                                        # macOS, Ubuntu / Linux
```
Der Befehl installiert, was fehlt (Go, Node.js, Wails, FFmpeg; unter Linux die Fensterbibliotheken), baut
Orcaton aus dem Quellcode, installiert es für Ihren Benutzer und öffnet es. Ein erneuter Aufruf
aktualisiert das Programm; die Einstellungen bleiben.

**Starten** — wie jedes Programm, ohne Terminal:

| System | Wo Orcaton ist | Programmdatei |
|---|---|---|
| Windows | **Orcaton** im Startmenü und auf dem Desktop | `%LOCALAPPDATA%\Programs\Orcaton\orcaton-gui.exe` |
| macOS | Launchpad / Spotlight „Orcaton“, Finder → `~/Applications` | `~/Applications/Orcaton.app`; aus der `.dmg` — `/Applications/Orcaton.app` |
| Ubuntu / Linux | Anwendungsmenü → **Orcaton** | `~/.local/share/orcaton/orcaton-gui`; aus dem Paket — `/usr/bin/orcaton-gui` |

**Deinstallieren** — Windows: Einstellungen → Apps → Orcaton (oder `scripts\uninstall.ps1`); macOS, Linux:
`bash scripts/uninstall.sh`. Entfernt wird nur Orcaton; Go, Node.js, Wails und FFmpeg bleiben.

## Sprache
Unten in der linken Spalte (Globus-Symbol) — **English**, **Deutsch**, **Русский**, **ქართული**. Die
Beschriftungen wechseln sofort, ohne Neustart; die Wahl bleibt zwischen den Starts erhalten (Datei
`gui.json` neben Ihrer `orcaton.toml`). Standard ist English. Unten heißen die Schaltflächen wie im
deutschen Fenster.

## Dateien und Ordner wählen Sie
Das Programm wählt nichts für Sie: Jeder Pfad ist ein **Systemdialog** von Windows / macOS / Linux
(derselbe wie in anderen Programmen), in dem Sie durch die Ordner gehen und das Gewünschte wählen.
Dialog geschlossen („Abbrechen“) — es passiert nichts.

| Wo im Fenster | Was sich öffnet | Was Sie wählen |
|---|---|---|
| Import → Feld **XML / DRT hierher ziehen oder durchsuchen…** (Klick auf das ganze Feld), großes Feld in der Mitte des leeren Fensters, **Ersetzen…** | Dialog „Öffnen“ mit Filter `*.xml; *.drt` | die Timeline-Datei (den Ordner — über den Pfad im Dialog); die Datei lässt sich auch ins Fenster ziehen |
| Import → **+ Ordner hinzufügen** | Ordnerdialog | den Ordner mit dem Quellmaterial; Suche nach Dateinamen in allen Unterordnern |
| **N Dateien nicht gefunden → Suchen…** und der Dialog „Mediendateien nicht gefunden“ → **Ordner wählen…** | Ordnerdialog | den Ordner mit den fehlenden Dateien („Überspringen“ — nicht suchen, „Abbrechen“ — abbrechen) |
| **Timeline speichern** (oben nach der Synchronisierung) | Dialog „Speichern unter“ | Ordner und Namen der Ausgabe; Format wie die Eingabe (`.xml` → `.xml`, `.drt` → `.drt`); vorgeschlagen wird `<Eingabe> - synced.<ext>` neben der Eingabe — Ordner und Name lassen sich ändern |
| **Bericht speichern (CSV)** | Dialog „Speichern unter“ | Ordner und Namen des Berichts; vorgeschlagen wird `<Ausgabe>_report.csv` |
| Einstellungen → **Cache-Ordner → Wählen…** | Ordnerdialog | den Ordner für den Audio-Cache (leer — Systemordner) |
| Meldung **Im Ordner zeigen**, Entwicklerwerkzeuge → **Protokollordner öffnen** | Explorer / Finder | — (zeigt nur den Ordner) |

Ohne Dialog nimmt das Programm Pfade nur aus dem, was Sie selbst vorher angegeben haben: Suchordner und
Einstellungen aus `orcaton.toml` (Einstellungen, `orcaton config`) und die in der Timeline
gespeicherten Medienpfade.

## Kurz: vom Schnittprogramm und zurück
1. **Schnittprogramm:** eine Kamera oder ein Rekorder — eine Spur; Timeline exportieren
   (*Datei → Exportieren → Timeline…*) → **FCP 7 XML V5** (`.xml`) oder **DRT** (`.drt`).
2. **Orcaton:** die Datei ins Fenster ziehen oder **Import → durchsuchen…** drücken und sie im Dialog
   wählen. Werden Medien teilweise nicht gefunden — **+ Ordner hinzufügen** und im Dialog den Ordner mit
   dem Quellmaterial angeben.
3. **Synchronisieren** drücken (grüne Schaltfläche unten links). Den Fortschritt oben verfolgen; **■**
   (Synchronisierung stoppen) — abbrechen (es ändert sich nichts).
4. **Timeline speichern** → im Dialog Ordner und Namen wählen (vorgeschlagen `Timeline 1 - synced.xml`
   oder `.drt` — im Format der Eingabe, daneben); bei Bedarf **Bericht speichern (CSV)**.
5. **Schnittprogramm:** die gespeicherte Datei importieren (*Datei → Importieren → Timeline…*).

## Das Fenster
```
┌──────── Fortschritt: Schritt, Datei, %, vergangen / verbleibend · danach Ergebnis und Speichern ───────┐
│ Über                   │  Timeline | Dateien     Vorher | Nachher     Zoom · Einpassen               │
│ Einstellungen          │                                                                            │
│ Import  ◀ Standard     │  Timeline: eine Zeile je Kamera, Clips in der Farbe ihres Zustands          │
│ ── Entwicklerwerkzeuge │  oder die Dateitabelle                                                     │
│ Deutsch ▾              │                                                                            │
│ [ Synchronisieren ] ■  │  Legende-Filter: Per Ton abgeglichen · Nach Kamerauhr · Kein Treffer ·      │
│                        │  Nicht auf dem Datenträger                                                 │
└────────────────────────┴────────────────────────────────────────────────────────────────────────────┘
```
* **Linke Spalte** — Bereiche: **Import** (was synchronisiert wird), **Einstellungen**, **Über**,
  **Entwicklerwerkzeuge** (Fehlersuche; sichtbar, solange das Programm Pre-Alpha ist). Unten — die Sprache
  und **Synchronisieren** / **■**.
* **Rechts** — zwei Ansichten mit Umschalter: **Timeline** (eine Zeile je Kamera; Kamera = Spur + Ordner,
  z. B. „6D · V1+A1“) und **Dateien** (Tabelle). **Vorher / Nachher** — ursprüngliche Positionen oder das
  Ergebnis. Vor der Synchronisierung öffnet sich **Dateien**, danach — **Timeline · Nachher**; danach merkt
  sich das Fenster Ihre Wahl.

## Import: Timeline und Medien
* Datei: Name, Format, fps, Spuren, Clips, Dateien. **Ersetzen…** — eine andere öffnen.
* **Medienordner** — wo nach Quellmaterial gesucht wird, wenn die Pfade in der Timeline nicht gefunden
  werden (Projekt von einem anderen Computer, Umzug auf ein anderes Laufwerk). Suche nach Dateinamen in
  allen Unterordnern (bis zu 6 Ebenen, Sony `PRIVATE/M4ROOT/CLIP` wird gefunden). Gleiche Namen in
  verschiedenen Ordnern werden anhand der Ordner am Ende des ursprünglichen Pfads unterschieden; geht das
  nicht, zeigt das Fenster die Möglichkeiten.
* **Kamera-Originale** — wenn auf der Timeline Proxys liegen (ProRes aus dem Schnittprogramm): Ordner mit
  den Kamera-Originalen, um die Aufnahmezeit zu übernehmen (Proxys haben sie nicht). *(später im Fenster;
  vorerst — der Schlüssel `originals` in `orcaton.toml`, den das Fenster liest.)*
* **Medienstatus:** gefunden / nicht gefunden / offline. Nicht gefunden > 0 — gelber Kasten mit **Suchen…**.

**Der Dialog „N Mediendateien nicht gefunden“** (erscheint auch vor der Synchronisierung, wenn etwas
fehlt): **Ordner wählen…** — in einem weiteren Ordner suchen; **Überspringen** — der Rest kommt ganz ans
Ende der Timeline als „nicht auf dem Datenträger“ (im Schnittprogramm offline, später neu verknüpfen);
**Abbrechen** — nicht starten.

## Ergebnis
Oben — „✓ Fertig in 37 s“ und Zähler. Jede Datei hat eine Farbe **und** ein Symbol:

| Im Fenster | Im Schnittprogramm nach dem Import | Bedeutung |
|---|---|---|
| ✓ Per Ton abgeglichen (grün) | ohne Markierung | **per Ton** synchronisiert, bildgenau |
| ◷ Nach Kamerauhr (bernstein) | Markierung **Mango** | **nach der Kamerauhr** platziert — der Ton hat es nicht bestätigt (≈ 1 s) |
| ✕ Kein Treffer (rosa) | Markierung **Rose** | kein Partner gefunden — steht nach Aufnahmezeit |
| ⌀ Nicht auf dem Datenträger (gestrichelt) | Offline-Clip am Ende | die Datei ist nicht auf dem Datenträger |

* Reihenfolge der Dateien einer Kamera — nach der Nummer im Namen (MVI_0001, C0001, 00001…), dann nach
  Aufnahmezeit; Nummern verschiedener Kameras werden nicht verglichen (6D und 6D2 zählen getrennt).
* Dateien ohne Treffer laufen nie gleichzeitig mit anderen Dateien ohne Tontreffer (Einstellung
  **Anordnung ohne Treffer**, Standard *Nacheinander*): Sie rücken neben ihre Kameranachbarn, bei Bedarf
  wird Zeit eingefügt; synchrone Blöcke bleiben unverändert. *Nach Kamera gestapelt* — wie früher.
* Dateien ohne Treffer — in dieser Reihenfolge ihrer Kamera, neben ihren Nachbarn (auch innerhalb eines
  synchronen Blocks); ganz ans Ende kommen nur Dateien, die nicht auf dem Datenträger sind (vor der
  Synchronisierung fragt das Fenster, wo sie zu suchen sind: **Ordner wählen…** oder **Überspringen** —
  dann ans Ende); keine Lücken.
* Klick auf eine Datei — eine Karte: Kamera, Position (TC), Dauer, Partner, Konfidenz, Aufnahmezeit,
  **Grund** (warum nicht per Ton).
* **Dateien**: Sortierung nach Spalten, Suche nach Namen, Filter nach Ergebnis — beginnen Sie mit
  **Kein Treffer**.
* **Synchronisieren** erneut — ein neuer Lauf mit den aktuellen Einstellungen (die Wiederholung ist schnell:
  der Ton kommt aus dem Cache).

## Speichern
* **Timeline speichern** — im Format der Eingabe, ohne Konvertierung: geöffnete `.xml` wird als `.xml`
  gespeichert, geöffnete `.drt` — als `.drt` (es ändern sich nur Clip-Positionen, Dauer und der Name der
  Timeline, alles andere bleibt). DRT in XML umwandeln geht im Terminal (`orcaton sync in.drt out.xml`).
  Ordner und Namen wählen Sie im Dialog „Speichern unter“; vorgeschlagen wird `<Eingabe> - synced.<ext>`
  neben der Eingabe. Der Schalter **Ursprüngliche Medienpfade behalten** (Einstellungen) — die
  ursprünglichen Pfade schreiben (wenn das Ergebnis auf dem ursprünglichen Computer geöffnet wird).
* In `.drt` werden die Markierungen Rose/Mango noch nicht geschrieben — das Fenster bietet an, den Bericht
  zu speichern.
* **Bericht speichern (CSV)** — eine Tabelle für jede Datei (öffnet sich in Excel). **Im Ordner zeigen** —
  den Ordner öffnen.

## Einstellungen
Jedes Feld entspricht einer Terminal-Option (Hinweis am Feld). **Auf Standard zurücksetzen** — alles
zurück. Ein ungültiger Wert wird unter dem Feld markiert. Einstellungen bleiben zwischen den Starts
erhalten — sofort nach der Änderung, in Ihrer Datei `orcaton.toml` (der Pfad steht unten im Bereich).

| Gruppe · Feld im Fenster | Option von `orcaton sync` | Standard |
|---|---|---|
| Dateien ohne Treffer · Platzierung | `--unsynced order_time\|edges\|end\|remove` | order_time (In Kamerareihenfolge) |
| Dateien ohne Treffer · Kamerareihenfolge | `--order-by name\|time` | name (Zuerst Dateinummer) |
| Dateien ohne Treffer · Anordnung ohne Treffer | `--singles sequential\|stacked` | sequential (Nacheinander) |
| Dateien ohne Treffer · Markierung / Markierung „nur Uhr“ | `--label`, `--time-label` | Rose, Mango |
| Tonabgleich · Tonquelle | `--audio-mode mix\|first_stream\|first_channel` | mix (Alle Kanäle mischen) |
| Tonabgleich · Min. Konfidenz / Schwelle an Projektrauschen anheben | `--min-conf`, `--no-auto-threshold` | 12, an |
| Tonabgleich · Min. Verhältnis / Min. Überlappung / Max. Versatz | `--min-ratio`, `--min-overlap`, `--max-offset` | 1.5, 2 s, ohne Grenze |
| Kamerauhr · Aufnahmezeit verwenden / Dateireihenfolge innerhalb einer Kamera beibehalten / Clips auf derselben Spur vergleichen | `--no-record-time`, `--no-file-order`, `--intra-lane` | an, an, aus |
| Lücken · Lücken entfernen länger als / Behalten | `--max-gap`, `--keep-gap` | 0 (alle), 0 |
| Leistung · Parallele Decoder / Threads / Speichergrenze | `-j`, `--threads`, `--mem-limit` | auto: ½ der Kerne (HDD ≤ 4), alle Kerne, ¾ des RAM 1…7 GB |
| Leistung · Cache-Ordner | `--cache` | System |
| Import · Medienordner | `--map ORDNER`, `--search` | — |
| Import · Kamera-Originale | `--originals` | — |
| Speichern · Ursprüngliche Medienpfade behalten | `--keep-paths` | aus |
| Bericht speichern | `--report` | für `.drt` — immer |

Einstellungen von Fenster und Terminal sind gemeinsam: Ihre `orcaton.toml` und die neben der Timeline
([usage.de.md](../go/usage.de.md#einstellungsdatei-orcatontoml)); was im Fenster geändert wird, gilt
auch für `orcaton sync`. Die Sprache des Fensters ist keine Terminal-Einstellung — sie steht in `gui.json`.

## Über und Entwicklerwerkzeuge
* **Über:** Programmversion, FFmpeg (Pfad und Version; Warnung, wenn älter als 4.4), Audio-Cache (Ordner, Größe, **Cache leeren**),
  Lizenzen. macOS: **Befehlszeile** — **Befehlszeilen-Tool installieren** / **Befehlszeilen-Tool entfernen**: der Befehl
  `orcaton` für das Terminal (`/usr/local/bin/orcaton`, ein Verweis auf das Programm in Orcaton.app; macOS fragt nach dem
  Passwort). Verfügbar, wenn Orcaton aus dem Ordner „Programme“ gestartet ist.
* **Entwicklerwerkzeuge:** Ereignisprotokoll der Synchronisierung (kopieren), Protokollordner des Fensters
  (`%LocalAppData%\orcaton\logs`), Cache.

## Nur im Terminal
`inspect`, `clips`, `convert`, `probe`, `offset`, `config`, `cache`, `--json` / `--progress json` für Skripte — [docs/go/usage.de.md](../go/usage.de.md).

## Fragen
**Das Fenster meldet „FFmpeg ist nicht installiert“.** Orcaton liest den Ton mit FFmpeg — einem freien
Programm, das separat installiert wird. Drücken Sie **FFmpeg installieren**: Das Fenster zeigt zuerst den
genauen Befehl und installiert FFmpeg mit der Paketverwaltung des Systems — winget unter Windows, Homebrew
unter macOS, apt / dnf / pacman / zypper unter Linux (fragt nach Ihrem Passwort). Danach prüft es von selbst
erneut. Wird die Schaltfläche nicht angeboten (kein winget, kein Homebrew oder keine grafische
Passwortabfrage), führen Sie den angezeigten Befehl im Terminal aus und drücken Sie **Erneut prüfen**; auf
einem Mac ohne Homebrew installieren Sie zuerst Homebrew (brew.sh). Mit der Variable `ORCATON_FFMPEG_DIR`
können Sie Orcaton auch auf einen beliebigen Ordner mit FFmpeg verweisen. Jedes FFmpeg ab 4.4 funktioniert;
bei einer älteren Version zeigt das Fenster eine Warnung. Eine Timeline lässt sich ohne FFmpeg öffnen,
synchronisieren — nicht. Andere Fehler bei Installation und Start —
[quickstart.de.md](../quickstart.de.md#wenn-etwas-schiefgeht).

**Beim ersten Mal dauert es lange.** Der Ton jeder Datei wird einmal vollständig gelesen, danach kommt er
aus dem Cache. Läuft parallel das Schnittprogramm — **Threads** verringern.

**Viele „Kein Treffer“.** Sehen Sie sich den Grund in der Karte an: kein gemeinsamer Ton, Überlappung
< 2 s, Stille. Für Proxys **Kamera-Originale** angeben — die Dateien werden nach der Kamerauhr platziert.

*(später)* zuletzt geöffnete Projekte auf dem leeren Bildschirm, echte Wellenform auf den Clips.
