import { OCRService } from "../bindings/github.com/thiagozs/ocrtools";
import { Settings } from "../bindings/github.com/thiagozs/ocrtools/internal/config/models.js";

const $ = (id) => document.getElementById(id);

async function refreshEngine() {
  const st = await OCRService.GetTesseractStatus();
  $("installBox").hidden = st.installed;
  $("langBox").hidden = !st.installed;
  if (st.installed) {
    $("engineState").textContent = `✓ ${st.version} — idiomas: ${st.languages.join(", ") || "nenhum"}`;
  } else {
    $("engineState").textContent = "⚠ Tesseract não encontrado";
    $("installCmd").textContent = st.installCommand;
    $("installNote").textContent = st.note;
    $("install").hidden = !st.canInstall;
  }
  $("langs").replaceChildren(...st.languages.map((l) => Object.assign(document.createElement("option"), { value: l })));
  return st;
}

let confirmUninstall = false;

async function refreshInstall() {
  const st = await OCRService.GetInstallStatus();
  $("installPath").textContent = "Executando de: " + st.exe;
  $("installActions").hidden = !st.canInstall && !st.installed;
  if (st.installed) {
    $("installState").textContent = st.scope === "system" ? "✓ Instalado para todos os usuários" : "✓ Instalado para o seu usuário";
  } else {
    $("installState").textContent = "Executável avulso (não instalado)";
  }
  $("installMsg").textContent = st.canInstall ? "" : st.note;
  $("instUser").hidden = !st.canInstall || (st.installed && st.scope === "user");
  $("instUser").textContent = "Instalar para mim (" + st.userDir + ")";
  $("instSystem").hidden = !st.canInstall || !st.canSystem || (st.installed && st.scope === "system");
  $("instSystem").textContent = "Instalar para todos (" + st.systemDir + ")";
  $("uninst").hidden = !st.installed;
  $("uninst").textContent = confirmUninstall ? "Confirmar desinstalação" : "Desinstalar";
}

async function doInstall(scope) {
  $("installMsg").textContent = "Instalando…" + (scope === "system" ? " (pode pedir senha de administrador)" : "");
  try {
    await OCRService.InstallApp(scope); // the app reopens from the new location and this window closes
    $("installMsg").textContent = "Instalado. Reabrindo…";
  } catch (e) {
    $("installMsg").textContent = "Falhou: " + e;
  }
}
$("instUser").addEventListener("click", () => doInstall("user"));
$("instSystem").addEventListener("click", () => doInstall("system"));
$("uninst").addEventListener("click", async () => {
  if (!confirmUninstall) { confirmUninstall = true; await refreshInstall(); return; }
  try {
    await OCRService.UninstallApp();
    $("installMsg").textContent = "Desinstalado. Suas configurações e idiomas foram mantidos.";
  } catch (e) {
    $("installMsg").textContent = "Falhou: " + e;
    confirmUninstall = false;
  }
});

async function load() {
  const s = await OCRService.GetSettings();
  $("lang").value = s.language;
  $("autoCopy").checked = s.autoCopy;
  $("showResult").checked = s.showResult;
  $("tess").value = s.tesseractPath;
  $("hotkey").value = s.hotkey;
  $("autostart").checked = s.autostart;
  await refreshEngine();
  await refreshInstall();
}

$("install").addEventListener("click", async () => {
  $("install").disabled = true;
  $("installStatus").textContent = "Instalando… (pode pedir sua senha)";
  try {
    await OCRService.InstallTesseract();
    $("installStatus").textContent = "";
  } catch (e) {
    $("installStatus").textContent = "Falhou: " + e;
  }
  $("install").disabled = false;
  await refreshEngine();
});

$("download").addEventListener("click", async () => {
  $("download").disabled = true;
  $("dlStatus").textContent = "Baixando…";
  try {
    await OCRService.DownloadLanguage($("dlLang").value.trim());
    $("dlStatus").textContent = "Pronto ✓";
  } catch (e) {
    $("dlStatus").textContent = "Falhou: " + e;
  }
  $("download").disabled = false;
  await refreshEngine();
});

$("save").addEventListener("click", async () => {
  try {
    await OCRService.SaveSettings(new Settings({
      language: $("lang").value.trim(),
      autoCopy: $("autoCopy").checked,
      showResult: $("showResult").checked,
      hotkey: $("hotkey").value,
      autostart: $("autostart").checked,
      tesseractPath: $("tess").value.trim(),
    }));
    $("status").textContent = "Salvo ✓";
  } catch (e) {
    $("status").textContent = "Erro: " + e;
  }
  setTimeout(() => ($("status").textContent = ""), 2500);
});

// Capture a key combination as a Wails accelerator ("CmdOrCtrl+Alt+O").
const MODS = ["Control", "Shift", "Alt", "Meta"];
const NAMED = { " ": "Space", ArrowUp: "Up", ArrowDown: "Down", ArrowLeft: "Left", ArrowRight: "Right" };
$("hotkey").addEventListener("keydown", (e) => {
  e.preventDefault();
  if (MODS.includes(e.key)) return;
  if (e.key === "Backspace" || e.key === "Delete") { $("hotkey").value = ""; return; }
  const parts = [];
  if (e.ctrlKey || e.metaKey) parts.push("CmdOrCtrl");
  if (e.altKey) parts.push("Alt");
  if (e.shiftKey) parts.push("Shift");
  if (!parts.length) { $("status").textContent = "Use pelo menos um modificador (Ctrl, Alt, Shift)."; return; }
  const key = NAMED[e.key] ?? (e.key.length === 1 ? e.key.toUpperCase() : e.key);
  $("hotkey").value = [...parts, key].join("+");
});

load();
