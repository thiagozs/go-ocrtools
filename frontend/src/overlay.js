import { OCRService } from "../bindings/github.com/thiagozs/ocrtools";

const mon = Number(new URLSearchParams(location.search).get("m") || 0);
const shot = document.getElementById("shot");
const sel = document.getElementById("sel");
const dim = document.getElementById("dim");
let start = null;
let rect = null;

OCRService.GetScreenshot(mon).then((url) => (shot.src = url)).catch(() => OCRService.CancelSelection());

function draw(x1, y1, x2, y2) {
  rect = { x: Math.min(x1, x2), y: Math.min(y1, y2), w: Math.abs(x2 - x1), h: Math.abs(y2 - y1) };
  Object.assign(sel.style, { display: "block", left: rect.x + "px", top: rect.y + "px", width: rect.w + "px", height: rect.h + "px" });
}

addEventListener("mousedown", (e) => {
  if (e.button !== 0) return;
  start = { x: e.clientX, y: e.clientY };
  dim.style.display = "none"; // the selection box provides its own dimming
  draw(e.clientX, e.clientY, e.clientX, e.clientY);
});
addEventListener("mousemove", (e) => start && draw(start.x, start.y, e.clientX, e.clientY));
addEventListener("mouseup", () => {
  if (!start) return;
  start = null;
  if (!rect || rect.w < 4 || rect.h < 4) return OCRService.CancelSelection();
  // Fractions of the window so Go can map to physical pixels regardless of HiDPI scaling.
  OCRService.SubmitSelection(mon, rect.x / innerWidth, rect.y / innerHeight, rect.w / innerWidth, rect.h / innerHeight);
});
addEventListener("keydown", (e) => e.key === "Escape" && OCRService.CancelSelection());
addEventListener("contextmenu", (e) => { e.preventDefault(); OCRService.CancelSelection(); });
