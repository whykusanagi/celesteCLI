#!/usr/bin/env python3
import sys
import time
from PIL import Image
import numpy as np

ESC = "\033["
RESET = ESC + "0m"

def clear_screen():
    # Move cursor home + clear screen
    sys.stdout.write(ESC + "H" + ESC + "2J")
    sys.stdout.flush()

def smart_crop_rgba(frame, alpha_thresh=10):
    """
    Crop away fully-transparent borders so we only scale the actual sprite.
    This avoids wasting width on blank canvas and keeps detail focused.
    """
    frame = frame.convert("RGBA")
    arr = np.array(frame)
    alpha = arr[..., 3]

    mask = alpha > alpha_thresh
    if not mask.any():
        return frame  # nothing to crop; just return

    ys, xs = np.where(mask)
    top, bottom = ys.min(), ys.max()
    left, right = xs.min(), xs.max()

    # tiny padding so she doesn't get clipped
    pad = 2
    left = max(0, left - pad)
    top = max(0, top - pad)
    right = min(frame.width - 1, right + pad)
    bottom = min(frame.height - 1, bottom + pad)

    return frame.crop((left, top, right + 1, bottom + 1))

def frame_to_blocks(frame: Image.Image, width: int = 40, crop=True) -> str:
    """
    Render a frame as colored pixel blocks using '█' characters.
    This version matches the original behavior you liked.
    """
    if crop:
        frame = smart_crop_rgba(frame)

    frame = frame.convert("RGBA")
    w, h = frame.size

    # compensate for tall terminal characters (~2:1)
    new_height = int((h / w) * width * 0.5)
    if new_height < 1:
        new_height = 1

    frame = frame.resize((width, new_height), Image.NEAREST)
    arr = np.array(frame)

    r = arr[..., 0]
    g = arr[..., 1]
    b = arr[..., 2]
    a = arr[..., 3]

    H, W = a.shape
    lines = []

    for y in range(H):
        row = []
        for x in range(W):
            if a[y, x] < 128:
                row.append(" ")
                continue

            rr, gg, bb = int(r[y, x]), int(g[y, x]), int(b[y, x])
            row.append(f"\033[38;2;{rr};{gg};{bb}m█")
        lines.append("".join(row) + RESET)

    return "\n".join(lines)

def load_gif_frames(path: str):
    img = Image.open(path)
    frames = []
    delays = []

    try:
        while True:
            frames.append(img.copy())
            delay_ms = img.info.get("duration", 100)
            delays.append(max(delay_ms / 1000.0, 0.01))
            img.seek(img.tell() + 1)
    except EOFError:
        pass

    return frames, delays

def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <image_or_gif> [width]")
        sys.exit(1)

    path = sys.argv[1]
    width = int(sys.argv[2]) if len(sys.argv) > 2 else 48  # bump this for more detail

    frames, delays = load_gif_frames(path)
    block_frames = [frame_to_blocks(f, width=width, crop=True) for f in frames]

    try:
        while True:
            for art, delay in zip(block_frames, delays):
                clear_screen()
                sys.stdout.write(art + "\n")
                sys.stdout.flush()
                time.sleep(delay)
    except KeyboardInterrupt:
        clear_screen()
        print("bye~")

if __name__ == "__main__":
    main()