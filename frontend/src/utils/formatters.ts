import { HARI_INDONESIA, type Schedule } from "../types/models";

export function formatDays(days: number[]): string {
  if (!days || days.length === 0 || days.length === 7) return "Setiap hari";
  return HARI_INDONESIA.filter((h) => days.includes(h.value))
    .map((h) => h.short)
    .join(", ");
}

export function formatTimes(item: Schedule): string[] {
  if (item.timesOfDay && item.timesOfDay.length > 0) return item.timesOfDay;
  return item.timeOfDay ? [item.timeOfDay] : [];
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}
