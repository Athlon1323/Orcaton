# Orcaton — Installation und erster Start (Windows, macOS, Ubuntu)

[English](quickstart.md) · **Deutsch** · [Русский](quickstart.ru.md) · [ქართული](quickstart.ka.md)

Orcaton ist ein **Programm mit eigenem Fenster**: Es öffnet sich wie jede andere Anwendung (eigenes
Fenster, Symbol in der Taskleiste oder im Dock). Ein Webbrowser wird nicht benötigt. Alles Folgende geht
Schritt für Schritt und ohne Terminal-Erfahrung: Das Terminal brauchen Sie **ein einziges Mal** — um
einen Befehl einzufügen. Danach startet Orcaton über eine Verknüpfung.

Sie brauchen: den Projektordner auf Ihrem Computer, eine Internetverbindung während der Installation,
10–20 Minuten und ca. 3 GB freien Speicher (Compiler und Bibliotheken für den Build). Installationsprogramme
können um Erlaubnis bitten: Windows — das Fenster „Möchten Sie zulassen, dass durch diese App Änderungen
an Ihrem Gerät vorgenommen werden?“ → **Ja**; macOS und Ubuntu — das Passwort Ihres Benutzerkontos
(die Zeichen werden beim Tippen nicht angezeigt — das ist normal). Den Installationsbefehl erneut
auszuführen ist unbedenklich: Was schon installiert ist, wird nicht noch einmal installiert.

> Die Installationsskripte melden ihren Fortschritt **auf Russisch** (zum Beispiel
> `== 2/3 Сборка ==`, am Ende `Готово` — „Fertig“). Lesen müssen Sie das nicht: Warten
> Sie einfach, bis sich das Orcaton-Fenster öffnet.

## Windows 10 / 11
1. **PowerShell im Projektordner öffnen.** Öffnen Sie im Explorer den Projektordner, Rechtsklick auf
   eine freie Stelle → **In Terminal öffnen** (Windows 11). In Windows 10 — **Umschalt** gedrückt halten,
   Rechtsklick → **PowerShell-Fenster hier öffnen**. Die Eingabezeile muss den Pfad zum Projektordner
   zeigen, zum Beispiel `PS D:\Projects\Orcaton>`.
2. **Prüfen, was schon installiert ist.** Befehl einfügen (Rechtsklick im Fenster fügt ein) und **Enter** drücken:
   ```powershell
   powershell -ExecutionPolicy Bypass -File scripts\install.ps1
   ```
   Das Skript installiert selbst nichts: Es prüft nur Go, Node.js, WebView2 und die Wails CLI. Fehlt etwas,
   gibt es die Befehle aus. Führen Sie sie **einzeln** aus (erscheinen Fenster von Installationsprogrammen — **Ja**):
   ```powershell
   winget install --id GoLang.Go -e
   winget install --id OpenJS.NodeJS.LTS -e
   winget install --id Microsoft.EdgeWebView2Runtime -e
   go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0
   ```
   Öffnen Sie dann ein **neues** PowerShell-Fenster im Projektordner und führen Sie den Befehl aus Schritt 2 erneut aus.
   FFmpeg ist optional: Fehlt es, bietet das Fenster beim ersten Start die Installation an.
3. **Installieren und bauen.** Sind alle Werkzeuge vorhanden, führen Sie aus:
   ```powershell
   powershell -ExecutionPolicy Bypass -File scripts\install.ps1 -Run
   ```
   Das Skript baut das Programm, legt die Verknüpfungen an und öffnet das Fenster. Es erscheinen die Zeilen
   `== 2/3 Сборка ==` (Bauen) und `== 3/3 Установка ==` (Installation).
4. **Fertig**, wenn sich das Fenster **Orcaton** öffnet und im Terminal ein grünes `Готово` („Fertig“)
   steht. Die Verknüpfung **Orcaton** liegt jetzt auf dem Desktop und im Startmenü; das Terminal können
   Sie schließen. Ab jetzt — nur noch die Verknüpfung.

## macOS (Apple Silicon und Intel)
1. **Homebrew** (Paketverwaltung; überspringen, wenn bereits vorhanden). Öffnen Sie das **Terminal**
   (Spotlight: ⌘ + Leertaste → „Terminal“) und führen Sie den Befehl von <https://brew.sh> aus (eine
   Zeile, fragt nach dem Mac-Passwort). Am Ende zeigt Homebrew „Next steps“ — führen Sie die beiden
   angezeigten Befehle `echo …` und `eval …` aus.
2. **Terminal im Projektordner:** Tippen Sie `cd ` (mit Leerzeichen), ziehen Sie den Projektordner aus
   dem Finder in das Terminal-Fenster und drücken Sie **Enter**.
3. **Installationsbefehl:**
   ```bash
   bash scripts/install.sh --run
   ```
   Bietet macOS an, die **Befehlszeilentools (Command Line Tools)** zu installieren — „Installieren“
   klicken, das Ende abwarten und **denselben Befehl noch einmal ausführen**.
4. **Fertig**, wenn sich das Fenster **Orcaton** öffnet. Das Programm liegt in **~/Applications**
   (Programme des Benutzers): Finden Sie es über Launchpad oder Spotlight („Orcaton“) und behalten Sie es
   im Dock (Rechtsklick auf das Symbol → Optionen → Im Dock behalten).

### macOS: fertiges Paket (.dmg)
Wenn Sie Orcaton als Datei `Orcaton-<Version>-macos-universal.dmg` erhalten haben (ein Paket für Apple
Silicon und Intel; macOS 12 Monterey oder neuer), muss nichts gebaut oder im Terminal installiert werden:
1. **Programm kopieren.** Doppelklick auf die `.dmg` → ein Fenster mit **Orcaton** und dem Ordner
   **Programme** (Applications) öffnet sich → **Orcaton** auf **Programme** ziehen. Danach das Laufwerk
   auswerfen (Finder-Seitenleiste → ⏏).
2. **Erster Start.** Orcaton ist kostenlos und nicht mit einem kostenpflichtigen Apple-Zertifikat
   signiert, deshalb stoppt macOS den ersten Start. Orcaton aus „Programme“ öffnen → Meldung „Apple konnte
   nicht überprüfen …“ (oder „kann nicht geöffnet werden“) → **Fertig** / **OK**. **Systemeinstellungen →
   Datenschutz & Sicherheit** öffnen, nach unten zu „Orcaton wurde blockiert …“ scrollen → **Dennoch
   öffnen** → Mac-Passwort → **Öffnen**. Unter macOS 12–14 genügt: Rechtsklick auf Orcaton → **Öffnen** →
   **Öffnen**. Das ist einmal nötig (und erneut nach einer neuen Version).
3. **FFmpeg.** Zeigt das Fenster **FFmpeg ist nicht installiert** — **FFmpeg installieren** drücken (nutzt
   Homebrew, <https://brew.sh>; ohne Homebrew zeigt das Fenster, was zu tun ist).
4. **Befehl im Terminal (optional):** **Über → Befehlszeile → Befehlszeilen-Tool installieren** (macOS
   fragt nach Ihrem Passwort). Danach funktioniert `orcaton` in einem neuen Terminal-Fenster.

Aktualisieren — das neue Orcaton auf „Programme“ ziehen → **Ersetzen**. Datei prüfen: im Terminal muss
`shasum -a 256 Orcaton-…dmg` denselben Wert ausgeben wie die Datei `.sha256` daneben.

## Ubuntu 22.04 / 24.04 (und andere Linux)
1. **Terminal im Projektordner:** Öffnen Sie in „Dateien“ den Projektordner, Rechtsklick auf eine freie
   Stelle → **Im Terminal öffnen**.
2. **Installationsbefehl** (fragt nach Ihrem Passwort — die Zeichen sind nicht sichtbar, das ist normal):
   ```bash
   bash scripts/install.sh --run
   ```
   Das Skript installiert die Fensterbibliotheken (GTK, WebKitGTK), FFmpeg, Go, Node.js und Wails, baut
   das Programm und fügt eine Verknüpfung hinzu.
3. **Fertig**, wenn sich das Fenster **Orcaton** öffnet. Die Verknüpfung ist im Anwendungsmenü
   (Super-Taste → „Orcaton“); Sie können sie zu den Favoriten hinzufügen (Rechtsklick → Zu Favoriten
   hinzufügen).
Fedora und Arch — derselbe Befehl (das Skript verwendet `dnf` / `pacman`).

### Linux: fertiges Paket (.deb / .rpm)
Wenn Sie Orcaton als Paketdatei erhalten haben, muss nichts gebaut werden. Die Paketverwaltung installiert
Orcaton zusammen mit allem Nötigen: den Fensterbibliotheken (GTK 3, WebKitGTK 4.1) und **FFmpeg** aus Ihrer
Distribution (FFmpeg bleibt ein eigenes Paket unter seiner eigenen Lizenz). Öffnen Sie ein Terminal im
Ordner mit der Datei und führen Sie aus (fragt nach Ihrem Passwort):

| System | Datei | Befehl |
|---|---|---|
| Ubuntu 22.04 / 24.04, Debian 12, Linux Mint 21+ | `orcaton_<Version>-1_amd64.deb` | `sudo apt install ./orcaton_<Version>-1_amd64.deb` |
| Fedora 39 oder neuer | `orcaton-<Version>-1.x86_64.rpm` | `sudo dnf install ./orcaton-<Version>-1.x86_64.rpm` |

* `./` am Anfang des Dateinamens ist nötig — sonst sucht apt / dnf das Paket im Internet.
* Orcaton ist nicht signiert: lehnt dnf wegen fehlender Signatur ab, `--nogpgcheck` zum Befehl hinzufügen.
* Danach: Anwendungsmenü → **Orcaton**; im Terminal — der Befehl `orcaton`.
* Wurde Orcaton früher mit `bash scripts/install.sh` installiert, diese Kopie zuerst entfernen:
  `bash scripts/uninstall.sh` — sonst öffnet das Menü weiter die alte.
* Aktualisieren — derselbe Befehl mit der neuen Datei. Datei prüfen: `sha256sum -c <Datei>.sha256`.

## Erste Synchronisierung
0. **Sprache des Fensters:** Unten in der linken Spalte (Globus-Symbol) **Deutsch** wählen — die
   Beschriftungen wechseln sofort, die Wahl wird gespeichert. Die Schaltflächen unten heißen so wie im
   deutschen Fenster.
1. **Schnittprogramm:** jede Kamera / jeder Rekorder auf einer eigenen Spur; Timeline exportieren
   (*Datei → Exportieren → Timeline…*) → **DRT (.drt)** oder **FCP 7 XML V5 (.xml)**.
2. **Orcaton:** Klicken Sie auf das Feld **„XML / DRT hierher ziehen oder durchsuchen…“** — der übliche
   Dateidialog öffnet sich; wählen Sie die exportierte Timeline (oder ziehen Sie die Datei einfach in das
   Orcaton-Fenster).
3. Liegen die Medien nicht mehr dort, wo sie beim Schnitt waren — **+ Ordner hinzufügen** und den Ordner
   mit dem Quellmaterial wählen (Suche nach Namen in allen Unterordnern). Das geht auch später: Vor der
   Synchronisierung fragt Orcaton von selbst.
4. Drücken Sie das grüne **Synchronisieren** (unten links). Oben — der Fortschritt; **■** — stoppen.
5. **Timeline speichern** (oben) — Ordner und Namen im Dialog „Speichern unter“ wählen.
6. **Schnittprogramm:** die gespeicherte Datei importieren (*Datei → Importieren → Timeline…*). Ohne
   Markierung — per Ton abgeglichen, **Mango** — nach Kamerauhr, **Rose** — kein Treffer gefunden.

Jede Schaltfläche im Detail — [docs/gui/user.de.md](gui/user.de.md).

## Programm aktualisieren (nach Änderungen am Code)
Führen Sie denselben Installationsbefehl im Projektordner erneut aus: Er baut Orcaton neu und ersetzt
die installierte Kopie (vorher das Orcaton-Fenster schließen). Die Einstellungen bleiben erhalten.

## Deinstallieren
| System | Wie |
|---|---|
| Windows | Einstellungen → Apps → **Orcaton** → Deinstallieren, oder `powershell -ExecutionPolicy Bypass -File scripts\uninstall.ps1` |
| macOS / Linux | `bash scripts/uninstall.sh` |
| Linux, aus `.deb` / `.rpm` | `sudo apt remove orcaton` / `sudo dnf remove orcaton`; FFmpeg und die Bibliotheken bleiben, Einstellungen — in `~/.config/orcaton` |
| macOS, aus der `.dmg` | Über → **Befehlszeilen-Tool entfernen** (falls installiert), dann Orcaton aus „Programme“ in den Papierkorb ziehen; die Einstellungen bleiben in `~/Library/Application Support/orcaton` |

Entfernt wird nur Orcaton: das Programm, seine Verknüpfungen und der Befehl `orcaton`. Go, Node.js, Wails
und FFmpeg bleiben (andere Programme können sie nutzen). Mit `-Purge` / `--purge` werden auch die
Einstellungen und der Audio-Cache gelöscht.

## Wenn etwas schiefgeht
| Was Sie sehen | Was zu tun ist |
|---|---|
| `…cannot be loaded because running scripts is disabled` | den Befehl vollständig einfügen, mit `powershell -ExecutionPolicy Bypass -File` am Anfang |
| `winget` wird nicht erkannt (Windows 10) | Microsoft Store → „App-Installer“ → Aktualisieren, dann wiederholen |
| `не найдено: go / node` („nicht gefunden“) oder `… not recognized` nach der Installation | Terminal schließen, ein **neues** im Projektordner öffnen und den Befehl wiederholen (PATH wird aktualisiert) |
| das Fenster öffnet sich weiß / leer (Windows 10) | die WebView2-Engine fehlt: `winget install Microsoft.EdgeWebView2Runtime`, dann wieder die Verknüpfung |
| im Fenster der Hinweis **FFmpeg ist nicht installiert** | im Fenster **FFmpeg installieren** drücken (oder den angezeigten Befehl im Terminal ausführen), dann **Erneut prüfen** |
| macOS: „Apple konnte nicht überprüfen …“, „kann nicht geöffnet werden, da der Entwickler nicht verifiziert werden kann“ | Systemeinstellungen → Datenschutz & Sicherheit → **Dennoch öffnen** (macOS 12–14: Rechtsklick auf Orcaton → Öffnen → Öffnen) — siehe „macOS: fertiges Paket“ |
| macOS: „Orcaton ist beschädigt und kann nicht geöffnet werden“ | die `.dmg` erneut laden und `shasum -a 256` prüfen; stimmt der Wert und die Meldung kommt wieder — Terminal: `xattr -dr com.apple.quarantine /Applications/Orcaton.app` |
| Ubuntu: `libwebkit2gtk-4.1` nicht gefunden | Ubuntu älter als 22.04 wird nicht unterstützt: 22.04 oder neuer ist nötig |
| etwas anderes | Entwicklerwerkzeuge → **Protokoll kopieren** im Orcaton-Fenster und den Text aus dem Terminal — an den Entwickler |

## Ohne Fenster: Terminal
Dieselben Funktionen gibt es als Befehl `orcaton` (wird mit derselben Installation eingerichtet) — für
Skripte, Analysen und Stapelverarbeitung: `orcaton --help`; alle Optionen — [docs/go/usage.md](go/usage.md)
(auf Russisch).
