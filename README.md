# 🛡️ Clipboard Monitor - Portapapeles seguro

Clipboard Monitor es una herramienta **cross-platform** (Windows/Linux) diseñada para proteger información sensible 
(IPs, contraseñas, tokens, API keys) al interactuar con IAs o chats externos. 
Vigila tu portapapeles y reemplaza automáticamente datos sensibles antes de pegarlos.

## 🚀 Compilación

```bash
# Windows
set GOOS=windows && set GOARCH=amd64 && go build -o clipboard_monitor.exe .

# Linux
GOOS=linux GOARCH=amd64 go build -o clipboard_monitor .
```

O usa los scripts en `scripts/`:
- Windows: `scripts\build.bat`
- Linux: `scripts\build.sh`

## 🖥️ TUI (Interfaz en terminal)

Al ejecutar el programa entras en una interfaz TUI moderna con pestañas:

| Pestaña     | Descripción                                    |
|-------------|------------------------------------------------|
| Dashboard   | Estado del monitor, estadísticas, últimas detecciones |
| Rules       | Lista de reglas con navegación y acciones       |
| Stats       | Estadísticas detalladas de reemplazos           |
| History     | Historial de detecciones                        |

### Navegación
- `←` `→` / `Tab`: Cambiar de pestaña
- `1` `2` `3` `4`: Ir directamente a una pestaña
- `↑` `↓`: Navegar reglas (en pestaña Rules)
- `Enter`: Toggle habilitar/deshabilitar regla (Rules, input vacío)
- `d`: Eliminar regla seleccionada (Rules, input vacío)
- `e`: Toggle enable/disable regla (Rules, input vacío)
- `Esc`: Limpiar input
- `Ctrl+C`: Salir

## 📋 Comandos disponibles

Escribe los comandos en la barra inferior de la TUI:

| Comando | Formato | Descripción |
|---------|---------|-------------|
| `add` | `add "buscar" "reemplazo"` | Añade regla de texto exacto |
| `add -regex` | `add -regex "patrón" "reemplazo"` | Añade regla con expresión regular |
| `del` | `del "buscar"` | Elimina una regla |
| `enable` | `enable "buscar"` | Habilita una regla |
| `disable` | `disable "buscar"` | Deshabilita una regla |
| `toggle` | `toggle "buscar"` | Alterna estado de una regla |
| `list` | `list` | Muestra las reglas (cambia a pestaña Rules) |
| `stats` | `stats` | Muestra estadísticas (cambia a pestaña Stats) |
| `history` | `history` | Muestra historial (cambia a pestaña History) |
| `pause` | `pause` | Pausa la vigilancia del portapapeles |
| `resume` | `resume` | Reanuda la vigilancia |
| `dryrun` | `dryrun` | Alterna modo simulación (no modifica portapapeles) |
| `autodetect` | `autodetect` | Alterna detección automática de patrones sensibles |
| `profile` | `profile nombre` | Cambia a un perfil de reglas |
| `scan` | `scan ruta/archivo.txt` | Escanea y sanitiza un archivo |
| `export` | `export [archivo.json]` | Exporta reglas a JSON |
| `import` | `import archivo.json` | Importa reglas desde JSON |
| `clear` | `clear` | Limpia el historial de detecciones |
| `help` | `help` | Muestra la ayuda rápida |
| `quit` | `quit` | Cierra el programa |

## 🧪 Auto-detección de patrones

Al activar `autodetect`, el monitor detecta automáticamente:

- **API Keys** (`api_key=...`, `apikey:...`)
- **Tokens** (`token:...`, `bearer ...`, `jwt ...`)
- **Contraseñas** (`password=...`, `secret:...`)
- **Emails** (`usuario@dominio.com`)
- **Direcciones IP** (`192.168.1.1`)
- **Claves privadas** (`-----BEGIN PRIVATE KEY-----`)

## 📂 Formatos de archivo soportados (scan)

- `.txt`, `.log`, `.md`
- `.json` (formatea y sanitiza)
- `.yaml`, `.yml`
- `.env`
- `.csv`

## 📁 Perfiles de reglas

Puedes tener múltiples perfiles de reglas para diferentes contextos:

```bash
profile trabajo    # Cambia al perfil "trabajo"
profile personal   # Cambia al perfil "personal"
```

## 🔄 Exportar/Importar reglas

```bash
export mis_reglas.json    # Exporta reglas actuales
import mis_reglas.json    # Importa reglas desde archivo
```

## Notas de seguridad

- El programa no envía datos a internet
- Todo el proceso ocurre localmente en RAM
- Al cerrar la terminal, el monitor deja de proteger

## 🐧 Cross-platform

El proyecto compila y funciona en **Windows** y **Linux**:
- Lock de instancia única adaptado a cada plataforma
- Monitoreo de portapapeles con `github.com/atotto/clipboard`
- TUI con bubbletea (compatible con ambas plataformas)

## Autor

Desarrollado con ❤️ por [Angel Lucero](https://www.linkedin.com/in/angellucero/)
