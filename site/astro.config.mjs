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
          items: [
            { label: "Introduction", slug: "guides/introduction" },
            { label: "Quick Start", slug: "guides/quickstart" },
          ],
        },
        {
          label: "Architecture",
          items: [
            { label: "The Three Gates", slug: "guides/three-gates" },
            { label: "CUE as Source of Truth", slug: "guides/cue-source" },
            { label: "Domain Policies", slug: "guides/domains" },
          ],
        },
        {
          label: "Reference",
          items: [
            { label: "Specification", slug: "reference/spec" },
            { label: "Implementation Plan", slug: "reference/plan" },
            { label: "CLI & API", slug: "reference/cli" },
          ],
        },
      ],
    }),
  ],
});
