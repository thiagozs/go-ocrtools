#!/usr/bin/env python3
"""Renders build/appicon.png (1024) and build/trayicon.png (128) with the geometry of appicon.svg.

Usage: python3 build/render_icon.py   (needs Pillow)
"""
from pathlib import Path
from PIL import Image, ImageDraw

SS = 4          # supersampling factor
N = 1024 * SS
OUT = Path(__file__).parent


def s(v):
    return int(round(v * SS))


def gradient(size):
    # diagonal #2f80ff -> #8b3dff
    a, b = (0x2F, 0x80, 0xFF), (0x8B, 0x3D, 0xFF)
    g = Image.linear_gradient("L").resize((size, size)).rotate(45, expand=False)
    # rotate() leaves corners empty; build the diagonal explicitly instead
    px = Image.new("RGB", (size, size))
    d = ImageDraw.Draw(px)
    steps = 2 * size
    for i in range(steps):
        t = i / (steps - 1)
        c = tuple(int(a[k] + (b[k] - a[k]) * t) for k in range(3))
        d.line([(i, 0), (0, i)], fill=c, width=2)
    return px


def round_line(d, pts, w):
    d.line(pts, fill="white", width=w, joint="curve")
    r = w // 2
    for x, y in (pts[0], pts[-1]):
        d.ellipse([x - r, y - r, x + r, y + r], fill="white")


def main():
    img = Image.new("RGBA", (N, N), (0, 0, 0, 0))
    mask = Image.new("L", (N, N), 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, N - 1, N - 1], radius=s(228), fill=255)
    bg = gradient(N)
    img.paste(bg, (0, 0), mask)

    d = ImageDraw.Draw(img)
    w = s(58)
    for pts in (
        [(196, 346), (196, 196), (346, 196)],
        [(678, 196), (828, 196), (828, 346)],
        [(828, 678), (828, 828), (678, 828)],
        [(346, 828), (196, 828), (196, 678)],
    ):
        round_line(d, [(s(x), s(y)) for x, y in pts], w)
    d.rounded_rectangle([s(352), s(366), s(672), s(442)], radius=s(24), fill="white")
    d.rounded_rectangle([s(474), s(366), s(550), s(666)], radius=s(24), fill="white")

    full = img.resize((1024, 1024), Image.LANCZOS)
    full.save(OUT / "appicon.png")
    full.resize((128, 128), Image.LANCZOS).save(OUT / "trayicon.png")


if __name__ == "__main__":
    main()
