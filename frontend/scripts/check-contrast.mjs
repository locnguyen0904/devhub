// Asserts every foreground/background token pair meets WCAG AA.
//
// Values are parsed out of theme.css rather than duplicated here: two copies of
// the palette would drift, and the drift would be invisible until someone
// audited the site by hand.
//
// WCAG 2.1 §1.4.3 — https://www.w3.org/TR/WCAG21/#contrast-minimum

import { readFileSync } from "node:fs";
import path from "node:path";

const THEME = path.resolve(import.meta.dirname, "../src/app/theme.css");

/** Pairs to verify, as [foreground token, background token, minimum ratio]. */
const PAIRS = [
  ["text-primary", "surface", 4.5],
  ["text-muted", "surface", 4.5],
  ["text-subtle", "surface", 4.5],
  ["accent", "surface", 4.5],
  ["accent-fg", "accent", 4.5],
  ["text-primary", "surface-raised", 4.5],
  ["text-muted", "surface-raised", 4.5],
  ["success", "surface-raised", 4.5],
  ["danger", "surface-raised", 4.5],
];

const channels = (hex) => {
  const v = hex.replace("#", "");
  const full =
    v.length === 3
      ? v
          .split("")
          .map((c) => c + c)
          .join("")
      : v;
  return [0, 2, 4].map((i) => parseInt(full.slice(i, i + 2), 16) / 255);
};

const linearize = (c) =>
  c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);

const luminance = (hex) => {
  const [r, g, b] = channels(hex).map(linearize);
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
};

const contrast = (fg, bg) => {
  const [hi, lo] = [luminance(fg), luminance(bg)].sort((a, b) => b - a);
  return (hi + 0.05) / (lo + 0.05);
};

/** Extracts `--token: #hex;` declarations from one CSS rule block. */
function readTokens(css, selector) {
  const block = new RegExp(`${selector}\\s*\\{([^}]*)\\}`).exec(css);
  if (!block) throw new Error(`block ${selector} not found in theme.css`);

  const tokens = {};
  for (const [, name, value] of block[1].matchAll(
    /--([\w-]+):\s*(#[0-9a-fA-F]{3,8})\s*;/g,
  )) {
    tokens[name] = value;
  }
  return tokens;
}

const css = readFileSync(THEME, "utf8");
const modes = {
  light: readTokens(css, ":root"),
  dark: readTokens(css, "\\.dark"),
};

let failed = 0;
for (const [mode, tokens] of Object.entries(modes)) {
  console.log(`\n${mode}`);
  for (const [fg, bg, min] of PAIRS) {
    if (!tokens[fg] || !tokens[bg]) {
      console.error(`  MISSING  ${fg} or ${bg} is not declared in ${mode}`);
      failed++;
      continue;
    }
    const ratio = contrast(tokens[fg], tokens[bg]);
    const ok = ratio >= min;
    if (!ok) failed++;
    console.log(
      `  ${ok ? "pass" : "FAIL"}  ${ratio.toFixed(2).padStart(5)}  (need ${min})  ${fg} on ${bg}`,
    );
  }
}

if (failed > 0) {
  console.error(`\n${failed} colour pair(s) below WCAG AA.`);
  process.exit(1);
}
console.log("\nEvery colour pair meets WCAG AA.");
