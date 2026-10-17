# catch_ajfsp

Windows-Helfer für appleJuice: übergibt `ajfsp://`- und `web+ajfsp://`-Links direkt an einen lokalen oder entfernten Core, ohne JavaGUI.

## Verwendung

1. EXE in einen beschreibbaren Ordner legen und starten.
2. Core-Host, XML-Port (standardmäßig 9851) und Passwort einstellen. Mit **Core testen** Verbindung prüfen.
3. **Einstellungen speichern** schreibt `catch_ajfsp.ini` neben die EXE. Das Passwort wird als MD5-Hash gespeichert. Ein leeres Passwortfeld behält den gespeicherten Wert; bei neuer Konfiguration gilt das leere Core-Passwort.
4. **Links an catch_ajfsp übergeben** registriert beide Protokolle für den aktuellen Windows-Benutzer. Windows fragt gegebenenfalls nach der Standard-App.
5. Link anklicken oder ins Fenster einfügen und **Link an Core senden** drücken. Das Fenster zeigt Typ, Name, Größe, Hash und Ergebnis.

Der Aufruf `catch_ajfsp.exe "ajfsp://server|server.example|9850/"` öffnet das Fenster und übergibt den Link automatisch. Das Fenster bleibt danach offen. Dateien mit Leerzeichen, Umlauten und `&` im Namen werden korrekt URL-kodiert.

Die INI-Datei des alten Programms wird gelesen. Zum Wechsel zurück auf eine GUI deren Setup oder [fixajfsp](https://github.com/applejuicenetz/fixajfsp) verwenden; eine benutzerspezifische Zuordnung kann eine systemweite Zuordnung überlagern. Die Auswahl unter Windows-Standard-Apps bleibt maßgeblich.

## Sicherheit

Die Core-Verbindung verwendet HTTP. Nur über ein vertrauenswürdiges Netz oder einen SSH-Tunnel verwenden. Auch der gespeicherte Passworthash ermöglicht Zugriff auf den Core: INI-Datei nicht weitergeben.

## Lizenz

MIT, siehe `LICENSE`.
