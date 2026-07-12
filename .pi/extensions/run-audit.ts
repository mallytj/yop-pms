import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import type { AutocompleteItem } from "@earendil-works/pi-tui";

const AGENTS = [
  { value: "full", label: "full — All advisory agents (CTO, Boutique, Compliancy, UX)" },
  { value: "cto", label: "cto — CTO: scale, multi-tenancy, reliability" },
  { value: "boutique", label: "boutique — Boutique Director: small team, personalization, reputation" },
  { value: "compliancy", label: "compliancy — Compliancy: GDPR, PII, data protection" },
  { value: "ux", label: "ux — UX Expert: power-user speed, keyboard, accessibility" },
];

export default function (pi: ExtensionAPI) {
  pi.registerCommand("run-audit", {
    description: "Run role-specific audit agents against a feature or design",
    getArgumentCompletions: (prefix: string): AutocompleteItem[] | null => {
      const parts = prefix.split(" ");
      // After first token (agent selector), stop suggesting
      if (parts.length > 1 && parts[0] && AGENTS.some((a) => parts[0].startsWith(a.value))) {
        return null; // Let user type freeform context
      }
      const items = AGENTS.filter((a) => a.value.startsWith(parts[0] || ""));
      return items.length > 0 ? items : null;
    },
    handler: async (args, ctx) => {
      if (!args?.trim()) {
        ctx.ui.notify("Usage: /run-audit <full|cto|boutique|compliancy|ux> \"context\"", "warning");
        return;
      }
      // Delegate to the run-audit skill. Command runs while idle — no delivery mode needed.
      pi.sendUserMessage(`run-audit ${args}`);
    },
  });
}
