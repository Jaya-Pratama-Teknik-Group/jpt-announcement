import { addIcon } from "@iconify/vue";
import iconSet from "./hugeicons-data.json";

export type HugeIconName = keyof typeof iconSet;

let registered = false;

export function registerHugeIcons(): void {
  if (registered) return;
  for (const [name, data] of Object.entries(iconSet)) {
    addIcon(`hugeicons:${name}`, data);
  }
  registered = true;
}
