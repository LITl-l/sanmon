import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

export default defineConfig({
  site: "https://litl-l.github.io",
  base: "/sanmon",
  integrations: [
    starlight({
      title: "sanmon (三門)",
      description:
        "Three-gate formal verification stack for AI agent actions.",
      defaultLocale: "en",
      locales: {
        en: { label: "English", lang: "en" },
        ja: { label: "日本語", lang: "ja" },
      },
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/LITl-l/sanmon",
        },
      ],
      sidebar: [
        {
          label: "Getting Started",
          translations: { ja: "はじめに" },
          items: [
            { label: "Introduction", slug: "guides/introduction", translations: { ja: "sanmon とは" } },
            { label: "Quick Start", slug: "guides/quickstart", translations: { ja: "クイックスタート" } },
          ],
        },
        {
          label: "Architecture",
          translations: { ja: "アーキテクチャ" },
          items: [
            { label: "The Three Gates", slug: "guides/three-gates", translations: { ja: "三つの門" } },
            { label: "CUE as Source of Truth", slug: "guides/cue-source", translations: { ja: "CUE による一元管理" } },
            { label: "Domain Policies", slug: "guides/domains", translations: { ja: "ドメインポリシー" } },
          ],
        },
        {
          label: "Reference",
          translations: { ja: "リファレンス" },
          items: [
            { label: "Specification", slug: "reference/spec", translations: { ja: "仕様書" } },
            { label: "Implementation Plan", slug: "reference/plan", translations: { ja: "実装計画" } },
            { label: "CLI & API", slug: "reference/cli", translations: { ja: "CLI・API" } },
          ],
        },
      ],
    }),
  ],
});
