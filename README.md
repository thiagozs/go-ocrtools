# OCR Tools

🇺🇸 [English version](README-us.md)

Utilitário de bandeja do sistema multiplataforma (Linux, Windows, macOS)
feito em Go com [Wails v3](https://v3.wails.io/). Capture qualquer região da
tela, extraia o texto com [Tesseract](https://github.com/tesseract-ocr/tesseract)
e copie para a área de transferência.

> Status: Linux (GNOME/Wayland) testado de ponta a ponta. O código para Windows
> e macOS compila (`go vet` com `GOOS`), mas nunca foi executado.

## Funcionalidades

- **Captura de região** – um overlay em tela cheia abre em cada monitor; arraste
  para selecionar a área. Funciona com vários monitores, HiDPI e Wayland (via
  XDG Desktop Portal). O recorte é feito a partir do screenshot original, então
  a resolução é preservada.
- **OCR** – Tesseract via CLI (sem cgo), com pré-processamento: escala de cinza,
  ampliação de recortes pequenos e inversão automática de fundos escuros.
- **Resultado** – o texto é copiado automaticamente (opcional) e exibido em uma
  janela.
- **Bandeja do sistema** – menu com *Capturar texto*, *Configurações* e *Sair*;
  clicar no ícone inicia uma captura.
- **Atalho global** – padrão `CmdOrCtrl+Alt+O`, configurável. No Wayland usa o
  portal GlobalShortcuts.
- **Configuração do Tesseract** – detecta o binário, oferece instalação guiada
  ou automática e baixa os idiomas (`tessdata_fast`) em
  `~/.config/ocrtools/tessdata`.
- **Iniciar com o sistema** – opção por sistema operacional nas Configurações.
- **Autoinstalação** – instala o app para o usuário atual ou para todos,
  seguindo as convenções de cada SO (veja abaixo), e desinstala pelas
  Configurações.
- **Identidade correta no desktop** – app id `io.github.thiagozs.ocrtools`, para
  que o dock/barra de tarefas mostre o nome e o ícone certos.

## Requisitos

- Go 1.25+ (veja `go.mod`), Node.js + npm (frontend), CLI `wails3`.
- **Tesseract** em tempo de execução (o app ajuda a instalar).
- Build no Linux: `libgtk-4-dev`, `libwebkitgtk-6.0-dev`,
  `libayatana-appindicator3-dev`. No Wayland: `xdg-desktop-portal` e o backend
  de portal do seu desktop.

## Build e execução

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
wails3 dev                  # desenvolvimento, hot reload
wails3 build               # binário de produção em bin/
wails3 package             # instaladores/pacotes para o SO atual
go test ./...
```

Depois de alterar métodos exportados do serviço, regenere os bindings:

```sh
wails3 generate bindings -clean=true
```

## Locais de instalação

| SO      | Usuário                               | Todos os usuários             |
|---------|---------------------------------------|-------------------------------|
| Linux   | `~/.local/opt/ocrtools` (+ link em `~/.local/bin`, `.desktop`) | `/opt/ocrtools` (via `pkexec`) |
| Windows | `%LOCALAPPDATA%\Programs\OCR Tools`   | `Program Files` (elevado)     |
| macOS   | `~/Applications` (bundle `.app`)      | `/Applications` (senha de admin) |

As configurações ficam no diretório de config do usuário, em
`ocrtools/settings.json`.

## Estrutura do projeto

```
main.go, service.go     setup do app, bandeja, OCRService exposto ao frontend
monitors*.go            geometria por monitor (GDK no Linux, Wails nos demais)
relaunch_*.go           reinício após a autoinstalação
internal/capture        screenshot (X11/nativo, portal no Wayland)
internal/ocr            engine Tesseract, plano de instalação, download de idiomas
internal/hotkey         backends de atalho global (Wails / portal Wayland)
internal/launcher       entrada .desktop e app id (Linux)
internal/install        instalar/desinstalar por SO
internal/config         armazenamento das configurações
frontend/               páginas vite: configurações (index), overlay, resultado
build/                  ícones (appicon.svg, render_icon.py) e empacotamento
```

## Limitações conhecidas

- A seleção ainda não pode atravessar mais de um monitor.
- Instalação para todos, desinstalação e o botão "Instalar Tesseract" exigem
  privilégios elevados e não foram testados de ponta a ponta.
- Windows e macOS não foram executados.

Veja o [STEPS.md](STEPS.md) para o histórico de desenvolvimento.

## Licença

MIT. Veja o arquivo [LICENSE](LICENSE).

## Autor

2026, Thiago Zilli Sarmento :heart:
