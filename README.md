# clipmonitor

clipmonitor es una herramienta **cross-platform** (Windows/Linux) diseñada para proteger información sensible
(IPs, contraseñas, tokens, API keys) al interactuar con IAs o chats externos.
Vigila tu portapapeles y reemplaza automáticamente datos sensibles antes de pegarlos.

## 🚀 Compilación

```bash
# Windows
set GOOS=windows && set GOARCH=amd64 && go build -o clipboard_monitor.exe .

# Linux
GOOS=linux GOARCH=amd64 go build -o clipboard_monitor .
```

## TUI (Interfaz en terminal)

La TUI es **responsive**: se adapta al tamaño de la terminal.

Dos paneles lado a lado sin tabs — toda la información visible de una vez:

| Panel izquierdo                                             | Panel derecho                             |
|-------------------------------------------------------------|-------------------------------------------|
| Stats (reemplazos, reglas activas, historial, top reglas)   | Active Rules (lista de reglas con scroll) |
| Recent Activity (últimas detecciones)                       |                                           |

### Navegación

- `↑` `↓`: Navegar reglas (en pestaña Rules)
- `Enter`: Toggle habilitar/deshabilitar regla (Rules, input vacío)
- `Esc`: Limpiar input
- `Ctrl+C`: Salir

### Comandos

Escribe `/` en la barra inferior para abrir la paleta de comandos. Filtra escribiendo parte del nombre y navega con `↑`/`↓`, confirma con `Enter`.

```
┌────────────────────────────────────────────┐
│   Commands                                 │
│ ────────────────────────────────────────── │
│ ▸ add       Add a replacement rule         │
│   del       Delete a rule                  │
│   enable    Enable a rule                  │
│   disable   Disable a rule                 │
│   toggle    Toggle a rule on/off           │
│   pause     Pause clipboard monitoring     │
│   resume    Resume clipboard monitoring    │
│   dryrun    Toggle dry-run mode            │
│   notify    Toggle desktop notifications   │
│   clear     Clear detection history        │
│   scan      Scan a file for sensitive data │
│   export    Export rules to JSON file      │
│   import    Import rules from JSON file    │
│   help      Show available commands        │
│   quit      Exit the application           │
└────────────────────────────────────────────┘
```

## Comandos disponibles

| Comando      | Formato                           | Descripción                                        |
|--------------|-----------------------------------|----------------------------------------------------|
| `add`        | `add "buscar" "reemplazo"`        | Añade regla de texto exacto                        |
| `add -regex` | `add -regex "patrón" "reemplazo"` | Añade regla con expresión regular                  |
| `del`        | `del "buscar"`                    | Elimina una regla                                  |
| `enable`     | `enable "buscar"`                 | Habilita una regla                                 |
| `disable`    | `disable "buscar"`                | Deshabilita una regla                              |
| `toggle`     | `toggle "buscar"`                 | Alterna estado de una regla                        |
| `pause`      | `pause`                           | Pausa la vigilancia del portapapeles               |
| `resume`     | `resume`                          | Reanuda la vigilancia                              |
| `dryrun`     | `dryrun`                          | Alterna modo simulación (no modifica portapapeles) |
| `notify`     | `notify`                          | Alterna notificaciones de escritorio               |
| `clear`      | `clear`                           | Limpia el historial de detecciones                 |
| `scan`       | `scan ruta/archivo.txt`           | Escanea y sanitiza un archivo                      |
| `export`     | `export [archivo.json]`           | Exporta reglas a JSON                              |
| `import`     | `import archivo.json`             | Importa reglas desde JSON                          |
| `help`       | `help`                            | Muestra la ayuda rápida                            |
| `quit`       | `quit`                            | Cierra el programa                                 |

## Formatos de archivo soportados (scan)

- `.txt`, `.log`, `.md`
- `.json` (formatea y sanitiza)
- `.yaml`, `.yml`
- `.env`
- `.csv`

## Exportar/Importar reglas

```bash
export mis_reglas.json    # Exporta reglas actuales
import mis_reglas.json    # Importa reglas desde archivo
```

## Auto-detección de patrones

Al activar `autodetect` (vía config), el monitor detecta automáticamente:

- **API Keys** (`api_key=...`, `apikey:...`)
- **Tokens** (`token:...`, `bearer ...`, `jwt ...`)
- **Contraseñas** (`password=...`, `secret:...`)
- **Emails** (`usuario@dominio.com`)
- **Direcciones IP** (`192.168.1.1`)
- **Claves privadas** (`-----BEGIN PRIVATE KEY-----`)

## Perfiles de reglas

Puedes tener múltiples perfiles de reglas para diferentes contextos editando `replacements.json`:

```json
{
  "profiles": {
    "trabajo": { },
    "personal": { }
  }
}
```

## Notas de seguridad

- El programa no envía datos a internet
- Todo el proceso ocurre localmente en RAM
- Al cerrar la terminal, el monitor deja de proteger

## Cross-platform

El proyecto compila y funciona en **Windows** y **Linux**:
- Lock de instancia única adaptado a cada plataforma
- Monitoreo de portapapeles con `github.com/atotto/clipboard`
- TUI con bubbletea (compatible con ambas plataformas)

## Autor

Desarrollado con ❤️ por [Angel Lucero](https://www.linkedin.com/in/angellucero/)
