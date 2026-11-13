import { OCRService } from "../bindings/github.com/thiagozs/ocrtools";
import { Window } from "@wailsio/runtime";

const $ = (id) => document.getElementById(id);

OCRService.LastResult().then((r) => {
  $("err").textContent = r.error;
  $("text").value = r.text;
  if (r.error) $("text").style.display = "none";
  else if (!r.text) $("err").textContent = "Nenhum texto foi reconhecido na área selecionada.";
});

$("copy").addEventListener("click", async () => {
  const ok = await OCRService.CopyText($("text").value);
  $("status").textContent = ok ? "Copiado ✓" : "Falha ao copiar";
  setTimeout(() => ($("status").textContent = ""), 2000);
});
$("close").addEventListener("click", () => Window.Close());
addEventListener("keydown", (e) => e.key === "Escape" && Window.Close());
